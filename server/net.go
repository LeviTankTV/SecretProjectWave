package main

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// net.go - сеть: WebSocket, протокол сообщений.

type Client struct {
	conn     *websocket.Conn
	playerID string
	room     *Room
	send     chan []byte
	token    string
	// lastRecvNano — момент последнего полученного кадра (hello, input, ...).
	// Атомарно: пишет readPump, читает AddClient из другой горутины. Нужно,
	// чтобы отличать «молчащее» соединение после обрыва сети от живого игрока
	// при повторном входе с тем же ником.
	lastRecvNano atomic.Int64
}

func (c *Client) touchRecv() { c.lastRecvNano.Store(time.Now().UnixNano()) }

// lastRecvAge — сколько времени назад приходили данные. Для коннекта без
// единого кадра возвращаем 0 (считаем живым — не перехватывать).
func (c *Client) lastRecvAge() time.Duration {
	t := c.lastRecvNano.Load()
	if t == 0 {
		return 0
	}
	return time.Since(time.Unix(0, t))
}

type HelloMsg struct {
	Type  string `json:"type"`
	Token string `json:"token"`
	Color uint32 `json:"color"`
	Code  string `json:"code,omitempty"` // код лобби (волновой режим)
}

type JoinMsg struct {
	Type      string  `json:"type"`
	ID        string  `json:"id"`
	Admin     bool    `json:"admin"`
	God       bool    `json:"god"`
	Invisible bool    `json:"invisible"`
	Passive   bool    `json:"passive"`
	Team      int     `json:"team"`
	OneShot   bool    `json:"oneshot"`
	Size      float64 `json:"size"`
}

type ErrorMsg struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

