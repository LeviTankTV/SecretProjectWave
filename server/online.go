package main

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// Счётчик «кто сейчас онлайн». Игроки в мире считаются напрямую по
// room.players (авторитетный источник: WS-соединение = в игре). Сидящие в
// меню отмечаются heartbeat-запросами на /api/online (клиент шлёт их раз в
// 15 секунд, пока открыта страница с валидным токеном); запись живёт окно
// onlineWindow и протухает, если вкладку закрыли.
var menuSeen = struct {
	mu   sync.Mutex
	last map[string]time.Time
}{last: make(map[string]time.Time)}

// onlineWindow — сколько секунд после последнего heartbeat считается, что
// игрок всё ещё держит страницу открытой (3 интервала опроса).
const onlineWindow = 45 * time.Second

type onlineResponse struct {
	OK     bool `json:"ok"`
	Total  int  `json:"total"`
	InGame int  `json:"inGame"`
	InMenu int  `json:"inMenu"`
}

// handleOnline — POST {token}: отметить «в меню» и вернуть текущий онлайн.
// total = в игре + в меню (ник в мире и он же в меню считается одним игроком).
func handleOnline(w http.ResponseWriter, r *http.Request) {
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

	// Кто реально в мире прямо сейчас (по никам, не по playerID — heartbeat
	// знает только ник из токена). В волновом режиме игроки сидят в комнатах
	// лобби, поэтому опрашиваем их все + общий мир (legacy).
	inGameNicks := make(map[string]bool)
	collect := func(room *Room) {
		if room == nil {
			return
		}
		room.mu.Lock()
		for _, p := range room.players {
			inGameNicks[p.State.Name] = true
		}
		room.mu.Unlock()
	}
	collect(theRoom)
	lobbyMu.Lock()
	lobbyRooms := make([]*Room, 0, len(lobbies))
	for _, l := range lobbies {
		if l.Room != nil {
			lobbyRooms = append(lobbyRooms, l.Room)
		}
	}
	lobbyMu.Unlock()
	for _, room := range lobbyRooms {
		collect(room)
	}

	now := time.Now()
	menuSeen.mu.Lock()
	menuSeen.last[nick] = now
	inMenu := 0
	for n, t := range menuSeen.last {
		if now.Sub(t) > onlineWindow {
			delete(menuSeen.last, n)
			continue
		}
		if !inGameNicks[n] {
			inMenu++
		}
	}
	menuSeen.mu.Unlock()

	inGame := len(inGameNicks)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(onlineResponse{
		OK:     true,
		Total:  inGame + inMenu,
		InGame: inGame,
		InMenu: inMenu,
	})
}
