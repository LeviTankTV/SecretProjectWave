package main

// waves.go - волновой режим: директор волн вместо зонного спавна мобов.
// Каждое лобби получает собственную комнату (newWaveRoom) с одной ареной
// (зона 0). Мобы спавнятся пачками до конца волны; после зачистки — пауза
// и следующая волна с более высокими редкостями. Каждые 5 волн — спец-волна,
// каждая 10-я — босс.

import (
	"encoding/json"
	"strconv"
	"math"
	"math/rand"
)

// Параметры волнового режима.
const (
	waveBreakDuration = 3.0   // пауза между волнами
	waveStartBreak    = 5.0  // секунд до первой волны после старта
	waveMaxRarity     = rarityChaos // потолок редкостей волн (пока Chaos; этап 6 поднимет до Origin)
	waveSpawnNearDist = 350  // мобы появляются не дальше этого от случайного игрока
	waveSpawnSafeDist = 300.0 // ...но не ближе этого к живому игроку (не на голове)
	waveArenaRadius   = 1300 // радиус круглой арены (фиксированный, от волны не растёт)
	waveHoldDuration  = 20.0 // hold-слой: сколько секунд даётся на добивание после полного спавна
	waveClearAliveMax = 5    // живых ≤5 при полном спавне — волна сразу завершена
)

// waveArenaCenterIn — центр круглой арены выбранной зоны (0-7).
func waveArenaCenterIn(zone int) (float64, float64) {
	if zone < 0 || zone >= len(zones) {
		zone = 0
	}
	z := zones[zone]
	return z.X + z.W/2, z.Y + z.H/2
}

// clampArenaIn прижимает точку (с радиусом тела r) внутрь круглой арены зоны.
func clampArenaIn(zone int, x, y, r float64) (float64, float64) {
	cx, cy := waveArenaCenterIn(zone)
	dx, dy := x-cx, y-cy
	d := dx*dx + dy*dy
	max := waveArenaRadius - r
	if max < 0 {
		max = 0
	}
	if d <= max*max {
		return x, y
	}
	d = math.Sqrt(d)
	if d == 0 {
		return cx, cy
	}
	return cx + dx/d*max, cy + dy/d*max
}

// clampArena / waveArenaCenter — обёртки над ареной этой комнаты.
func (r *Room) clampArena(x, y, rad float64) (float64, float64) {
	return clampArenaIn(r.waveZone, x, y, rad)
}

func (r *Room) waveArenaCenter() (float64, float64) {
	return waveArenaCenterIn(r.waveZone)
}

// waveSpecialFor выбирает тип спец-волны для волны n.
// Каждая 10-я — босс Fractal; каждая 5-я (кроме 10-й) — ротация спец-волн.
func waveSpecialFor(n int) string {
	if n <= 0 {
		return ""
	}
	if n%10 == 0 {
		return "fractal"
	}
	if n%5 == 0 {
		switch (n / 5) % 3 {
		case 1:
			return "swarm"
		case 2:
			return "tanks"
		default:
			return "hunters"
		}
	}
	return ""
}

// waveRarityFor — редкость моба для волны n: [floor..ceil], веса смещены к полу.
func waveRarityFor(n int) (floor, ceil int) {
	floor = n / 6
	if floor > waveMaxRarity-1 {
		floor = waveMaxRarity - 1
	}
	ceil = n/3 + 1
	if ceil > waveMaxRarity {
		ceil = waveMaxRarity
	}
	if ceil < floor {
		ceil = floor
	}
	return floor, ceil
}

// rollWaveRarity — случайная редкость в [floor..ceil] со смещением к полу
// (нижние редкости частые, верхние — редкие).
func rollWaveRarity(floor, ceil int) int {
	if ceil <= floor {
		return floor
	}
	span := ceil - floor + 1
	// rand^2: квадрат смещает выбор к началу диапазона.
	return floor + int(rand.Float64()*rand.Float64()*float64(span))
}

