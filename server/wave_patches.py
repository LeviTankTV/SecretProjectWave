import io

p = "room.go"
s = io.open(p, encoding='utf-8').read()
n0 = len(s)

def rep(old, new, tag):
    global s
    assert old in s, "ANCHOR FAIL: " + tag
    s = s.replace(old, new, 1)
    print("ok:", tag)

# 1) Room struct: волновые поля + stopCh
rep("""	chainDebugMissMin  float64 // ближайший моб при «пустой» цепи (min за окно)
	chainDebugMissInfo string  // тип/редкость того моба
}""",
"""	chainDebugMissMin  float64 // ближайший моб при «пустой» цепи (min за окно)
	chainDebugMissInfo string  // тип/редкость того моба

	// ===== SecretProjectWave: волновой режим (см. waves.go) =====
	waveMode          bool
	waveZone          int   // зона-арена 0-7 (выбирает хост лобби)
	waveRunning       bool
	wavePending       bool  // старт запланирован (отсчёт до первой волны)
	waveOver          bool  // матч завершён (вайп): ждём ► Start от хоста
	waveNum           int
	waveSpecial       string
	waveToSpawn       int
	waveMaxAlive      int
	waveSpawnInterval float64
	waveSpawnT        float64
	waveBreakT        float64
	waveTotal         int     // всего мобов в текущей волне (для прогресс-бара)
	waveSpawned       int     // сколько уже заспавнено
	waveHoldT         float64 // остаток hold-таймера (второй слой волны-бара)
	waveRarFloor      int
	waveRarCeil       int
	stopCh            chan struct{} // закрытие комнаты лобби: остановка gameLoop/autosave
}""", "room struct")

# 2) NewRoom: stopCh + setWaveZone/close/newWaveRoom после NewRoom
rep("""	r.mobs = append(r.mobs, makeShowcaseMobs(testRoomZone)...)
	go r.gameLoop()
	go r.autosaveLoop()
	return r
}""",
"""	r.mobs = append(r.mobs, makeShowcaseMobs(testRoomZone)...)
	r.stopCh = make(chan struct{})
	go r.gameLoop()
	go r.autosaveLoop()
	return r
}

// newWaveRoom — комната лобби для волнового режима: без зонных мобов,
// без мини-боссов и витрины; мобов выдаёт директор волн (waves.go).
func newWaveRoom(tickRate int) *Room {
	initWorld()
	r := &Room{
		players:    make(map[string]*Player),
		clients:    make(map[string]*Client),
		usedIPs:    make(map[string]bool),
		usedNiks:   make(map[string]bool),
		nextID:     1,
		tickRate:   tickRate,
		waveMode:   true,
		waveZone:   0,
		waveBreakT: waveStartBreak,
		stopCh:     make(chan struct{}),
	}
	go r.gameLoop()
	go r.autosaveLoop()
	return r
}

// close останавливает горутины комнаты (лобби удалено).
func (r *Room) close() {
	select {
	case <-r.stopCh:
		// уже закрыт
	default:
		close(r.stopCh)
	}
}

// setWaveZone переносит арену в выбранную хостом зону (0-7).
func (r *Room) setWaveZone(zone int) {
	if zone < 0 || zone >= secondTierOffset {
		zone = 0
	}
	r.mu.Lock()
	r.waveZone = zone
	// Переносим живых игроков в центр новой арены.
	for _, p := range r.players {
		spawnWavePlayer(r, p)
	}
	r.mu.Unlock()
}

// spawnWavePlayer ставит игрока в круглой арене комнаты — в центре (матч
// начинается со сбора всех игроков посередине карты).
func spawnWavePlayer(r *Room, p *Player) {
	cx, cy := r.waveArenaCenter()
	a := rand.Float64() * 2 * math.Pi
	d := rand.Float64() * (waveArenaRadius * 0.1)
	p.State.X = cx + math.Cos(a)*d
	p.State.Y = cy + math.Sin(a)*d
	p.zone = r.waveZone
	p.State.Zone = r.waveZone
}""", "newWaveRoom")

io.open(p, 'w', encoding='utf-8', newline='\n').write(s)
print("room.go bytes:", n0, "->", len(s))
