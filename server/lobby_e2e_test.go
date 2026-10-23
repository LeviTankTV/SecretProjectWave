package main

// lobby_e2e_test.go - сквозные тесты лобби (пати): регистрация → создание
// пати → вход по коду → выбор зоны → старт → WS-подключения → отключения →
// чистка призраков. Нужна живая MongoDB на 127.0.0.1:27017; без неё E2E-тесты
// пропускаются (t.Skip), чисто серверные — работают всегда.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ---- helpers ----

func ensureTestDB(t *testing.T) {
	t.Helper()
	if usersCollection != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cl, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://127.0.0.1:27017"))
	if err != nil {
		t.Skip("MongoDB недоступна, E2E пропущен:", err)
	}
	defer cl.Disconnect(context.Background())
	if err := cl.Ping(ctx, nil); err != nil {
		t.Skip("MongoDB недоступна, E2E пропущен:", err)
	}
	initDB()
}

func newTestServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/register", handleRegister)
	mux.HandleFunc("/api/login", handleLogin)
	mux.HandleFunc("/api/lobby-create", handleLobbyCreate)
	mux.HandleFunc("/api/lobby-join", handleLobbyJoin)
	mux.HandleFunc("/api/lobby-list", handleLobbyList)
	mux.HandleFunc("/api/lobby-leave", handleLobbyLeave)
	mux.HandleFunc("/api/lobby-info", handleLobbyInfo)
	mux.HandleFunc("/api/lobby-start", handleLobbyStart)
	mux.HandleFunc("/api/lobby-zone", handleLobbyZone)
	mux.HandleFunc("/ws", handleWebSocket)
	return httptest.NewServer(mux)
}

// lobbyListOf безопасно достаёт список лобби из ответа.
func lobbyListOf(t *testing.T, res map[string]interface{}) []interface{} {
	t.Helper()
	if l, ok := res["list"].([]interface{}); ok {
		return l
	}
	return nil
}

func postJSON(t *testing.T, base, path string, body map[string]interface{}) map[string]interface{} {
	t.Helper()
	raw, _ := json.Marshal(body)
	resp, err := http.Post(base+path, "application/json", bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	defer resp.Body.Close()
	var out map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("POST %s: bad json: %v", path, err)
	}
	return out
}

// registerTestUser — уникальный аккаунт на каждый запуск тестов (в Mongo
// уникальный индекс ника, повторный запуск не должен падать).
func registerTestUser(t *testing.T, base, tag string) (nick, token string) {
	t.Helper()
	nick = fmt.Sprintf("e2e%s%d", tag, time.Now().UnixNano()%1000000000)
	res := postJSON(t, base, "/api/register", map[string]interface{}{"nick": nick, "password": "pass1234"})
	if res["ok"] != true {
		t.Fatalf("register %s failed: %v", nick, res["error"])
	}
	return nick, res["token"].(string)
}

func dialWS(t *testing.T, base, token, code string) *websocket.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(base, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	hello := map[string]interface{}{"type": "hello", "token": token, "color": 0x54c468, "code": code}
	if err := conn.WriteJSON(hello); err != nil {
		t.Fatalf("ws hello: %v", err)
	}
	return conn
}

// wsReadUntil читает кадры до нужного type (сервер шлёт state 20 раз/с).
func wsReadUntil(t *testing.T, conn *websocket.Conn, wantType string, timeout time.Duration) map[string]interface{} {
	t.Helper()
	conn.SetReadDeadline(time.Now().Add(timeout))
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("ws read (жду %s): %v", wantType, err)
		}
		var m map[string]interface{}
		if json.Unmarshal(raw, &m) != nil {
			continue
		}
		if m["type"] == wantType {
			return m
		}
		if m["type"] == "error" {
			t.Fatalf("ws получил ошибку вместо %s: %v", wantType, m["message"])
		}
	}
}

func waitFor(t *testing.T, timeout time.Duration, what string, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("не дождались: %s", what)
}