// spawnWaveMob создаёт одного моба текущей волны рядом со случайным живым
// игроком (или в случайной точке арены, если игроков нет/все мертвы).
// Тип моба — из РЕАЛЬНОГО пула выбранной зоны с её весами (randMobType),
// как в основном мире SecretProject.
func (r *Room) spawnWaveMob() {
	if r.waveToSpawn <= 0 {
		return
	}
	tname := randMobType(r.waveZone)
	floor, ceil := waveRarityFor(r.waveNum)
	rar := rollWaveRarity(floor, ceil)
	// Спец-волны сдвигают редкость/пул.
	switch r.waveSpecial {
	case "swarm":
		tname = []string{"dot", "pebble"}[rand.Intn(2)]
		if rar > 2 {
			rar = 2
		}
	case "tanks":
		tname = []string{"square", "block"}[rand.Intn(2)]
	case "hunters":
		tname = []string{"dart", "arrow"}[rand.Intn(2)]
		if rar < 1 {
			rar = 1
		}
	}
	r.waveToSpawn--
	r.waveSpawned++
	if m, ok := r.spawnWaveMobAt(tname, rar); ok {
		if txt := superSpawnText(r.waveZone, m); txt != "" {
			r.broadcastSystemLocked("super", txt)
		}
	}
}

// spawnWaveMobAt спавнит моба заданного типа/редкости в арене рядом с игроком.
// Моб НИКОГДА не появляется на самом игроке: точка отклоняется, если ближе
// waveSpawnSafeDist к любому живому игроку.
func (r *Room) spawnWaveMobAt(tname string, rar int) (Mob, bool) {
	var x, y float64
	targets := make([]*Player, 0, len(r.players))
	for _, p := range r.players {
		if !p.Dead {
			targets = append(targets, p)
		}
	}
	if len(targets) > 0 {
		t := targets[rand.Intn(len(targets))]
		x = t.State.X + (rand.Float64()*2-1)*waveSpawnNearDist
		y = t.State.Y + (rand.Float64()*2-1)*waveSpawnNearDist
		// Не на игроках: до 8 попыток отодвинуть точку, потом дальняя случайная.
		for attempt := 0; attempt < 8; attempt++ {
			if !r.tooCloseToPlayers(x, y) {
				break
			}
			if attempt >= 7 {
				cx, cy := r.waveArenaCenter()
				a := rand.Float64() * 2 * math.Pi
				d := math.Sqrt(rand.Float64()) * (waveArenaRadius - 250)
				x, y = cx+math.Cos(a)*d, cy+math.Sin(a)*d
				break
			}
			x = t.State.X + (rand.Float64()*2-1)*waveSpawnNearDist*1.5
			y = t.State.Y + (rand.Float64()*2-1)*waveSpawnNearDist*1.5
		}
	} else {
		cx, cy := r.waveArenaCenter()
		a := rand.Float64() * 2 * math.Pi
		d := math.Sqrt(rand.Float64()) * (waveArenaRadius - 200)
		x, y = cx+math.Cos(a)*d, cy+math.Sin(a)*d
	}
	// Круглая арена: моб не выходит за границу.
	x, y = r.clampArena(x, y, 100)
	m := makeAdminMob(tname, rar, r.waveZone, x, y, r.mobIDSeq, "nothing")
	r.mobIDSeq++
	if tname == "trinity" {
		// Тройной организм: раскладываем троих в треугольник (порт родителя).
		r.mobs = append(r.mobs, m)
		r.spawnTrinity(&r.mobs[len(r.mobs)-1])
		return m, true
	}
	r.mobs = append(r.mobs, m)
	return m, true
}

// spawnTrinity (порт родителя) раскладывает трёх членов Trinity в треугольник
// вокруг точки спавна. Сам m становится «вожаком» (GSlot 0); двое других —
// отдельными мобами с общим GroupID и своими картами урона (у каждого свой дроп).
func (r *Room) spawnTrinity(m *Mob) {
	const n = 3
	cx, cy := m.X, m.Y
	sp := m.Radius * 2.7
	base := rand.Float64() * math.Pi * 2
	for s := 0; s < n; s++ {
		a := base + float64(s)*math.Pi*2/float64(n)
		ox := math.Cos(a) * sp
		oy := math.Sin(a) * sp
		if s == 0 {
			m.GSlot = 0
			m.GroupID = m.ID
			m.X, m.Y = cx+ox, cy+oy
			continue
		}
		c := *m
		c.ID = "mob_" + strconv.Itoa(r.mobIDSeq)
		r.mobIDSeq++
		c.GSlot = s
		c.GroupID = m.GroupID
		c.X, c.Y = cx+ox, cy+oy
		c.Damage = nil // каждая единица — самостоятельный моб (карта ленивая)
		r.mobs = append(r.mobs, c)
	}
}

