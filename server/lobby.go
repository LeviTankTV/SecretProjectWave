package main

// lobby.go - лобби: изолированные матчи с кодом приглашения.
// Хост создаёт лобби (публичное — видно в списке, приватное — только по коду),
// друзья подключаются по 6-значному коду. Каждое лобби владеет собственной
// волновой комнатой (newWaveRoom). Удаляется, когда все участники вышли.

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"sort"
	"sync"
	"time"
)

const lobbyMaxPlayers = 4

type Lobby struct {
	Code    string
	Public  bool
	Host    string
	Room    *Room
	Zone    int             // зона арены 0-7 (Point..Chaos), выбирает хост
	Members map[string]bool // ники участников (включая хоста)
	Created time.Time
}

var (
	lobbyMu     sync.Mutex
	lobbies     = map[string]*Lobby{} // code -> лобби
	lobbyOfNick = map[string]string{} // ник -> code
)

// memberSeen — время последней активности участника лобби (любой lobby-API
// вызов: клиент в меню пати опрашивает /api/lobby-info каждые 2с). Нужно,
// чтобы вычищать «призраков» — игроков, закрывших вкладку без lobby-leave.
var (
	memberSeenMu sync.Mutex
	memberSeen   = map[string]time.Time{} // ник -> время последней активности
)

// bumpMemberSeen отмечает активность игрока (вызывается из lobbyBase).
func bumpMemberSeen(nick string) {
	memberSeenMu.Lock()
	memberSeen[nick] = time.Now()
	memberSeenMu.Unlock()
}

func memberSeenRecently(nick string, now time.Time, within time.Duration) bool {
	memberSeenMu.Lock()
	t, ok := memberSeen[nick]
	memberSeenMu.Unlock()
	return ok && now.Sub(t) < within
}

// genLobbyCode — уникальный 6-значный код среди активных лобби.
// Читает lobbies ПОД мьютексом: параллельные createLobby без лока давали
// гонку «concurrent map read and map write» с возможным падением сервера.
func genLobbyCode() string {
	lobbyMu.Lock()
	defer lobbyMu.Unlock()
	for attempts := 0; attempts < 1000; attempts++ {
		code := ""
		for i := 0; i < 6; i++ {
			code += string(rune('0' + rand.Intn(10)))
		}
		if _, taken := lobbies[code]; !taken {
			return code
		}
	}
	// Практически недостижимо; добавляем суффикс из времени.
	return time.Now().Format("050403")
}

// createLobby создаёт лобби с волновой комнатой. Игрок не может состоять
// в двух лобби — сначала выходит из прежнего.
func createLobby(nick string, public bool) *Lobby {
	leaveLobby(nick)
	bumpMemberSeen(nick)
	l := &Lobby{
		Code:    genLobbyCode(),
		Public:  public,
		Host:    nick,
		Room:    newWaveRoom(20),
		Members: map[string]bool{nick: true},
		Created: time.Now(),
	}
	lobbyMu.Lock()
	lobbies[l.Code] = l
	lobbyOfNick[nick] = l.Code
	lobbyMu.Unlock()
	log.Printf("Лобби %s создано (хост %s, публичное: %v)", l.Code, nick, public)
	return l
}

// joinLobby добавляет игрока в лобби по коду.
func joinLobby(nick, code string) *Lobby {
	leaveLobby(nick)
	bumpMemberSeen(nick)
	lobbyMu.Lock()
	defer lobbyMu.Unlock()
	l, ok := lobbies[code]
	if !ok {
		return nil
	}
	if len(l.Members) >= lobbyMaxPlayers {
		return nil
	}
	l.Members[nick] = true
	lobbyOfNick[nick] = code
	return l
}