func roomPlayerCount(t *testing.T, code string) int {
	t.Helper()
	lobbyMu.Lock()
	l := lobbies[code]
	lobbyMu.Unlock()
	if l == nil || l.Room == nil {
		return -1
	}
	l.Room.mu.Lock()
	defer l.Room.mu.Unlock()
	return len(l.Room.players)
}

// ---- Тесты ----

// TestLobbyE2EFlow — полный путь: регистрация двух игроков, создание публичного
// пати, вход по коду, права хоста, старт матча, WS обоих, отключения.
func TestLobbyE2EFlow(t *testing.T) {
	ensureTestDB(t)
	srv := newTestServer()
	defer srv.Close()

	_, hostTok := registerTestUser(t, srv.URL, "H")
	_, guestTok := registerTestUser(t, srv.URL, "G")

	// Создание публичного пати.
	res := postJSON(t, srv.URL, "/api/lobby-create", map[string]interface{}{"token": hostTok, "public": true})
	if res["ok"] != true {
		t.Fatalf("lobby-create: %v", res["error"])
	}
	code, _ := res["code"].(string)
	if len(code) != 6 {
		t.Fatalf("код лобби должен быть 6 цифр, got %q", code)
	}

	// Публичное пати видно в списке.
	list := lobbyListOf(t, postJSON(t, srv.URL, "/api/lobby-list", nil))
	found := false
	for _, it := range list {
		if m, ok := it.(map[string]interface{}); ok && m["code"] == code {
			found = true
		}
	}
	if !found {
		t.Fatalf("созданное публичное пати %s не видно в списке", code)
	}

	// Вход по коду; неверный код отклоняется.
	bad := postJSON(t, srv.URL, "/api/lobby-join", map[string]interface{}{"token": guestTok, "code": "000001"})
	if bad["ok"] == true {
		t.Fatalf("вход по несуществующему коду должен отклоняться")
	}
	join := postJSON(t, srv.URL, "/api/lobby-join", map[string]interface{}{"token": guestTok, "code": code})
	if join["ok"] != true {
		t.Fatalf("lobby-join: %v", join["error"])
	}
	if players, ok := join["players"].([]interface{}); !ok || len(players) != 2 {
		t.Fatalf("после join должно быть 2 игрока, got %v", join["players"])
	}

	// lobby-info для гостя: не хост, есть данные превью.
	info := postJSON(t, srv.URL, "/api/lobby-info", map[string]interface{}{"token": guestTok})
	if info["ok"] != true || info["inLobby"] != true {
		t.Fatalf("lobby-info: %v", info)
	}
	if info["isHost"] == true {
		t.Fatalf("гость не должен быть хостом")
	}
	if pd, ok := info["playersData"].([]interface{}); !ok || len(pd) != 2 {
		t.Fatalf("playersData должно быть 2, got %v", info["playersData"])
	}

	// Зона: гость не может, хост может.
	zg := postJSON(t, srv.URL, "/api/lobby-zone", map[string]interface{}{"token": guestTok, "zone": float64(3)})
	if zg["ok"] == true {
		t.Fatalf("гость не должен менять зону")
	}
	zh := postJSON(t, srv.URL, "/api/lobby-zone", map[string]interface{}{"token": hostTok, "zone": float64(3)})
	if zh["ok"] != true || zh["zoneName"] != "Square" {
		t.Fatalf("хост меняет зону: %v", zh)
	}

	// Старт: гость не может, хост может.
	sg := postJSON(t, srv.URL, "/api/lobby-start", map[string]interface{}{"token": guestTok})
	if sg["ok"] == true {
		t.Fatalf("гость не должен стартовать матч")
	}

	// WS обоих игроков до старта (как клиенты, ждущие в арене).
	wsHost := dialWS(t, srv.URL, hostTok, code)
	defer wsHost.Close()
	wsReadUntil(t, wsHost, "joined", 5*time.Second)
	wsReadUntil(t, wsHost, "world", 5*time.Second)

	wsGuest := dialWS(t, srv.URL, guestTok, code)
	defer wsGuest.Close()
	wsReadUntil(t, wsGuest, "joined", 5*time.Second)
	wsReadUntil(t, wsGuest, "world", 5*time.Second)

	// Старт матча: оба получают wave/pending.
	sh := postJSON(t, srv.URL, "/api/lobby-start", map[string]interface{}{"token": hostTok})
	if sh["ok"] != true || sh["started"] != true {
		t.Fatalf("lobby-start: %v", sh)
	}
	for _, conn := range []*websocket.Conn{wsHost, wsGuest} {
		m := wsReadUntil(t, conn, "wave", 5*time.Second)
		if m["state"] != "pending" {
			t.Fatalf("первое wave-сообщение должно быть pending, got %v", m["state"])
		}
	}

	// Идущий матч скрыт из списка поиска.
	for _, it := range lobbyListOf(t, postJSON(t, srv.URL, "/api/lobby-list", nil)) {
		if m, ok := it.(map[string]interface{}); ok && m["code"] == code {
			t.Fatalf("идущий матч не должен быть в списке поиска")
		}
	}
	// Но lobby-info показывает started.
	info2 := postJSON(t, srv.URL, "/api/lobby-info", map[string]interface{}{"token": guestTok})
	if info2["started"] != true {
		t.Fatalf("lobby-info должно показывать started=true, got %v", info2)
	}

	// Гость рвёт соединение (закрыл вкладку): игрок уходит из комнаты,
	// но остаётся в пати (может переподключиться).
	wsGuest.Close()
	waitFor(t, 5*time.Second, "гость исчез из комнаты", func() bool { return roomPlayerCount(t, code) == 1 })
	info3 := postJSON(t, srv.URL, "/api/lobby-info", map[string]interface{}{"token": guestTok})
	if info3["inLobby"] != true {
		t.Fatalf("гость после обрыва должен оставаться в пати (реконнект)")
	}

	// Хост выходит в меню через WS (кнопка Exit): тоже остаётся в пати.
	if err := wsHost.WriteJSON(map[string]interface{}{"type": "menu"}); err != nil {
		t.Fatalf("ws menu: %v", err)
	}
	waitFor(t, 5*time.Second, "хост вышел из комнаты в меню", func() bool { return roomPlayerCount(t, code) == 0 })

	// Гость (вкладка «оборвана», но он в пати) выходит через API,
	// затем хост — после выхода последнего пати удаляется.
	lvG := postJSON(t, srv.URL, "/api/lobby-leave", map[string]interface{}{"token": guestTok})
	if lvG["ok"] != true {
		t.Fatalf("lobby-leave guest: %v", lvG)
	}
	lv := postJSON(t, srv.URL, "/api/lobby-leave", map[string]interface{}{"token": hostTok})
	if lv["ok"] != true {
		t.Fatalf("lobby-leave: %v", lv)
	}
	waitFor(t, 3*time.Second, "пати удалилось после выхода последнего", func() bool {
		lobbyMu.Lock()
		defer lobbyMu.Unlock()
		_, ok := lobbies[code]
		return !ok
	})
	info4 := postJSON(t, srv.URL, "/api/lobby-info", map[string]interface{}{"token": guestTok})
	if info4["inLobby"] != false {
		t.Fatalf("гость должен быть выкинут из удалённого пати, got %v", info4)
	}
}