// tooCloseToPlayers — точка ближе waveSpawnSafeDist к живому игроку?
func (r *Room) tooCloseToPlayers(x, y float64) bool {
	minD2 := waveSpawnSafeDist * waveSpawnSafeDist
	for _, p := range r.players {
		if p.Dead {
			continue
		}
		dx, dy := p.State.X-x, p.State.Y-y
		if dx*dx+dy*dy < minD2 {
			return true
		}
	}
	return false
}

// startWave запускает волну n: выбирает спец-тип, параметры спавна и
// оповещает игроков сообщением "wave".
func (r *Room) startWave(n int) {
	r.waveNum = n
	r.waveRunning = true
	r.waveSpecial = waveSpecialFor(n)
	floor, ceil := waveRarityFor(n)
	switch r.waveSpecial {
	case "swarm":
		r.waveToSpawn = 30 + n
		r.waveMaxAlive = 25
		r.waveSpawnInterval = 0.35
	case "tanks":
		r.waveToSpawn = 8 + n/2
		r.waveMaxAlive = 10
		r.waveSpawnInterval = 1.2
		if floor < 1 {
			floor = 1
		}
		if ceil < floor+2 {
			ceil = floor + 2
		}
	case "hunters":
		r.waveToSpawn = 12 + n*2
		r.waveMaxAlive = 14
		r.waveSpawnInterval = 0.8
	case "fractal":
		// Босс: сразу один Fractal-моб + немного обычных мобов зоны.
		r.waveToSpawn = 4 + n
		r.waveMaxAlive = 12
		r.waveSpawnInterval = 1.0
		tname := randMobType(r.waveZone)
		if m, ok := r.spawnWaveMobAt(tname, rarityFractal); ok {
			if txt := superSpawnText(r.waveZone, m); txt != "" {
				r.broadcastSystemLocked("super", txt)
			}
		}
	default:
		r.waveToSpawn = 8 + n*3
		r.waveMaxAlive = 10 + n*2
		r.waveSpawnInterval = 2.0 - 0.08*float64(n)
		if r.waveSpawnInterval < 0.4 {
			r.waveSpawnInterval = 0.4
		}
	}
	r.waveRarFloor, r.waveRarCeil = floor, ceil
	r.waveSpawnT = 1.0
	r.wavePending = false
	// Wave bar: параметры прогресса спавна и hold-таймера.
	r.waveTotal = r.waveToSpawn
	r.waveSpawned = 0
	r.waveHoldT = waveHoldDuration
	// Хвост прошлой волны (после hold-таймера) расчищается.
	r.clearWaveMobs()
	// Мёртвые игроки возвращаются в строй ровно на старте следующей волны.
	for _, p := range r.players {
		if p.Dead {
			r.respawnPlayer(p)
		}
	}
	r.broadcastWaveState("start")
}

// waveAliveCount — живые мобы арены (все мобы волновой комнаты).
func (r *Room) waveAliveCount() int {
	c := 0
	for i := range r.mobs {
		if r.mobs[i].HP > 0 {
			c++
		}
	}
	return c
}

