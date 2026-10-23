package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

// theRoom - глобальная "комната", в которой играют все подключившиеся игроки.
// Создаётся один раз в main и используется во всех остальных файлах.
var theRoom *Room

// noCache оборачивает http-хендлер, отключая кэш для HTML (чтобы браузер
// всегда брал свежую версию клиента) и ставя no-cache для остальных файлов.
func noCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".html") || r.URL.Path == "/" {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Header().Set("Pragma", "no-cache")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}
		next.ServeHTTP(w, r)
	})
}

// logRequests пишет каждый входящий HTTP-запрос в лог (метод, путь, IP клиента,
// протокол). Нужен для диагностики: доходят ли запросы от удалённых игроков
// (Radmin VPN, LAN) до сервера вообще.
func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		if h := r.Header.Get("X-Forwarded-For"); h != "" {
			ip = h + " (fwd from " + ip + ")"
		}
		log.Printf("[http] %s %s — %s %s\n", r.Method, r.URL.Path, ip, r.Proto)
		next.ServeHTTP(w, r)
	})
}

// Основная точка входа в программу.
// Что происходит: сервер слушает весь сетевой интерфейс (0.0.0.0) на порту 8090
// (HTTPS) и 8091 (HTTP), раздаёт клиенту (страницу + JS/PixiJS) и принимает
// WebSocket-подключения игроков.
func main() {
	// Создаём комнату. 20 "тиков" в секунду - ровный игровой темп.
	theRoom = NewRoom(20)
	// Подключаем MongoDB.
	initDB()
	// Фоновая чистка «призрачных» лобби (вкладку закрыли без lobby-leave).
	go lobbyReaperLoop()
	// Определяем порт. По умолчанию 8090, но можно поменять переменной PORT.
	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
	}

	// Настраиваем веб-роутер.
	mux := http.NewServeMux()

	// Статичные файлы клиента (index.html, js/, и т.п.) лежат в папке client/.
	// Мы отдаём их со скрипта по пути "/". Отключаем кэш HTML (index.html),
	// чтобы браузер всегда брал свежую версию клиента (а не старый main.js).
	clientDir := "../client"
	mux.Handle("/", noCache(http.StripPrefix("/", http.FileServer(http.Dir(clientDir)))))

	// WebSocket-эндпоинт для подключения игроков.
	mux.HandleFunc("/ws", handleWebSocket)

	// API эндпоинты авторизации.
	mux.HandleFunc("/api/register", handleRegister)
	mux.HandleFunc("/api/login", handleLogin)
	mux.HandleFunc("/api/profile", handleProfile)
	mux.HandleFunc("/api/menu-setbar", handleMenuSetBar)
	mux.HandleFunc("/api/menu-move", handleMenuMove)
	mux.HandleFunc("/api/menu-unequip", handleMenuUnequip)
	mux.HandleFunc("/api/menu-drop", handleMenuDrop)
	mux.HandleFunc("/api/craft", handleCraft)
	mux.HandleFunc("/api/craft-take", handleCraftTake)
	mux.HandleFunc("/api/absorb", handleAbsorb)
	mux.HandleFunc("/api/online", handleOnline)
	// API лобби (волновой режим): создание/вход по коду/список/выход/старт.
	mux.HandleFunc("/api/lobby-create", handleLobbyCreate)
	mux.HandleFunc("/api/lobby-join", handleLobbyJoin)
	mux.HandleFunc("/api/lobby-list", handleLobbyList)
	mux.HandleFunc("/api/lobby-leave", handleLobbyLeave)
	mux.HandleFunc("/api/lobby-info", handleLobbyInfo)
	mux.HandleFunc("/api/lobby-start", handleLobbyStart)
	mux.HandleFunc("/api/lobby-zone", handleLobbyZone)

	// API редактирования стен зон (editor.html): GET — текущая раскладка,
	// POST (только админ) — сохранить новую раскладку в walls.json.
	mux.HandleFunc("/api/walls", handleWalls)

	// ZHENYA: дорогой друг, обрати внимание на "0.0.0.0" - это значит "слушать
	// на всех сетевых интерфейсах". Так к серверу смогут подключаться другие
	// устройства в твоей сети (телефон, планшет) и сервер с Ubuntu.
	addr := "0.0.0.0:" + port
	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "8091"
	}
	httpAddr := "0.0.0.0:" + httpPort

	// Самоподписанный сертификат для HTTPS (если нет cert.pem/key.pem — генерируем).
	certFile, keyFile := "cert.pem", "key.pem"
	ensureSelfSignedCert(certFile, keyFile)

	log.Printf("HTTPS-сервер запущен на %s", addr)
	log.Printf("Открой в браузере: https://localhost:%s", port)
	if ip := lanIP(); ip != "" {
		log.Printf("Для других устройств в сети (LAN) заходи: http://%s:%s", ip, httpPort)
	}
	log.Printf("Публичный доступ (через HTTPS): https://ВАШ_ПУБЛИЧНЫЙ_IP:%s", port)

	// HTTPS (публичный доступ) и обычный HTTP (локальная отладка без предупреждений).
	handler := logRequests(mux)
	go func() {
		if err := http.ListenAndServe(httpAddr, handler); err != nil {
			log.Fatal("Ошибка запуска HTTP-сервера:", err)
		}
	}()
	if err := http.ListenAndServeTLS(addr, certFile, keyFile, handler); err != nil {
		log.Fatal("Ошибка запуска HTTPS-сервера:", err)
	}
}

