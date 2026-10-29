package main

// waves.go - волновой режим 2.0: директор волн вместо зонного спавна мобов.
// Каждое лобби получает собственную комнату (newWaveRoom) с одной ареной.
// Мобы спавнятся пачками до конца волны; после зачистки — пауза и следующая
// волна с более высокими редкостями. Каждые 5 волн — спец-волна (ротация
// swarm/tanks/hunters), каждая 10-я — БОСС-волна лестницы (Hexagon -> Fractal
// -> Chaos -> Dot -> ... -> Origin), каждая 25-я (не кратная 10) — ЭЛИТНАЯ
// волна подвидов (golden horde и др.).
//
// Волны 2.0 (порт «новой реальности»): полная лестница 16 редкостей —
// Dot впервые в пуле на ~16-й волне, Origin как потолок к ~28-60-й.

import (
	"encoding/json"
	"strconv"
	"math"
	"math/rand"
)

// Параметры волнового режима 2.0.
const (
	waveBreakDuration = 3.0  // пауза между обычными волнами
	waveSpecialBreak  = 5.0  // пауза после спец-волны
	waveBossBreak     = 10.0 // пауза после босс-волны (лобби готовится)
	waveStartBreak    = 5.0  // секунд до первой волны после старта
	waveMaxRarity     = rarityChaos // потолок редкости МОБОВ = Chaos (решение игрока:
	// истинные тиры Dot..Origin остаются только у ПЕТАЛЕЙ — крафт)
	waveSpawnNearDist = 350  // мобы появляются не дальше этого от случайного игрока
	waveSpawnSafeDist = 300.0 // ...но не ближе этого к живому игроку (не на голове)
	waveArenaRadius   = 1300 // стартовый радиус круглой арены
	waveArenaMaxR     = 2200 // потолок радиуса (рост +25 за каждые 5 волн)
	waveArenaGrowStep = 25.0
	waveHoldDuration  = 20.0 // hold-слой: сколько секунд даётся на добивание после полного спавна
	waveClearAliveMax = 5    // живых ≤5 при полном спавне — волна сразу завершена
)

// bossEscortVariant — подвид моба свиты Chaos+ босса: 40% элита
// (ring/star/crescent), 10% золотой, 50% обычный.
func bossEscortVariant() string {
	roll := rand.Intn(10)
	switch {
	case roll < 1:
		return "gold"
	case roll < 5:
		vs := []string{"ring", "star", "crescent"}
		return vs[rand.Intn(len(vs))]
	default:
		return ""
	}
}

// bossRarityFor — редкость босса n/10-й босс-волны: лестница Hexagon(10) ->
// Fractal(20) -> Chaos(30), дальше — Chaos+ циклы: босс остаётся Chaos, но
// циклы (каждые 10 волн после 30-й) умножают его статы (bossCycleMult) и
// дают элитную свиту подвидов.
func bossRarityFor(n int) int {
	b := n/10 + 4
	if b > waveMaxRarity {
		b = waveMaxRarity
	}
	return b
}

// bossCycleMult — множитель HP/DMG Chaos-босса в энделгейме: волна 40 — ×1.5,
// 50 — ×2.25, 60 — ×3.375…
func bossCycleMult(n int) float64 {
	if n <= 30 {
		return 1.0
	}
	cycles := (n-30+9)/10 // 40 -> 1, 50 -> 2, ...
	if cycles < 1 {
		cycles = 1
	}
	return math.Pow(1.5, float64(cycles))
}

// waveArenaCenterIn — центр круглой арены выбранной зоны (0-7).
func waveArenaCenterIn(zone int) (float64, float64) {
	if zone < 0 || zone >= len(zones) {
		zone = 0
	}
	z := zones[zone]
	return z.X + z.W/2, z.Y + z.H/2
}

// clampArenaIn прижимает точку (с радиусом тела r) внутрь круглой арены зоны
// (радиус по умолчанию; рост арены волн 2.0 — r.clampArenaR).
func clampArenaIn(zone int, x, y, r float64) (float64, float64) {
	return clampArenaInR(zone, x, y, r, waveArenaRadius)
}