// TestLobbyReaperGhosts — «призраки» (вкладка закрыта без leave) вычищаются:
// одиночный хост-призрак, а также призрак-гость при живом хосте.
func TestLobbyReaperGhosts(t *testing.T) {
	ensureTestDB(t)
	srv := newTestServer()
	defer srv.Close()

	oldKick, oldMatch := lobbyIdleKick, matchIdleClose
	lobbyIdleKick, matchIdleClose = 80*time.Millisecond, 80*time.Millisecond
	defer func() { lobbyIdleKick, matchIdleClose = oldKick, oldMatch }()

	// 1) Хост закрыл вкладку: пати удаляется целиком.
	nickA, tokA := registerTestUser(t, srv.URL, "R")
	res := postJSON(t, srv.URL, "/api/lobby-create", map[string]interface{}{"token": tokA, "public": true})
	code, _ := res["code"].(string)
	time.Sleep(200 * time.Millisecond) // тише clock jitter: активность давно не обновлялась
	reapIdleLobbies()
	if l := lobbyForNick(nickA); l != nil {
		t.Fatalf("пати хоста-призрака должно быть удалено reaper'ом")
	}

	// 2) Гость-призрак при живом хосте: гость выкинут, хост остался, пати живо.
	nickB, tokB := registerTestUser(t, srv.URL, "S")
	res2 := postJSON(t, srv.URL, "/api/lobby-create", map[string]interface{}{"token": tokB, "public": true})
	code2, _ := res2["code"].(string)
	nickC, tokC := registerTestUser(t, srv.URL, "T")
	postJSON(t, srv.URL, "/api/lobby-join", map[string]interface{}{"token": tokC, "code": code2})

	time.Sleep(200 * time.Millisecond)
	bumpMemberSeen(nickB) // хост «жив» — его клиент опрашивает lobby-info
	reapIdleLobbies()

	l := lobbyForNick(nickB)
	if l == nil {
		t.Fatalf("пати живого хоста не должно удаляться")
	}
	if len(l.Members) != 1 || l.Members[nickC] {
		t.Fatalf("гость-призрак должен быть удалён, члены: %v", l.Members)
	}
	if l.Host != nickB {
		t.Fatalf("хост должен остаться собой, got %q", l.Host)
	}
	leaveLobby(nickB)
	_ = code
}