// leaveLobby выводит игрока из его лобби (если состоял). Пустое лобби удаляется.
func leaveLobby(nick string) {
	lobbyMu.Lock()
	code, ok := lobbyOfNick[nick]
	if !ok {
		lobbyMu.Unlock()
		return
	}
	l := lobbies[code]
	delete(lobbyOfNick, nick)
	if l != nil {
		delete(l.Members, nick)
		// Хост ушёл — передаём хостство первому оставшемуся.
		if l.Host == nick && len(l.Members) > 0 {
			for m := range l.Members {
				l.Host = m
				break
			}
		}
		if len(l.Members) == 0 {
			delete(lobbies, code)
			if l.Room != nil {
				l.Room.close()
			}
			log.Printf("Лобби %s удалено (пустое)", code)
			l = nil
		}
	}
	lobbyMu.Unlock()
}

// setLobbyZone меняет зону арены (только хост). Возвращает false, если зона
// вне диапазона первого уровня или игрок не хост.
func setLobbyZone(nick string, zone int) bool {
	if zone < 0 || zone >= secondTierOffset {
		return false
	}
	lobbyMu.Lock()
	defer lobbyMu.Unlock()
	code, ok := lobbyOfNick[nick]
	if !ok {
		return false
	}
	l := lobbies[code]
	if l == nil || l.Host != nick {
		return false
	}
	l.Zone = zone
	if l.Room != nil {
		l.Room.setWaveZone(zone)
	}
	return true
}

// zoneInfoFor — имя и цвет зоны для клиента (безопасно для любого индекса).
func zoneInfoFor(zone int) (string, uint32) {
	if zone < 0 || zone >= len(zoneNames) {
		zone = 0
	}
	return zoneNames[zone], zoneColors[zone]
}

// lobbyForNick — лобби игрока или nil.
func lobbyForNick(nick string) *Lobby {
	lobbyMu.Lock()
	defer lobbyMu.Unlock()
	code, ok := lobbyOfNick[nick]
	if !ok {
		return nil
	}
	return lobbies[code]
}

// lobbyRoomForNick — комната лобби игрока (nil, если не в лобби).
func lobbyRoomForNick(nick string) *Room {
	if l := lobbyForNick(nick); l != nil {
		return l.Room
	}
	return nil
}

// lobbyListInfo — элемент списка лобби для клиента.
type lobbyListInfo struct {
	Code      string `json:"code"`
	Host      string `json:"host"`
	Players   int    `json:"players"`
	Max       int    `json:"max"`
	Started   bool   `json:"started"`
	Zone      int    `json:"zone"`
	ZoneName  string `json:"zoneName"`
	ZoneColor uint32 `json:"zoneColor"`
}

// listPublicLobbies — открытые лобби для /api/lobby-list: только публичные,
// ещё НАБИРАЮЩИЕСЯ (не начавшие матч).
func listPublicLobbies() []lobbyListInfo {
	lobbyMu.Lock()
	defer lobbyMu.Unlock()
	out := make([]lobbyListInfo, 0, len(lobbies))
	for _, l := range lobbies {
		if !l.Public {
			continue
		}
		if l.Room != nil && l.Room.roomWaveStarted() {
			continue // матч уже идёт — в список поиска не показываем
		}
		zname, zcol := zoneInfoFor(l.Zone)
		out = append(out, lobbyListInfo{
			Code: l.Code, Host: l.Host,
			Players: len(l.Members), Max: lobbyMaxPlayers,
			Started: false,
			Zone:    l.Zone, ZoneName: zname, ZoneColor: zcol,
		})
	}
	return out
}

// ---- HTTP-обработчики ----

// lobbyPlayerInfo — участник лобби с его снаряжением (для анимированного
// превью в меню: круг игрока с крутящимися петалами + хотбар позади).
type lobbyPlayerInfo struct {
	Nick      string     `json:"nick"`
	Host      bool       `json:"host"`
	Level     int        `json:"level"`
	Hotbar    []PetalDef `json:"hotbar"`
	ActiveBar int        `json:"activeBar"`
	BestWave  int        `json:"bestWave"`
}