// ensureSelfSignedCert генерирует самоподписанный сертификат (cert.pem/key.pem),
// если файлов ещё нет. В SAN добавляем localhost, 127.0.0.1 и локальные IP, а
// также IP из переменной TLS_EXTRA_IPS (через запятую — можно вписать публичный IP).
func ensureSelfSignedCert(certFile, keyFile string) {
	if _, err := os.Stat(certFile); err == nil {
		if _, err2 := os.Stat(keyFile); err2 == nil {
			return // сертификат уже есть
		}
	}
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatal("Не удалось сгенерировать ключ TLS:", err)
	}
	serial, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	template := x509.Certificate{
		SerialNumber: serial,
		Subject:      pkix.Name{CommonName: "SecretProject Server", Organization: []string{"SecretProject"}},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	// DNS-имена и IP в SAN.
	template.DNSNames = []string{"localhost"}
	template.IPAddresses = []net.IP{net.ParseIP("127.0.0.1")}
	if lip := lanIP(); lip != "" {
		if ip := net.ParseIP(lip); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		}
	}
	if extra := os.Getenv("TLS_EXTRA_IPS"); extra != "" {
		for _, s := range strings.Split(extra, ",") {
			if ip := net.ParseIP(strings.TrimSpace(s)); ip != nil {
				template.IPAddresses = append(template.IPAddresses, ip)
			}
		}
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		log.Fatal("Не удалось создать сертификат TLS:", err)
	}
	certOut, err := os.Create(certFile)
	if err != nil {
		log.Fatal("Не удалось записать cert.pem:", err)
	}
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: der})
	certOut.Close()
	keyOut, err := os.Create(keyFile)
	if err != nil {
		log.Fatal("Не удалось записать key.pem:", err)
	}
	pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	keyOut.Close()
	log.Printf("Сгенерирован самоподписанный сертификат (%s, %s)", certFile, keyFile)
}

// lanIP пытается определить IP-адрес этого компьютера в локальной сети.
// Нужно, чтобы показать адрес, по которому другие устройства будут заходить.
// Если не получилось определить - возвращаем пустую строку (это не страшно).
func lanIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return ""
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

type authRequest struct {
	Nick     string `json:"nick"`
	Password string `json:"password"`
}

type authResponse struct {
	OK    bool   `json:"ok"`
	Token string `json:"token,omitempty"`
	Error string `json:"error,omitempty"`
}

func handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request")
		return
	}
	req.Nick = strings.TrimSpace(req.Nick)
	if len(req.Nick) < 2 || len(req.Nick) > 16 {
		jsonError(w, "Nickname must be 2-16 characters")
		return
	}
	if len(req.Password) < 4 {
		jsonError(w, "Password must be at least 4 characters")
		return
	}
	token, errMsg := registerUser(req.Nick, req.Password)
	if errMsg != "" {
		jsonError(w, errMsg)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(authResponse{OK: true, Token: token})
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req authRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request")
		return
	}
	req.Nick = strings.TrimSpace(req.Nick)
	if req.Nick == "" || req.Password == "" {
		jsonError(w, "Fill in all fields")
		return
	}
	token, errMsg := loginUser(req.Nick, req.Password)
	if errMsg != "" {
		jsonError(w, errMsg)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(authResponse{OK: true, Token: token})
}