// updateWaves — тик директора волн (вызывается из tick() вместо зонного спавна).
func (r *Room) updateWaves(dt float64) {
	// После вайпа матч СТОИТ: никого не респавним и волн не стартуем,
	// пока хост не нажмёт ► Start (beginWaves).
	if r.waveOver {
		return
	}
	// Вайп: все игроки мертвы (или никого нет после старта) — матч окончен.
	if r.waveRunning || r.waveNum > 0 {
		alive := 0
		for _, p := range r.players {
			if !p.Dead {
				alive++
			}
		}
		if len(r.players) > 0 && alive == 0 {
			r.waveRunning = false
			r.wavePending = false
			r.waveOver = true
			r.broadcastWaveState("gameover")
			// Рекорд «лучшая волна» — в БД каждому участнику.
			for _, p := range r.players {
				nick := p.State.Name
				w := r.waveNum
				go bumpBestWave(nick, w)
			}
			// Арена остаётся как есть; рестарт — только через хоста.
			return
		}
	}

	if !r.waveRunning {
		if len(r.players) == 0 {
			return
		}
		r.waveBreakT -= dt
		if r.waveBreakT <= 0 {
			r.startWave(r.waveNum + 1)
		}
		return
	}

	// Спавн пачками до конца волны.
	if r.waveToSpawn > 0 {
		r.waveSpawnT -= dt
		if r.waveSpawnT <= 0 {
			r.waveSpawnT = r.waveSpawnInterval
			if r.waveAliveCount() < r.waveMaxAlive {
				r.spawnWaveMob()
			}
		}
		return
	}

	// Очередь спавна пуста. Правила wave bar:
	//  - живых ≤ waveClearAliveMax — волна сразу зачищена;
	//  - иначе течёт hold-слой (waveHoldT): дотикал — волна завершается
	//    ПРИНУДИТЕЛЬНО, даже если мобов больше пяти (хвост расчищается).
	alive := r.waveAliveCount()
	if alive <= waveClearAliveMax {
		r.waveRunning = false
		r.waveHoldT = 0
		r.waveBreakT = waveBreakDuration
		r.broadcastWaveState("clear")
		return
	}
	r.waveHoldT -= dt
	if r.waveHoldT <= 0 {
		// Принудительный запуск следующей волны: выжившие мобы убираются.
		r.waveRunning = false
		r.clearWaveMobs()
		r.waveBreakT = waveBreakDuration
		r.broadcastWaveState("clear")
	}
}

// beginWaves — хост нажал «Начать» (или переподключился после старта):
// планируем первую/следующую волну.
func (r *Room) beginWaves() {
	// ВЫЗЫВАЕТСЯ ИЗ HTTP-ГОРУТИНЫ (handleLobbyStart) параллельно игровому
	// тику: без r.mu это гонка на mobs/waveNum/waveBreakT/wavePending.
	r.mu.Lock()
	defer r.mu.Unlock()
	// Уже идёт матч или висит отсчёт до первой волны — повторный клик «Start»
	// не должен сбрасывать таймер и чистить арену.
	if r.waveRunning || r.wavePending {
		return
	}
	// Перезапуск после вайпа/паузы — всегда с волны 1, с чистой ареной.
	r.waveOver = false
	r.waveNum = 0
	r.clearWaveMobs()
	// Мёртвых поднимаем к старту матча.
	for _, p := range r.players {
		if p.Dead {
			r.respawnPlayer(p)
		}
	}
	r.waveBreakT = waveStartBreak
	r.wavePending = true
	r.broadcastWaveState("pending")
}

// clearWaveMobs убирает всех живых мобов (перезапуск матча).
func (r *Room) clearWaveMobs() {
	r.mobs = r.mobs[:0]
	r.gp = nil
}

// broadcastWaveState шлёт всем клиентам комнаты состояние волн.
func (r *Room) broadcastWaveState(state string) {
	breakT := 0.0
	if state == "pending" {
		breakT = r.waveBreakT
	}
	m := WaveMsg{Type: "wave", Num: r.waveNum, Special: r.waveSpecial, State: state, Break: breakT}
	b, err := json.Marshal(m)
	if err != nil {
		return
	}
	for _, c := range r.clients {
		c.deliver(b)
	}
}

// roomWaveStarted — для /api/lobby-info: матч запущен (идёт или вот-вот
// стартует). После вайпа (waveOver) — нет: лобби снова «набирается».
// Читает состояние волн из HTTP-горутины — под r.mu (тик пишет эти поля).
func (r *Room) roomWaveStarted() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.waveOver {
		return false
	}
	return r.waveRunning || r.wavePending || r.waveNum > 0
}