// lobbyLoadouts собирает снаряжение всех участников. Порядок СТАБИЛЬНЫЙ
// (хост первым, дальше по нику) — иначе игроки в превью меняются местами.
func lobbyLoadouts(l *Lobby) []lobbyPlayerInfo {
	names := lobbyMembers(l)
	out := make([]lobbyPlayerInfo, 0, len(names))
	for _, m := range names {
		_, _, _, hotbars, activeBar := loadProfile(m)
		hot := hotbars[activeBar]
		if len(hot) > 10 {
			hot = hot[:10]
		}
		out = append(out, lobbyPlayerInfo{
			Nick: m, Host: m == l.Host,
			Level: loadProgressLevel(m), Hotbar: hot, ActiveBar: activeBar,
			BestWave: loadBestWave(m),
		})
	}
	return out
}

// loadProgressLevel — только уровень (лёгкая обёртка над loadProgress).
func loadProgressLevel(nick string) int {
	lvl, _ := loadProgress(nick)
	return lvl
}

// zoneMobDecorInfo — вид моба для фонового декора лобби (форма/цвет/вес
// появления — как шанс спавна в зоне).
type zoneMobDecorInfo struct {
	Shape  string  `json:"shape"`
	Color  uint32  `json:"color"`
	Size   float64 `json:"size"`
	Weight int     `json:"weight"`
}

// zoneMobDecorFor — состав фоновых мобов выбранной зоны (пул + веса зоны).
func zoneMobDecorFor(zone int) []zoneMobDecorInfo {
	if zone < 0 || zone >= len(mobTypesInZone) {
		return nil
	}
	opts := mobTypesInZone[zone]
	ws := zoneSpawnWeights[zone]
	out := make([]zoneMobDecorInfo, 0, len(opts))
	for i, tname := range opts {
		mt, ok := mobTypes[tname]
		if !ok {
			continue
		}
		w := 1
		if i < len(ws) {
			w = ws[i]
		}
		out = append(out, zoneMobDecorInfo{Shape: mt.Shape, Color: mt.Color, Size: mt.BaseRadius, Weight: w})
	}
	return out
}

// lobbyResponse — общий ответ lobby-API.
type lobbyResponse struct {
	OK          bool               `json:"ok"`
	Error       string             `json:"error,omitempty"`
	Code        string             `json:"code,omitempty"`
	Host        string             `json:"host,omitempty"`
	IsHost      bool               `json:"isHost,omitempty"`
	InLobby     bool               `json:"inLobby"`
	Players     []string           `json:"players,omitempty"`
	PlayersData []lobbyPlayerInfo  `json:"playersData,omitempty"`
	Started     bool               `json:"started,omitempty"`
	Zone        int                `json:"zone"` // Point = зона 0: omitempty выкинул бы поле из JSON
	ZoneName    string             `json:"zoneName,omitempty"`
	ZoneColor   uint32             `json:"zoneColor,omitempty"`
	ZoneMobs    []zoneMobDecorInfo `json:"zoneMobs,omitempty"`
	// List без omitempty: пустой список должен приходить как [], иначе клиент
	// (Array.isArray check) оставляет на экране устаревший список с призраками.
	List []lobbyListInfo `json:"list"`
}

// Кэш «аккаунт существует» (30с): lobby-API дёргается каждые 2с каждым
// клиентом, а Mongo под нагрузкой игры отвечает сотнями миллисекунд — без
// кэша клики по зонам/кнопкам лобби ощущаются лагающими.
var (
	existsCacheMu sync.Mutex
	existsCache   = map[string]time.Time{}
)

func userExistsFast(nick string) bool {
	existsCacheMu.Lock()
	t, ok := existsCache[nick]
	existsCacheMu.Unlock()
	if ok && time.Since(t) < 30*time.Second {
		return true
	}
	if userExists(nick) {
		existsCacheMu.Lock()
		existsCache[nick] = time.Now()
		existsCacheMu.Unlock()
		return true
	}
	return false
}