func jsonError(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(authResponse{OK: false, Error: msg})
}

// wallsDoc — сериализуемая раскладка стен (локальные координаты зон).
type wallsDoc struct {
	Zones   [][]WallRect `json:"zones"`
	Names   []string     `json:"names"`
	ZoneW   float64      `json:"zoneW"`
	ZoneH   float64      `json:"zoneH"`
	SafeH   float64      `json:"safeH"`
	WallTh  float64      `json:"wallTh"`
	WallGap float64      `json:"wallGap"`
	// Индивидуальные параметры зон: у второго уровня SafeH/WallTh/WallGap = 0.
	ZoneSafeH   []float64 `json:"zoneSafeH"`
	ZoneWallTh  []float64 `json:"zoneWallTh"`
	ZoneWallGap []float64 `json:"zoneWallGap"`
}

// currentWallsDoc собирает текущую раскладку для редактора.
func currentWallsDoc() wallsDoc {
	doc := wallsDoc{
		Zones: make([][]WallRect, len(wallLayouts)),
		Names: make([]string, len(zones)),
		ZoneW: zoneW, ZoneH: zoneH,
		SafeH: safeZoneH, WallTh: zoneWallTh, WallGap: zoneWallGap,
		ZoneSafeH:   make([]float64, len(zones)),
		ZoneWallTh:  make([]float64, len(zones)),
		ZoneWallGap: make([]float64, len(zones)),
	}
	for i := range wallLayouts {
		doc.Zones[i] = make([]WallRect, 0, len(wallLayouts[i]))
		doc.Zones[i] = append(doc.Zones[i], wallLayouts[i]...)
		if i < len(zones) {
			doc.Names[i] = zones[i].Name
			safeH, wallTh, wallGap := safeZoneH, zoneWallTh, zoneWallGap
			if !hasSafeZone(&zones[i]) {
				safeH, wallTh, wallGap = 0, 0, 0
			}
			doc.ZoneSafeH[i] = safeH
			doc.ZoneWallTh[i] = wallTh
			doc.ZoneWallGap[i] = wallGap
		}
	}
	return doc
}