type InputMsg struct {
	Type  string  `json:"type"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	ViewW float64 `json:"viewW"`
	ViewH float64 `json:"viewH"`
}

type ClientMsg struct {
	Type string `json:"type"`
}

type ChatMsg struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

const (
	sendBufferSize = 32
	maxMessageSize = 512
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	// helloReadWait — макс. время ожидания первого сообщения (hello). Короче
	// pongWait, чтобы боты/сканеры/полуоткрытые соединения не висели по 60с.
	helloReadWait = 10 * time.Second
)

// Троттлинг лога ошибок чтения hello: боты/портосканеры/обрывы соединений
// могут открывать коннект и тут же закрывать, не прислав hello — такой сценарий
// безвреден (клиент не добавляется, прогресс не теряется), но если логировать
// КАЖДЫЙ раз, консоль зальёт сотней строк в секунду. Пишем не чаще раза в 30с
// с счётчиком пропущенных.
var (
	helloFailMutex  sync.Mutex
	helloFailLast   time.Time
	helloFailCount  int
	helloFailLogged int
)

func logHelloFail(ip string, err error) {
	helloFailMutex.Lock()
	defer helloFailMutex.Unlock()
	helloFailCount++
	now := time.Now()
	if helloFailLast.IsZero() || now.Sub(helloFailLast) >= 30*time.Second {
		skip := helloFailCount - helloFailLogged - 1
		if skip > 0 {
			log.Printf("Не удалось прочитать приветствие от %s (ещё %d безвредных пропущено): %v", ip, skip, err)
		} else {
			log.Printf("Не удалось прочитать приветствие от %s: %v", ip, err)
		}
		helloFailLast = now
		helloFailLogged = helloFailCount
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin:       func(r *http.Request) bool { return true },
	EnableCompression: true,
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Не удалось установить WebSocket:", err)
		return
	}
	client := &Client{
		conn: conn,
		room: theRoom,
		send: make(chan []byte, sendBufferSize),
	}
	go client.readPump()
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) readPump() {
	// Паника в обработчике сообщений одного клиента НЕ валит весь сервер:
	// логируем стек и закрытся connection.
	defer func() {
		if p := recover(); p != nil {
			log.Printf("[PANIC] readPump %s: %v\n%s", remoteIP(c), p, debug.Stack())
		}
	}()
	ip := remoteIP(c)
	ipRegistered := false

	defer func() {
		if ipRegistered {
			c.room.RemoveClient(c, ip)
		}
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	// Первое сообщение — hello с токеном авторизации. Короткий дедлайн (не 60с):
	// соединения, которые открылись, но ничего не прислали, убираем быстро.
	c.conn.SetReadDeadline(time.Now().Add(helloReadWait))
	var hello HelloMsg
	if err := c.conn.ReadJSON(&hello); err != nil {
		logHelloFail(ip, err)
		return
	}
	c.touchRecv()
	// Восстанавливаем обычный таймаут (pong) — дальше действует pong-обработчик.
	c.conn.SetReadDeadline(time.Now().Add(pongWait))

	// Валидация токена — получаем ник из БД.
	nick, ok := validateToken(hello.Token)
	if !ok {
		m := ErrorMsg{Type: "error", Message: "Invalid authorization token"}
		msg, _ := json.Marshal(m)
		c.conn.SetWriteDeadline(time.Now().Add(writeWait))
		c.conn.WriteMessage(websocket.TextMessage, msg)
		time.Sleep(100 * time.Millisecond)
		return
	}

	// Аккаунт с этим ником может быть уже УДАЛЁН из БД (токены живут дольше
	// записей): играть «призраком» нельзя — прогресс/петали не сохранились бы.
	if !userExists(nick) {
		m := ErrorMsg{Type: "error", Message: "Account not found — re-login"}
		msg, _ := json.Marshal(m)
		c.conn.SetWriteDeadline(time.Now().Add(writeWait))
		c.conn.WriteMessage(websocket.TextMessage, msg)
		time.Sleep(100 * time.Millisecond)
		return
	}

	// Волновой режим: игрок должен состоять в лобби, и его комната — комната
	// лобби (по коду из hello). Без лобби — отказ: общий мир больше не играется.
	lobby := lobbyForNick(nick)
	if lobby == nil {
		m := ErrorMsg{Type: "error", Message: "Join or create a lobby first"}
		msg, _ := json.Marshal(m)
		c.conn.SetWriteDeadline(time.Now().Add(writeWait))
		c.conn.WriteMessage(websocket.TextMessage, msg)
		time.Sleep(100 * time.Millisecond)
		return
	}
	if trimCode(hello.Code) != lobby.Code {
		m := ErrorMsg{Type: "error", Message: "Lobby code mismatch — rejoin the lobby"}
		msg, _ := json.Marshal(m)
		c.conn.SetWriteDeadline(time.Now().Add(writeWait))
		c.conn.WriteMessage(websocket.TextMessage, msg)
		time.Sleep(100 * time.Millisecond)
		return
	}
	c.room = lobby.Room

	if errMsg := c.room.AddClient(c, nick, ip, hello.Color); errMsg != "" {
		m := ErrorMsg{Type: "error", Message: errMsg}
		msg, _ := json.Marshal(m)
		c.conn.SetWriteDeadline(time.Now().Add(writeWait))
		c.conn.WriteMessage(websocket.TextMessage, msg)
		time.Sleep(100 * time.Millisecond)
		return
	}
	ipRegistered = true
	c.token = hello.Token

	// Сообщаем ID.
	god, invis, pas, team, size, oneshot := c.room.adminSelfState(c)
	m := JoinMsg{Type: "joined", ID: c.playerID, Admin: isAdmin(nick), God: god, Invisible: invis, Passive: pas, Team: team, OneShot: oneshot, Size: size}
	msg, _ := json.Marshal(m)
	c.deliver(msg)

	// Мир.
	wmsg, _ := json.Marshal(c.room.buildWorld())
	c.deliver(wmsg)

	go c.writePump()

	// Читаем все входящие сообщения и диспатчим по type.
	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		c.touchRecv()
		var base ClientMsg
		if json.Unmarshal(raw, &base) != nil {
			continue
		}
		switch base.Type {
		case "input":
			var in InputMsg
			if json.Unmarshal(raw, &in) == nil {
				c.room.UpdateInput(c.playerID, in.X, in.Y, in.ViewW, in.ViewH)
			}
		case "shiftDown":
			c.room.mu.Lock()
			if p, ok := c.room.players[c.playerID]; ok {
				p.petalDir = -1
			}
			c.room.mu.Unlock()
		case "shiftUp":
			c.room.mu.Lock()
			if p, ok := c.room.players[c.playerID]; ok {
				p.petalDir = 0
			}
			c.room.mu.Unlock()
		case "spaceDown":
			c.room.mu.Lock()
			if p, ok := c.room.players[c.playerID]; ok {
				p.petalDir = +1
				// Миссил (blade): по пробелу активные клинки выстреливают по
				// касательной к орбите.
				c.room.fireBlades(c.playerID)
				// Рывок (void): тратит активный void-петаль и делает рывок.
				c.room.dashVoid(c.playerID)
			}
			c.room.mu.Unlock()
		case "spaceUp":
			c.room.mu.Lock()
			if p, ok := c.room.players[c.playerID]; ok {
				p.petalDir = 0
			}
			c.room.mu.Unlock()
		case "chat":
			var cm ChatMsg
			if json.Unmarshal(raw, &cm) == nil {
				if text := strings.TrimSpace(cm.Text); text != "" {
					c.room.BroadcastChat(c.playerID, text)
				}
			}
		case "setbar":
			var sb struct {
				Type string `json:"type"`
				Bar  int    `json:"bar"`
			}
			if json.Unmarshal(raw, &sb) == nil {
				c.room.SwitchBar(c.playerID, sb.Bar)
			}
		case "equip":
			var m struct {
				Type   string `json:"type"`
				Bar    int    `json:"bar"`
				Slot   int    `json:"slot"`
				Kind   string `json:"kind"`
				Rarity int    `json:"rarity"`
			}
			if json.Unmarshal(raw, &m) == nil {
				c.room.EquipPetal(c.playerID, m.Bar, m.Slot, m.Kind, m.Rarity)
			}
		case "unequip":
			var m struct {
				Type string `json:"type"`
				Bar  int    `json:"bar"`
				Slot int    `json:"slot"`
			}
			if json.Unmarshal(raw, &m) == nil {
				c.room.UnequipPetal(c.playerID, m.Bar, m.Slot)
			}
		case "drop":
			var m struct {
				Type   string `json:"type"`
				Kind   string `json:"kind"`
				Rarity int    `json:"rarity"`
			}
			if json.Unmarshal(raw, &m) == nil {
				c.room.DropPetal(c.playerID, m.Kind, m.Rarity)
			}
		case "dropHotbar":
			var m struct {
				Type string `json:"type"`
				Bar  int    `json:"bar"`
				Slot int    `json:"slot"`
			}
			if json.Unmarshal(raw, &m) == nil {
				c.room.DropHotbar(c.playerID, m.Bar, m.Slot)
			}
		case "move":
			var m struct {
				Type   string                 `json:"type"`
				From   map[string]interface{} `json:"from"`
				ToBar  int                    `json:"toBar"`
				ToSlot int                    `json:"toSlot"`
			}
			if json.Unmarshal(raw, &m) == nil {
				c.room.MovePetal(c.playerID, m.From, m.ToBar, m.ToSlot)
			}
		case "craftLive":
			// Внутриигровой крафт (та же логика, что в меню).
			var m struct {
				Type   string `json:"type"`
				Kind   string `json:"kind"`
				Rarity int    `json:"rarity"`
				Count  int    `json:"count"`
			}
			if json.Unmarshal(raw, &m) == nil {
				inv := m.Count
				if inv < 5 || inv > 1000 {
					inv = 5
				}
				c.room.LiveCraft(c.playerID, m.Kind, m.Rarity, inv)
			}
		case "craftLiveTake":
			c.room.LiveCraftTake(c.playerID)
		case "respawn":
			c.room.RespawnPlayer(c.playerID)
		case "adminSpawn":
			var am struct {
				Type   string `json:"type"`
				Kind   string `json:"kind"`
				Rarity int    `json:"rarity"`
				Mtype  string `json:"mtype"` // тип моба: nothing | gold (расширим позже)
			}
			if json.Unmarshal(raw, &am) == nil && roomAdmin(c) {
				c.room.adminSpawnMob(c, am.Kind, am.Rarity, am.Mtype)
			}
		case "adminPlayers":
			if roomAdmin(c) {
				c.room.adminPlayersList(c)
			}
		case "adminBan":
			var am struct {
				Type string `json:"type"`
				Name string `json:"name"`
			}
			if json.Unmarshal(raw, &am) == nil && roomAdmin(c) {
				c.room.adminBan(c, am.Name)
			}
		case "adminUnban":
			var am struct {
				Type string `json:"type"`
				Name string `json:"name"`
			}
			if json.Unmarshal(raw, &am) == nil && roomAdmin(c) {
				c.room.adminUnban(c, am.Name)
			}
		case "adminSpeed":
			var am struct {
				Type  string  `json:"type"`
				Value float64 `json:"value"`
			}
			if json.Unmarshal(raw, &am) == nil && roomAdmin(c) {
				c.room.adminSetSpeed(c, am.Value)
			}
		case "adminHP":
			var am struct {
				Type  string  `json:"type"`
				Value float64 `json:"value"`
			}
			if json.Unmarshal(raw, &am) == nil && roomAdmin(c) {
				c.room.adminSetHP(c, am.Value)
			}
		case "adminTeleport":
			var am struct {
				Type string `json:"type"`
				Name string `json:"name"`
			}
			if json.Unmarshal(raw, &am) == nil && roomAdmin(c) {
				c.room.adminTeleportTo(c, am.Name)
			}
		case "adminZoneTel":
			var am struct {
				Type string `json:"type"`
				Zone int    `json:"zone"`
			}
			if json.Unmarshal(raw, &am) == nil && roomAdmin(c) {
				c.room.adminTeleportZone(c, am.Zone)
			}
		case "adminZoneSafe":
			var am struct {
				Type string `json:"type"`
				Zone int    `json:"zone"`
			}
			if json.Unmarshal(raw, &am) == nil && roomAdmin(c) {
				c.room.adminTeleportSafe(c, am.Zone)
			}
		case "adminPull":
			var am struct {
				Type string `json:"type"`
				Name string `json:"name"`
			}
			if json.Unmarshal(raw, &am) == nil && roomAdmin(c) {
				c.room.adminPullPlayer(c, am.Name)
			}
		case "adminGod":
			if roomAdmin(c) {
				c.room.adminToggleGod(c)
			}
		case "adminInvisible":
			if roomAdmin(c) {
				c.room.adminToggleInvisible(c)
			}
		case "adminPassive":
			if roomAdmin(c) {
				c.room.adminTogglePassive(c)
			}
		case "adminPvp":
			if roomAdmin(c) {
				c.room.adminTogglePVP(c)
			}
		case "adminMobTeam":
			if roomAdmin(c) {
				c.room.adminToggleMobTeam(c)
			}
		case "adminOneShot":
			if roomAdmin(c) {
				c.room.adminToggleOneShot(c)
			}
		case "adminSize":
			var am struct {
				Type  string  `json:"type"`
				Delta float64 `json:"delta"`
			}
			if json.Unmarshal(raw, &am) == nil && roomAdmin(c) {
				c.room.adminSizeDelta(c, am.Delta)
			}
		case "adminGivePetals":
			var am struct {
				Type   string `json:"type"`
				Kind   string `json:"kind"`
				Rarity int    `json:"rarity"`
				Count  int    `json:"count"`
			}
			if json.Unmarshal(raw, &am) == nil && roomAdmin(c) {
				c.room.adminGivePetals(c, am.Kind, am.Rarity, am.Count)
			}
		case "adminGiveXP":
			var am struct {
				Type  string  `json:"type"`
				Value float64 `json:"value"`
			}
			if json.Unmarshal(raw, &am) == nil && roomAdmin(c) {
				c.room.adminGiveXP(c, am.Value)
			}
		case "adminAnnounce":
			var am struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}
			if json.Unmarshal(raw, &am) == nil && roomAdmin(c) {
				c.room.adminAnnounce(c, am.Text)
			}
		case "adminMobs":
			if roomAdmin(c) {
				c.room.adminMobCounts(c)
			}
		case "menu":
			c.room.ReturnToMenu(c.playerID, ip)
			return
		}
	}
}

func remoteIP(c *Client) string {
	addr := c.conn.RemoteAddr().String()
	if host, _, err := net.SplitHostPort(addr); err == nil {
		return host
	}
	return addr
}

func (c *Client) deliver(msg []byte) {
	select {
	case c.send <- msg:
	default:
	}
}