func lobbyBase(w http.ResponseWriter, req map[string]interface{}) (string, bool) {
	tok, _ := req["token"].(string)
	nick, ok := validateToken(tok)
	if !ok {
		jsonError(w, "Invalid authorization token")
		return "", false
	}
	// Любой lobby-API вызов — признак жизни: клиент в меню пати опрашивает
	// lobby-info каждые 2с. Без этого призраки (закрытая вкладка) не отличимы
	// от живых участников.
	bumpMemberSeen(nick)
	if !userExistsFast(nick) {
		jsonError(w, "Account not found — re-login")
		return "", false
	}
	return nick, true
}

// handleLobbyCreate — {token, public:bool} → {ok, code}.
func handleLobbyCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request")
		return
	}
	nick, ok := lobbyBase(w, req)
	if !ok {
		return
	}
	public, _ := req["public"].(bool)
	l := createLobby(nick, public)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(lobbyResponse{OK: true, Code: l.Code, Host: l.Host, IsHost: true, InLobby: true, Players: lobbyMembers(l)})
}

func lobbyMembers(l *Lobby) []string {
	names := make([]string, 0, len(l.Members))
	for m := range l.Members {
		names = append(names, m)
	}
	// Хост первым, остальные по алфавиту — стабильный порядок для превью.
	sort.Slice(names, func(i, j int) bool {
		hi, hj := names[i] == l.Host, names[j] == l.Host
		if hi != hj {
			return hi
		}
		return names[i] < names[j]
	})
	return names
}

// handleLobbyJoin — {token, code} → {ok, code, host, players}.
func handleLobbyJoin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request")
		return
	}
	nick, ok := lobbyBase(w, req)
	if !ok {
		return
	}
	code, _ := req["code"].(string)
	code = trimCode(code)
	l := joinLobby(nick, code)
	if l == nil {
		jsonError(w, "Lobby not found or full")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(lobbyResponse{OK: true, Code: l.Code, Host: l.Host, IsHost: l.Host == nick, InLobby: true, Players: lobbyMembers(l)})
}

// trimCode нормализует введённый код: только цифры, максимум 6.
func trimCode(s string) string {
	out := make([]rune, 0, 6)
	for _, c := range s {
		if c >= '0' && c <= '9' {
			out = append(out, c)
		}
		if len(out) >= 6 {
			break
		}
	}
	return string(out)
}

// handleLobbyList — список публичных лобби.
func handleLobbyList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(lobbyResponse{OK: true, List: listPublicLobbies()})
}

// handleLobbyLeave — выход из лобби.
func handleLobbyLeave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request")
		return
	}
	nick, ok := lobbyBase(w, req)
	if !ok {
		return
	}
	leaveLobby(nick)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(lobbyResponse{OK: true, InLobby: false})
}

// handleLobbyInfo — состояние лобби игрока (или inLobby:false).
func handleLobbyInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request")
		return
	}
	nick, ok := lobbyBase(w, req)
	if !ok {
		return
	}
	l := lobbyForNick(nick)
	w.Header().Set("Content-Type", "application/json")
	if l == nil {
		json.NewEncoder(w).Encode(lobbyResponse{OK: true, InLobby: false})
		return
	}
	started := l.Room != nil && l.Room.roomWaveStarted()
	zname, zcol := zoneInfoFor(l.Zone)
	json.NewEncoder(w).Encode(lobbyResponse{OK: true, InLobby: true, Code: l.Code, Host: l.Host, IsHost: l.Host == nick, Players: lobbyMembers(l), PlayersData: lobbyLoadouts(l), Started: started, Zone: l.Zone, ZoneName: zname, ZoneColor: zcol, ZoneMobs: zoneMobDecorFor(l.Zone)})
}