// handleWalls обслуживает редактор стен:
//
//	GET  — отдать текущую раскладку;
//	POST — {token, zones:[[{x,y,w,h}...]...]} — применить и сохранить (только админ).
func handleWalls(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(currentWallsDoc())
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Token string       `json:"token"`
		Zones [][]WallRect `json:"zones"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	nick, ok := validateToken(req.Token)
	if !ok || !isAdmin(nick) {
		http.Error(w, "admin required", http.StatusForbidden)
		return
	}
	if len(req.Zones) != len(zones) {
		http.Error(w, "wrong zones count", http.StatusBadRequest)
		return
	}
	for i := range req.Zones {
		setWallLayout(i, req.Zones[i])
	}
	if err := saveWalls(); err != nil {
		http.Error(w, "save failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(currentWallsDoc())
}

type profileRequest struct {
	Token string `json:"token"`
}

type profileResponse struct {
	OK        bool          `json:"ok"`
	Nick      string        `json:"nick,omitempty"`
	Level     int           `json:"level"`
	XP        float64       `json:"xp"`
	XPMax     float64       `json:"xpMax"`
	Inventory []InvStack    `json:"inventory,omitempty"`
	Hotbars   [2][]PetalDef `json:"hotbars,omitempty"`
	ActiveBar int           `json:"activeBar"`
	BestWave  int           `json:"bestWave,omitempty"` // рекорд лучшей волны
	Pending   []PetalDef    `json:"pending,omitempty"`  // отложенные результаты крафта
	Error     string        `json:"error,omitempty"`
}

// handleProfile возвращает профиль игрока (уровень/XP) по валидному токену.
// Нужен, чтобы при перезагрузке страницы меню сразу показывало актуальные данные.
func handleProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req profileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request")
		return
	}
	nick, ok := validateToken(req.Token)
	if !ok {
		jsonError(w, "Invalid authorization token")
		return
	}
	lvl, xp, inv, hotbars, activeBar := loadProfile(nick)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profileResponse{
		OK: true, Nick: nick, Level: lvl, XP: xp, XPMax: xpForLevel(lvl),
		Inventory: inv, Hotbars: hotbars, ActiveBar: activeBar,
		BestWave: loadBestWave(nick),
		Pending:  loadPendingCraft(nick),
	})
}

// menuEditBase — общая валидация для правок меню: извлекает токен, грузит
// профиль из БД и возвращает statelessProfile для операции.
func menuEditBase(w http.ResponseWriter, req map[string]interface{}) (*statelessProfile, string) {
	tok, _ := req["token"].(string)
	nick, ok := validateToken(tok)
	if !ok {
		return nil, "Invalid authorization token"
	}
	lvl, _, inv, hotbars, activeBar := loadProfile(nick)
	sp := &statelessProfile{inv: inv, hotbars: hotbars, activeBar: activeBar, nick: nick, level: lvl}
	return sp, ""
}

// saveAndRespond сохраняет изменённый профиль и возвращает его клиенту.
func saveAndRespond(w http.ResponseWriter, sp *statelessProfile, nick string) {
	lvl, xp, _, _, _ := loadProfile(nick)
	saveProfile(nick, lvl, xp, sp.inv, sp.hotbars, sp.activeBar)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profileResponse{
		OK: true, Nick: nick, Level: lvl, XP: xp, XPMax: xpForLevel(lvl),
		Inventory: sp.inv, Hotbars: sp.hotbars, ActiveBar: sp.activeBar,
		Pending: loadPendingCraft(nick),
	})
}

// handleMenuSetBar меняет активную линейку прямо в профиле (БД).
func handleMenuSetBar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request")
		return
	}
	sp, errMsg := menuEditBase(w, req)
	if errMsg != "" {
		jsonError(w, errMsg)
		return
	}
	bar := int(req["bar"].(float64))
	sp.setActiveBar(bar)
	saveAndRespond(w, sp, sp.nick)
}

// handleMenuMove перемещает петаль между инвентарём и хотбарами (БД).
// into: {token, from:{src,kind,rarity|bar,slot}, toBar, toSlot}.
func handleMenuMove(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request")
		return
	}
	sp, errMsg := menuEditBase(w, req)
	if errMsg != "" {
		jsonError(w, errMsg)
		return
	}
	from, _ := req["from"].(map[string]interface{})
	toBar := int(req["toBar"].(float64))
	toSlot := int(req["toSlot"].(float64))
	sp.move(from, toBar, toSlot)
	saveAndRespond(w, sp, sp.nick)
}

// handleMenuUnequip снимает петаль со слота хотбара обратно в инвентарь (БД).
// into: {token, bar, slot}.
func handleMenuUnequip(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request")
		return
	}
	sp, errMsg := menuEditBase(w, req)
	if errMsg != "" {
		jsonError(w, errMsg)
		return
	}
	bar := int(req["bar"].(float64))
	slot := int(req["slot"].(float64))
	sp.unequip(bar, slot)
	saveAndRespond(w, sp, sp.nick)
}

// handleMenuDrop уничтожает петаль из профиля (БД). Начисляет XP, как абсорб
// одной петали (petalStackXP) — «выбросить за XP» обещает подсказка в меню.
// into: {token, from:{src,kind,rarity|bar,slot}}.
func handleMenuDrop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request")
		return
	}
	sp, errMsg := menuEditBase(w, req)
	if errMsg != "" {
		jsonError(w, errMsg)
		return
	}
	from, _ := req["from"].(map[string]interface{})

	// Захватываем статы петали ДО удаления — XP считается по виду/редкости.
	var dropped PetalDef
	var hasDef bool
	if from != nil {
		if src, _ := from["src"].(string); src == "hot" {
			bar, _ := from["bar"].(float64)
			slot, _ := from["slot"].(float64)
			if d, present := sp.hotbarAt(int(bar), int(slot)); present {
				dropped, hasDef = *d, true
			}
		} else {
			kind, _ := from["kind"].(string)
			rarity := 0
			if v, ok := from["rarity"].(float64); ok {
				rarity = int(v)
			}
			if i := sp.findStack(kind, rarity); i >= 0 && sp.inv[i].Count > 0 {
				dropped, hasDef = sp.inv[i].PetalDef, true
			}
		}
	}

	if !sp.drop(from) {
		jsonError(w, "Nothing to drop")
		return
	}

	// Начисляем XP (как абсорб, цена с потолком — порт родителя) и пересчитываем
	// уровень через общую кривую (кап 60, applyXPGain).
	lvl, xp, _, _, _ := loadProfile(sp.nick)
	if hasDef {
		gain := petalXPCapped(dropped)
		if gain > 0 {
			lvl, xp = applyXPGain(lvl, xp, gain)
		}
	}
	saveProfile(sp.nick, lvl, xp, sp.inv, sp.hotbars, sp.activeBar)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(profileResponse{
		OK: true, Nick: sp.nick, Level: lvl, XP: xp, XPMax: xpForLevel(lvl),
		Inventory: sp.inv, Hotbars: sp.hotbars, ActiveBar: sp.activeBar,
		Pending: loadPendingCraft(sp.nick),
	})
}

// craftResponse — результат операции крафта.
type craftResponse struct {
	OK        bool         `json:"ok"`
	Status    string       `json:"status"`               // success | fail
	Kind      string       `json:"kind"`                 // вид петалей попытки
	Rarity    int          `json:"rarity"`               // редкость петалей попытки
	NextRar   int          `json:"nextRarity,omitempty"` // новая редкость при успехе
	Attempts  int          `json:"attempts"`             // сколько попыток (вложено/5)
	Successes int          `json:"successes"`            // сколько удачных
	Lost      int          `json:"lost"`                 // сколько потеряно при провалах
	Chance    float64      `json:"chance"`               // шанс успеха, %
	Batches   []CraftBatch `json:"batches,omitempty"`    // лог каждого запуска катушки
	Pending   []PetalDef   `json:"pending,omitempty"`    // результаты (в окне крафта)
	Inventory []InvStack   `json:"inventory,omitempty"`
	Error     string       `json:"error,omitempty"`
}

// handleCraft выполняет крафт из пачки петалей (kind, rarity) количеством count.
// into: {token, kind, rarity, count}.
func handleCraft(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request")
		return
	}
	tok, _ := req["token"].(string)
	nick, ok := validateToken(tok)
	if !ok {
		jsonError(w, "Invalid authorization token")
		return
	}
	lvl, xp, inv, hotbars, activeBar := loadProfile(nick)
	sp := &statelessProfile{inv: inv, hotbars: hotbars, activeBar: activeBar, nick: nick, level: lvl}
	sp.pending = loadPendingCraft(nick)

	kind, _ := req["kind"].(string)
	rarity := int(req["rarity"].(float64))
	count := 5
	if v, has := req["count"]; has {
		c := int(v.(float64))
		if c >= 5 && c < 1000 {
			count = c
		}
	}
	status, successes, lost, results, batches := sp.craft(kind, rarity, count)
	if status == "error" {
		jsonError(w, "Need 5+ petals of the same kind and rarity")
		return
	}

	// Сохраняем инвентарь (расход) и, при успехе, отложенные результаты.
	saveProfile(nick, lvl, xp, sp.inv, sp.hotbars, sp.activeBar)
	savePendingCraft(nick, sp.pending)

	resp := craftResponse{
		OK: true, Status: status, Kind: kind, Rarity: rarity,
		Attempts: count / 5, Successes: successes, Lost: lost,
		Chance: craftSuccessChance(rarity), Batches: batches,
		Inventory: sp.inv, Pending: sp.pending,
	}
	if successes > 0 {
		resp.NextRar = rarity + 1
		// Всемирное объявление: игрок скрафтил Fractal (6) или Chaos (7).
		// Все результаты одного батча однородны (вид+редкость), так что берём
		// первый из results, а количество = число успехов.
		if nextRar := rarity + 1; nextRar >= rarityFractal {
			rarWord := rarityNames[nextRar] // "Fractal" | "Chaos"
			petalName := "petal"
			if len(results) > 0 && results[0].Name != "" {
				petalName = results[0].Name
			}
			msg := craftCelebrationText(nick, rarWord, petalName, successes)
			if msg != "" {
				// kind делит объявление на два цвета в чате: Fractal и Chaos.
				announceKind := "craft" + rarWord
				announceText := msg
				// Объявление появляется в чате ЧЕРЕЗ 6 СЕКУНД после начала крутки:
				// катушка крутится ~5 с (владелец видит результат), плюс 1 секунда.
				// Задержка на сервере: все видят синхронно, BroadcastSystem сам
				// берёт мьютекс, зовём из фоновой горутины.
				time.AfterFunc(6*time.Second, func() {
					// Анонс — в лобби игрока (волновой режим), если есть;
					// иначе в общий мир (legacy).
					if room := lobbyRoomForNick(nick); room != nil {
						room.BroadcastSystem(announceKind, announceText)
					} else if theRoom != nil {
						theRoom.BroadcastSystem(announceKind, announceText)
					}
				})
			}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// craftCelebrationText — красивое всемирное объявление о крафте Fractal/Chaos.
// Показывается ВСЕМ игрокам в чате, когда любой игрок скрафтил петаль самого
// высокого тира (Fractal = редкость 6, Chaos = редкость 7).
func craftCelebrationText(nick, rarWord, petalName string, n int) string {
	if n < 1 {
		return ""
	}
	switch rarWord {
	case "Chaos", "Fractal":
		return fmt.Sprintf("%s has crafted a %s %s x%d!", nick, rarWord, petalName, n)
	}
	return ""
}

// handleCraftTake переносит отложенный результат крафта в инвентарь.
// into: {token}.
func handleCraftTake(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request")
		return
	}
	sp, errMsg := menuEditBase(w, req)
	if errMsg != "" {
		jsonError(w, errMsg)
		return
	}
	sp.pending = loadPendingCraft(sp.nick)
	if !sp.takeCraft() {
		jsonError(w, "No pending craft result")
		return
	}
	savePendingCraft(sp.nick, nil)
	saveAndRespond(w, sp, sp.nick)
}

// absorbResponse — ответ абсорба: новый уровень/XP, сколько опыта дали петали.
type absorbResponse struct {
	OK        bool          `json:"ok"`
	Level     int           `json:"level"`
	XP        float64       `json:"xp"`
	XPMax     float64       `json:"xpMax"`
	XPAdd     int           `json:"xpAdd"`     // сколько опыта получено за абсорб
	Inventory []InvStack    `json:"inventory"` // всегда шлём (даже пустой)
	Hotbars   [2][]PetalDef `json:"hotbars,omitempty"`
	ActiveBar int           `json:"activeBar"`
	Error     string        `json:"error,omitempty"`
}

// handleAbsorb уничтожает петали за XP. into: {token, items:[{kind,rarity,count}]}.
// Петали списываются безвозвратно; за каждый игрок получает XP (petalStackXP).
func handleAbsorb(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request")
		return
	}
	tok, _ := req["token"].(string)
	nick, ok := validateToken(tok)
	if !ok {
		jsonError(w, "Invalid authorization token")
		return
	}
	lvl, xp, inv, hotbars, activeBar := loadProfile(nick)
	sp := &statelessProfile{inv: inv, hotbars: hotbars, activeBar: activeBar, nick: nick, level: lvl}

	// Разбираем items [{kind,rarity,count}].
	raw, _ := req["items"].([]interface{})
	var items []absorbItem
	for _, rw := range raw {
		obj, ok2 := rw.(map[string]interface{})
		if !ok2 {
			continue
		}
		kind, _ := obj["kind"].(string)
		rarity := int(obj["rarity"].(float64))
		count := int(obj["count"].(float64))
		if kind == "" || count <= 0 {
			continue
		}
		items = append(items, absorbItem{Kind: kind, Rarity: rarity, Count: count})
	}
	if len(items) == 0 {
		jsonError(w, "No petals to absorb")
		return
	}

	added := float64(int(sp.absorb(items)))
	if added <= 0 {
		jsonError(w, "No available petals to absorb")
		return
	}

	// Применяем опыт и пересчитываем уровни.
	xp += added
	for {
		need := xpForLevel(sp.level)
		if xp >= need {
			xp -= need
			sp.level++
		} else {
			break
		}
	}

	saveProfile(nick, sp.level, xp, sp.inv, sp.hotbars, sp.activeBar)

	resp := absorbResponse{
		OK: true, Level: sp.level, XP: xp, XPMax: xpForLevel(sp.level),
		XPAdd:     int(added),
		Inventory: sp.inv, Hotbars: sp.hotbars, ActiveBar: sp.activeBar,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