// TestLobbyReaperDuringMatch — все отвалились от идущего матча: после
// matchIdleClose пати закрывается, WS-коннекты кикнуты Room.close().
func TestLobbyReaperDuringMatch(t *testing.T) {
	ensureTestDB(t)
	srv := newTestServer()
	defer srv.Close()

	oldKick, oldMatch := lobbyIdleKick, matchIdleClose
	lobbyIdleKick, matchIdleClose = 80*time.Millisecond, 80*time.Millisecond
	defer func() { lobbyIdleKick, matchIdleClose = oldKick, oldMatch }()

	nick, tok := registerTestUser(t, srv.URL, "M")
	res := postJSON(t, srv.URL, "/api/lobby-create", map[string]interface{}{"token": tok, "public": true})
	code, _ := res["code"].(string)

	ws := dialWS(t, srv.URL, tok, code)
	defer ws.Close()
	wsReadUntil(t, ws, "joined", 5*time.Second)

	postJSON(t, srv.URL, "/api/lobby-start", map[string]interface{}{"token": tok})
	// В идущем матче inactive-участник с ЖИВЫМ WS не вычищается.
	time.Sleep(200 * time.Millisecond)
	reapIdleLobbies()
	if l := lobbyForNick(nick); l == nil {
		t.Fatalf("игрок с живым WS в матче не должен вычищаться reaper'ом")
	}

	// Обрыв: молчание дольше matchIdleClose → вычистка + кик коннекта.
	ws.Close()
	time.Sleep(200 * time.Millisecond)
	reapIdleLobbies()
	if l := lobbyForNick(nick); l != nil {
		t.Fatalf("матч без единого живого соединения должен закрываться reaper'ом")
	}
}