// handleLobbyZone — {token, zone} — хост выбирает зону арены (0-7).
func handleLobbyZone(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request")
		return
	}
	nick, ok := lobbyBase(w, req)
	if !ok {
		return
	}
	zf, _ := req["zone"].(float64)
	if !setLobbyZone(nick, int(zf)) {
		jsonError(w, "Only the host can pick a zone (0-7)")
		return
	}
	l := lobbyForNick(nick)
	zname, zcol := zoneInfoFor(l.Zone)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(lobbyResponse{OK: true, Code: l.Code, Zone: l.Zone, ZoneName: zname, ZoneColor: zcol, ZoneMobs: zoneMobDecorFor(l.Zone)})
}

// handleLobbyStart — хост запускает/перезапускает матч.
func handleLobbyStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request")
		return
	}
	nick, ok := lobbyBase(w, req)
	if !ok {
		return
	}
	l := lobbyForNick(nick)
	if l == nil {
		jsonError(w, "You are not in a lobby")
		return
	}
	if l.Host != nick {
		jsonError(w, "Only the host can start the game")
		return
	}
	if l.Room != nil {
		l.Room.beginWaves()
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(lobbyResponse{OK: true, Code: l.Code, Started: true})
}

// dropAllLobbyMembers вызывается при остановке сервера — заглушка совместимости.
func dropAllLobbyMembers() {}

// Тайминги чистки лобби. vars, а не const — тесты ужинают их для проверки.
var (
	lobbyIdleKick  = 5 * time.Minute  // участник без активности и без WS (закрыл вкладку)
	matchIdleClose = 15 * time.Minute // участник в идущем матче без WS (запас на реконнект)
	lobbyMaxAge    = 24 * time.Hour
)

// reapIdleLobbies вычищает призраков из лобби: игроков, которые исчезли без
// lobby-leave (закрытая вкладка, обрыв сети, краш браузера). Участник удаляется,
// если давно не было lobby-API активности И у него нет живого WS-соединения в
// комнате лобби. Когда участников не остаётся — лобби удаляется (как при
// обычном выходе: хост переносится, комната закрывается).
func reapIdleLobbies() {
	now := time.Now()
	lobbyMu.Lock()
	defer lobbyMu.Unlock()
	for code, l := range lobbies {
		started := l.Room != nil && l.Room.roomWaveStarted()
		kickAfter := lobbyIdleKick
		if started {
			kickAfter = matchIdleClose
		}
		for m := range l.Members {
			if memberSeenRecently(m, now, kickAfter) {
				continue
			}
			if l.Room != nil && l.Room.hasClientNamed(m) {
				continue // живой игрок в матче (WS активен, меню не опрашивает)
			}
			delete(l.Members, m)
			delete(lobbyOfNick, m)
			log.Printf("Лобби %s: призрак %q удалён (нет активности %v)", code, m, kickAfter)
			if l.Host == m {
				l.Host = ""
				for rem := range l.Members {
					l.Host = rem
					break
				}
				log.Printf("Лобби %s: хостство передано %q", code, l.Host)
			}
		}
		if len(l.Members) == 0 || now.Sub(l.Created) > lobbyMaxAge {
			for n, c := range lobbyOfNick {
				if c == code {
					delete(lobbyOfNick, n)
				}
			}
			delete(lobbies, code)
			if l.Room != nil {
				l.Room.close()
			}
			log.Printf("Лобби %s удалено (reaper: пустое/устаревшее)", code)
		}
	}
	// Глобальный кэш активности подчищаем от записей, которые уже ни на что
	// не влияют (тайминги чистки заведомо меньше lobbyMaxAge).
	for n, t := range memberSeen {
		if now.Sub(t) > lobbyMaxAge {
			memberSeenMu.Lock()
			delete(memberSeen, n)
			memberSeenMu.Unlock()
		}
	}
}

// lobbyReaperLoop — фоновая чистка лобби (запускается из main).
func lobbyReaperLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		reapIdleLobbies()
	}
}