// clampArenaInR — то же с явным радиусом арены.
func clampArenaInR(zone int, x, y, r, arenaR float64) (float64, float64) {
	cx, cy := waveArenaCenterIn(zone)
	dx, dy := x-cx, y-cy
	d := dx*dx + dy*dy
	max := arenaR - r
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

// clampArena / waveArenaCenter — обёртки над ареной этой комнаты. Радиус —
// динамический (растёт с номером волны, поле waveArenaR).
func (r *Room) clampArena(x, y, rad float64) (float64, float64) {
	ar := r.waveArenaR
	if ar <= 0 {
		ar = waveArenaRadius
	}
	return clampArenaInR(r.waveZone, x, y, rad, ar)
}

// waveBreakAfter — передышка после волны (2.0): боссы дают 10с на подготовку
// и сбор дропа, спец-волны 5с, обычные 3с.
func waveBreakAfter(n int, special string) float64 {
	switch special {
	case "boss":
		return waveBossBreak
	case "elite", "swarm", "tanks", "hunters":
		return waveSpecialBreak
	default:
		return waveBreakDuration
	}
}

// arenaRadiusFor — радиус арены на волне n: +25 за каждые 5 волн (кап 2200).
func arenaRadiusFor(n int) float64 {
	r := waveArenaRadius + float64(n/5)*waveArenaGrowStep
	if r > waveArenaMaxR {
		r = waveArenaMaxR
	}
	return r
}

func (r *Room) waveArenaCenter() (float64, float64) {
	return waveArenaCenterIn(r.waveZone)
}

// waveSpecialFor выбирает тип спец-волны для волны n (2.0):
//   - каждая 10-я — БОСС лестницы (Hexagon -> ... -> Origin, bossRarityFor);
//   - каждая 25-я (не кратная 10) — ЭЛИТНАЯ волна подвидов (все мобы с
//     вариантом gold/ring/star/crescent);
//   - каждая 5-я (кроме выше) — ротация swarm/tanks/hunters.
func waveSpecialFor(n int) string {
	if n <= 0 {
		return ""
	}
	if n%10 == 0 {
		return "boss"
	}
	if n%25 == 0 {
		return "elite"
	}
	if n%5 != 0 {
		return ""
	}
	// Ротация swarm/tanks/hunters по СЧЁТЧИКУ 5-х волн, минуя съеденные
	// слоты (босс каждые 10-е, элита каждые 25-е): без счётчика элита на 25-й
	// навсегда вытесняла tanks из ротации (остатки по модулю совпадают).
	seq := 0
	for k := 5; k <= n; k += 5 {
		if k%10 == 0 || k%25 == 0 {
			continue
		}
		seq++
	}
	switch (seq - 1) % 3 {
	case 0:
		return "swarm"
	case 1:
		return "tanks"
	default:
		return "hunters"
	}
}

// waveRarityFor — редкость моба для волны n (2.0): [floor..ceil], веса
// смещены к полу. Полная лестница: Dot (~8) впервые в пуле на волне ~16
// (ceil=9), как пол — с ~32-й; Origin (15) — потолок ceil с волны 28,
// пол — с 60-й.
//
//	n=1  -> 0..1   n=8  -> 2..5    n=16 -> 4..9 (Dot!)
//	n=12 -> 3..7   n=24 -> 6..13   n=28 -> 7..15 (Origin в потолке)
//	n=40 -> 10..15 n=60+ -> 14..15
func waveRarityFor(n int) (floor, ceil int) {
	floor = n / 4
	if floor > waveMaxRarity-1 {
		floor = waveMaxRarity - 1
	}
	ceil = n/2 + 1
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
		if rar >= 8 {
			tname = []string{"dart", "arrow", "darter"}[rand.Intn(3)]
		} else {
			tname = []string{"dart", "arrow"}[rand.Intn(2)]
		}
		if rar < 1 {
			rar = 1
		}
	}
	r.waveToSpawn--
	r.waveSpawned++
	if m, ok := r.spawnWaveMobAt(tname, rar); ok {
		// Свита Chaos+ босса: подвид по bossEscortVariant.
		if r.waveSpecial == "boss" && r.waveNum > 30 {
			if v := bossEscortVariant(); v != "" {
				m.Variant = v
				scaleMobForRarity(&m, m.Rarity)
				for i := range r.mobs {
					if r.mobs[i].ID == m.ID {
						r.mobs[i] = m
						break
					}
				}
			}
		}
		// Элитная волна: каждому мобу — гарантированный подвид (годролл золота ~15%).
		if r.waveSpecial == "elite" {
			if m.Variant == "" {
				if rand.Float64() < 0.15 {
					m.Variant = "gold"
				} else {
					vs := []string{"ring", "star", "crescent"}
					m.Variant = vs[rand.Intn(len(vs))]
				}
				scaleMobForRarity(&m, m.Rarity)
				// В срезе уже лежит копия ДО варианта: переписываем её.
				for i := range r.mobs {
					if r.mobs[i].ID == m.ID {
						r.mobs[i] = m
						break
					}
				}
			}
		}
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
	// Хвост ПРОШЛОЙ волны расчищается ДО спавна (босс спавнится в switch
	// ниже — очистка после него стирала босса; реальный порядок фиксирует
	// баг «боссовые волны без босса»).
	r.clearWaveMobs()
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
		// Охотники истинной лестницы: на высоких волнах в пул входит Darter.
	case "elite":
		// Элитная волна подвидов: обычный объём, но каждый моб — вариант.
		r.waveToSpawn = 10 + n
		r.waveMaxAlive = 12 + n/2
		r.waveSpawnInterval = 1.0
	case "boss":
		// Босс лестницы: Hexagon(10) -> Fractal(20) -> Chaos(30), дальше
		// Chaos+ циклы: тот же Chaos-босс, но stats ×1.5 за цикл и свита
		// мобов с подвидами (энделгейм; истинных тиров у мобов нет).
		r.waveToSpawn = 4 + n
		r.waveMaxAlive = 12
		r.waveSpawnInterval = 1.0
		bossRar := bossRarityFor(n)
		if bossRar > ceil {
			ceil = bossRar
		}
		tname := randMobType(r.waveZone)
		idx := len(r.mobs)
		if m, ok := r.spawnWaveMobAt(tname, bossRar); ok {
			// Chaos+ цикл: множитель статов всем свежезаспавненным (тринити
			// может добавить копий — масштабируем весь диапазон).
			mult := bossCycleMult(n)
			if mult > 1.0 {
				for i := idx; i < len(r.mobs); i++ {
					r.mobs[i].HP *= mult
					r.mobs[i].MaxHP *= mult
					r.mobs[i].DMG *= mult
				}
			}
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
	// Арена растёт с волной (2.0): +25 радиуса за каждые 5 волн, кап 2200.
	r.waveArenaR = arenaRadiusFor(n)
	r.waveSpawnT = 1.0
	r.wavePending = false
	// Wave bar: параметры прогресса спавна и hold-таймера.
	r.waveTotal = r.waveToSpawn
	r.waveSpawned = 0
	r.waveHoldT = waveHoldDuration
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
			// Статистика забега в сообщении gameover: волна + киллы команды.
			r.broadcastWaveGameOver()
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
		r.waveBreakT = waveBreakAfter(r.waveNum, r.waveSpecial)
		r.broadcastWaveState("clear")
		return
	}
	r.waveHoldT -= dt
	if r.waveHoldT <= 0 {
		// Принудительный запуск следующей волны: выжившие мобы убираются.
		r.waveRunning = false
		r.clearWaveMobs()
		r.waveBreakT = waveBreakAfter(r.waveNum, r.waveSpecial)
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
	r.waveKills = 0
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

// broadcastWaveGameOver — gameover со статистикой забега (2.0): достигнутая
// волна, киллы команды и рекорд (рекорд обновляется параллельно в MongoDB).
func (r *Room) broadcastWaveGameOver() {
	m := WaveMsg{Type: "wave", Num: r.waveNum, State: "gameover", Kills: r.waveKills, Best: r.waveNum}
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