// TestNicknameTakeover — «молчащее» соединение с ником перехватывается новым
// входом, живое — защищено («already in use»).
func TestNicknameTakeover(t *testing.T) {
	ensureTestDB(t)
	srv := newTestServer()
	defer srv.Close()

	nick, tok := registerTestUser(t, srv.URL, "K")
	res := postJSON(t, srv.URL, "/api/lobby-create", map[string]interface{}{"token": tok, "public": false})
	code, _ := res["code"].(string)

	ws1 := dialWS(t, srv.URL, tok, code)
	defer ws1.Close()
	wsReadUntil(t, ws1, "joined", 5*time.Second)

	// Живое соединение: повторный вход отклоняется.
	ws2, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+"/ws", nil)
	if err != nil {
		t.Fatalf("dial ws2: %v", err)
	}
	defer ws2.Close()
	ws2.WriteJSON(map[string]interface{}{"type": "hello", "token": tok, "code": code})
	ws2.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, raw, err := ws2.ReadMessage()
	if err != nil {
		t.Fatalf("ожидали сообщение об ошибке, got %v", err)
	}
	var m map[string]interface{}
	json.Unmarshal(raw, &m)
	if m["type"] != "error" || !strings.Contains(fmt.Sprint(m["message"]), "already in use") {
		t.Fatalf("живой коннект должен защищать ник, got %v", m)
	}

	// «Молчащее» соединение: backdate lastRecv → перехват.
	l := lobbyForNick(nick)
	if l == nil || l.Room == nil {
		t.Fatalf("лобби/комната не найдены для takeover")
	}
	l.Room.mu.Lock()
	for _, c := range l.Room.clients {
		c.lastRecvNano.Store(time.Now().Add(-time.Minute).UnixNano())
	}
	l.Room.mu.Unlock()

	ws3 := dialWS(t, srv.URL, tok, code)
	defer ws3.Close()
	wsReadUntil(t, ws3, "joined", 5*time.Second) // перехват успешен

	// Старый коннект закрыт сервером.
	ws1.SetReadDeadline(time.Now().Add(3 * time.Second))
	for {
		if _, _, err := ws1.ReadMessage(); err != nil {
			break // соединение закрыто — как и ожидалось
		}
	}

	leaveLobby(nick)
}

// TestBeginWavesIdempotent — повторный Start во время отсчёта не сбрасывает
// таймер (и не чистит арену); после вайпа (waveOver) рестарт работает.
func TestBeginWavesIdempotent(t *testing.T) {
	l := createLobby("idem_user_a", false)
	defer leaveLobby("idem_user_a")
	r := l.Room

	r.beginWaves()
	if !r.wavePending {
		t.Fatalf("после beginWaves должно быть wavePending=true")
	}
	r.mu.Lock()
	r.waveBreakT = 2.0 // «прошло 3 секунды отсчёта»
	r.mu.Unlock()

	r.beginWaves() // повторный клик Start
	r.mu.Lock()
	got := r.waveBreakT
	r.mu.Unlock()
	if got != 2.0 {
		t.Fatalf("повторный Start сбросил отсчёт: waveBreakT=%v, хотели 2.0", got)
	}

	// После вайпа рестарт доступен.
	r.mu.Lock()
	r.waveOver = true
	r.wavePending = false
	r.mu.Unlock()
	r.beginWaves()
	r.mu.Lock()
	pending := r.wavePending
	over := r.waveOver
	r.mu.Unlock()
	if !pending || over {
		t.Fatalf("после waveOver beginWaves должен перезапускать матч (pending=%v, over=%v)", pending, over)
	}
}

// TestLobbyCapacity — в пати максимум lobbyMaxPlayers; лишний вход отклонён.
func TestLobbyCapacity(t *testing.T) {
	code := createLobby("cap_host_user", false).Code
	defer leaveLobby("cap_host_user")
	for i := 1; i < lobbyMaxPlayers; i++ {
		nick := fmt.Sprintf("cap_user_%d", i)
		if joinLobby(nick, code) == nil {
			t.Fatalf("join #%d должен проходить", i)
		}
		defer leaveLobby(nick)
	}
	if l := joinLobby("cap_user_overflow", code); l != nil {
		leaveLobby("cap_user_overflow")
		t.Fatalf("вход сверх %d игроков должен отклоняться", lobbyMaxPlayers)
	}
}

// TestLobbyCodeRace — параллельное создание лобби не даёт гонок на lobbies
// (запускать с -race): genLobbyCode должен читать карту под мьютексом.
func TestLobbyCodeRace(t *testing.T) {
	var wg sync.WaitGroup
	codes := make(chan string, 64)
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			nick := fmt.Sprintf("race_user_%d", i)
			l := createLobby(nick, true)
			codes <- l.Code
			leaveLobby(nick)
		}(i)
	}
	wg.Wait()
	close(codes)
	seen := map[string]bool{}
	for c := range codes {
		if seen[c] {
			t.Fatalf("дубликат кода лобби: %s", c)
		}
		seen[c] = true
	}
}
