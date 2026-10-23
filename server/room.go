package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// room.go - игровая комната: серверный движок с мобами, петалями, боёвкой.

// autosaveInterval — период фонового автосейва профилей всех онлайн-игроков
// (защита от жёсткой остановки процесса без дисконнект-сейвов).
const autosaveInterval = 30.0

const (
	portalCooldown         = 12.0
	transferTime           = 2.5
	playerSpeed            = 200.0
	playerSize             = 15.0
	playerMaxHP            = 20.0
	mobHitCooldown         = 0.3   // кулдаун урона мобу от петали
	playerHitCD            = 0.133 // кулдаун урона игроку от моба (в 3 раза быстрее)
	petalCountMax          = 10    // жёсткий максимум (кап по уровню — petalCapForLevel)
	groundPetalPickDist    = 55.0  // радиус подбора петаля с земли
	groundPetalMagnetDist  = 150.0 // с этого расстояния дроп магнитится к игроку
	groundPetalMagnetSpeed = 260.0 // скорость притяжения дропа (мир-единиц/сек)
	groundDropLifetime     = 60.0  // через сколько сек дроп из моба исчезает, если не подобрали
)

type Room struct {
	mu                 sync.Mutex
	players            map[string]*Player
	clients            map[string]*Client
	mobs               []Mob
	minions            []Minion // союзные миньоны (петали Entropy)
	minionIDSeq        int
	proj               []Projectile // летящие снаряды (миссил-blade, осколки-hiveling)
	projIDSeq          int
	orbs               []BeamOrb // энергетические шары моба Beam
	orbIDSeq           int
	chains             []Chain         // визуальные линии-цепи петеля spark
	gp                 []GroundPetal   // петали-предметы на земле
	gpRespawn          map[int]float64 // зона -> таймер до следующего спавна
	mobSpawnTimer      map[int]float64 // зона -> таймер до следующего спавна моба
	hexTimer           map[int]float64 // зона -> таймер до спавна Hexagon-мини-босса (5-15 мин)
	nextID             int
	mobIDSeq           int
	gpIDSeq            int
	tickRate           int
	usedIPs            map[string]bool
	usedNiks           map[string]int // ник -> число подключений (одно подключение = один счётчик)
	chainDebugT        float64
	chainDebugFired    int
	chainDebugEmpty    int
	chainDebugMissMin  float64 // ближайший моб при «пустой» цепи (min за окно)
	chainDebugMissInfo string  // тип/редкость того моба

	// ===== SecretProjectWave: волновой режим (см. waves.go) =====
	waveMode          bool
	waveZone          int // зона-арена 0-7 (выбирает хост лобби)
	waveRunning       bool
	wavePending       bool // старт запланирован (отсчёт до первой волны)
	waveOver          bool // матч завершён (вайп): ждём ► Start от хоста
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
}

func NewRoom(tickRate int) *Room {
	initWorld()
	r := &Room{
		players:       make(map[string]*Player),
		clients:       make(map[string]*Client),
		gpRespawn:     make(map[int]float64),
		mobSpawnTimer: make(map[int]float64),
		hexTimer:      make(map[int]float64),
		usedIPs:       make(map[string]bool),
		usedNiks:      make(map[string]int),
		nextID:        1,
		tickRate:      tickRate,
	}
	// Начальный спавн мобов по всем зонам. Петали на полу НЕ генерируются —
	// они появляются только как дроп с убитых мобов.
	for i := 0; i < len(zones); i++ {
		for n := 0; n < zoneMobLimits[i]; n++ {
			if r.mobsCount(i) < zoneMobLimits[i] {
				r.spawnMob(i)
			}
		}
		// Первый Fractal-босс зоны — через таймер был; теперь Fractal
		// спавнится рандомным роллом 1/1500 в верхней полосе L2 (без таймеров).
		// Hexagon-мини-боссы — таймером зоны: первый через 5-15 минут.
		if zoneRarityAllowed(i, 5) && zoneRarityCeil[i] >= 5 {
			r.hexTimer[i] = hexTimerMin + rand.Float64()*(hexTimerMax-hexTimerMin)
		}
		// Случайный первый таймер моб-спавна (чтобы не спавнились разом).
		r.mobSpawnTimer[i] = rand.Float64()*10 + 5
	}
	// Витрина TestRoom (зона 16): 16 статичных мокапов, создаются один раз.
	r.mobs = append(r.mobs, makeShowcaseMobs(testRoomZone)...)
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
		usedNiks:   make(map[string]int),
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

// close останавливает горутины комнаты (лобби удалено) и закрывает WS всех
// подключённых клиентов: без этого игроки остаются в замершей матче комнате —
// state больше не идёт, а коннекты висят до pong-таймаута.
func (r *Room) close() {
	r.mu.Lock()
	select {
	case <-r.stopCh:
		r.mu.Unlock()
		return // уже закрыт
	default:
		close(r.stopCh)
	}
	clients := make([]*Client, 0, len(r.clients))
	for _, c := range r.clients {
		clients = append(clients, c)
	}
	r.mu.Unlock()
	for _, c := range clients {
		c.conn.Close() // readPump разматывается, RemoveClient дочищает за ним
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
}

// mobsCount сколько живых мобов в зоне.
func (r *Room) mobsCount(zone int) int {
	c := 0
	for _, m := range r.mobs {
		if m.HP > 0 && m.Zone == zone {
			c++
		}
	}
	return c
}

// chaosActive — есть ли живой Chaos-моб где угодно (по всей игре максимум 1).
func (r *Room) chaosActive() bool {
	for i := range r.mobs {
		if r.mobs[i].HP > 0 && r.mobs[i].Rarity == rarityChaos {
			return true
		}
	}
	return false
}

// fractalInZone — есть ли живой Fractal-моб в зоне (максимум 1 на зону).
func (r *Room) fractalInZone(zone int) bool {
	for i := range r.mobs {
		m := &r.mobs[i]
		if m.HP > 0 && m.Zone == zone && m.Rarity == rarityFractal {
			return true
		}
	}
	return false
}

// spawnMob создаёт моба в зоне, добавляет его в мир.
// High-тиры роллятся ТОЛЬКО в верхней полосе L2: Fractal 1/1500, Chaos 1/5000
// (не больше одного живого Chaos на мир). Hexagon роллом НЕ спавнится —
// только таймером зоны (5-15 мин, кап 2, см. spawnHexagonBoss).
func (r *Room) spawnMob(zone int) {
	m := makeMob(zone, r.mobIDSeq)
	r.mobIDSeq++
	if m.Rarity == 5 {
		// Hexagon больше не роллится — оставляем кап-даунгрейд как страховку
		// на случай будущих путей спавна: 5 → Polygon.
		scaleMobForRarity(&m, 4)
	}
	if m.Rarity == rarityChaos && r.chaosActive() {
		// Chaos уже существует — второй не спавнится, понижаем.
		scaleMobForRarity(&m, rarityChaos - 2)
	}
	r.mobs = append(r.mobs, m)
	if m.Type == "trinity" {
		r.spawnTrinity(&r.mobs[len(r.mobs)-1])
	}
	if txt := superSpawnText(zone, m); txt != "" {
		r.broadcastSystemLocked("super", txt)
	}
}

// hexInZone — сколько живых Hexagon-мобов в зоне (мини-боссы, кап 2).
func (r *Room) hexInZone(zone int) int {
	n := 0
	for i := range r.mobs {
		if r.mobs[i].HP > 0 && r.mobs[i].Zone == zone && r.mobs[i].Rarity == 5 {
			n++
		}
	}
	return n
}

// spawnHexagonBoss — спавн Hexagon-мини-босса по таймеру зоны: верхняя 1/4
// зоны Mob-полосы, зональное оповещение (только игрокам ЭТОЙ зоны).
func (r *Room) spawnHexagonBoss(zone int) {
	z := zones[zone]
	m := makeMob(zone, r.mobIDSeq)
	r.mobIDSeq++
	scaleMobForRarity(&m, 5)
	// Верхняя 1/4 полосы (там, где по лестнице живут Hexagon-мобы).
	top := z.Y + m.Radius
	bot := z.Y + (mobZoneBottom(&z)-z.Y)/4
	if bot <= top {
		bot = top + 1
	}
	m.X = z.X + m.Radius + rand.Float64()*(z.W-2*m.Radius)
	m.Y = top + rand.Float64()*(bot-top)
	r.mobs = append(r.mobs, m)
	r.broadcastSystemZoneLocked("super", zone, "A Hexagon "+mobTypes[m.Type].Name+" has spawned!")
}

// spawnFractalBoss спавнит Fractal-босса (редкость Fractal) в верхней 1/8 зоны.
// ЕДИНСТВЕННЫЙ путь появления боссов: таймер зоны (5-30 мин ПОСЛЕ смерти
// предыдущего, максимум 1 в зоне). Обычные спавны Fractal-редкость не рождают.
func (r *Room) spawnFractalBoss(zone int) {
	r.spawnSuperBoss(zone, rarityFractal)
}

// spawnSuperBoss — общий спавн босса Fractal/Chaos заданной редкости
// (позиция: верхняя 1/8 зоны Mob-полосы) + уведомление всем.
func (r *Room) spawnSuperBoss(zone int, rar int) {
	z := zones[zone]
	m := makeMob(zone, r.mobIDSeq)
	r.mobIDSeq++
	scaleMobForRarity(&m, rar)
	// Верхняя полоса зоны: самая редкая редкость — самая опасная зона.
	top := z.Y + m.Radius
	bot := z.Y + (mobZoneBottom(&z)-z.Y)/8
	if bot <= top {
		bot = top + 1
	}
	m.X = z.X + m.Radius + rand.Float64()*(z.W-2*m.Radius)
	m.Y = top + rand.Float64()*(bot-top)
	r.mobs = append(r.mobs, m)
	if txt := superSpawnText(zone, m); txt != "" {
		r.broadcastSystemLocked("super", txt)
	}
}

// superSpawnText возвращает текст уведомления, если заспавнился Fractal (6)
// или Chaos (7) моб. Оба возможны в верхней полосе ЛЮБОЙ зоны — уведомляем всегда.
func superSpawnText(zone int, m Mob) string {
	// Подвид моба: к имени добавляется префикс (и у обычных, и у боссов).
	if m.Rarity >= rarityChaos {
		return "A Chaos " + variantPrefix(m.Variant) + mobTypes[m.Type].Name + " has spawned!"
	}
	if m.Rarity == rarityFractal {
		return "A Fractal " + variantPrefix(m.Variant) + mobTypes[m.Type].Name + " has spawned!"
	}
	return ""
}

func (r *Room) gameLoop() {
	ticker := time.NewTicker(time.Second / time.Duration(r.tickRate))
	defer ticker.Stop()
	for {
		select {
		case <-r.stopCh:
			return
		case <-ticker.C:
		}
		// Паника в тике НЕ валит сервер: логируем стек и продолжаем
		// (потеряется один тик, а не весь игровой процесс).
		func() {
			defer func() {
				if p := recover(); p != nil {
					log.Printf("[PANIC] tick: %v\n%s", p, debug.Stack())
				}
			}()
			r.tick()
		}()
	}
}

// autosaveLoop периодически сохраняет профили онлайн-игроков в фоне, НЕ блокируя
// игровой тик. Раньше автосейв был синхронным внутри tick() под r.mu: при
// подтормаживающей MongoDB запрос уходит до 3с (auth.go saveProfile), и весь
// игровой цикл «замирал» на секунды — игроки видели огромный пинг. Теперь данные
// копируются под локом (быстро, без сетевых вызовов), а запись идёт в отдельной
// горутине. Сейв всё ещё синхронный (disconnect/выход в меню), но тот — точечный.
func (r *Room) autosaveLoop() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	secs := 0
	for {
		select {
		case <-r.stopCh:
			return
		case <-ticker.C:
		}
		secs++
		if secs < int(autosaveInterval) {
			continue
		}
		secs = 0
		// Копируем состояние под локом: только слайс-головы + копии данных.
		r.mu.Lock()
		all := make([]profileSnapshot, 0, len(r.players))
		for _, p := range r.players {
			s := profileSnapshot{
				Name:    p.State.Name,
				Level:   p.State.Level,
				XP:      p.State.XP,
				Inv:     append([]InvStack(nil), p.inventory...),
				Active:  p.activeBar,
				Pending: append([]PetalDef(nil), p.pendingCraft...),
			}
			s.Hotbars[0] = append([]PetalDef(nil), p.hotbars[0]...)
			s.Hotbars[1] = append([]PetalDef(nil), p.hotbars[1]...)
			all = append(all, s)
		}
		r.mu.Unlock()
		// Запись в БД — ВНЕ лока и ОДНИМ bulkWrite (снимки всех игроков),
		// вместо N×2 запросов. Тик продолжает работать, пока Mongo отвечает.
		bulkSaveProfiles(all)
	}
}

func (r *Room) tick() {
	dt := 1.0 / float64(r.tickRate)

	r.mu.Lock()

	// Автосейв профилей вынесен в отдельную горутину autosaveLoop: раньше он
	// синхронно писал в MongoDB ПОД r.mu внутри тика, и при подвисшей БД весь
	// сервер «замирал» (игроки видели огромный пинг).

	for _, p := range r.players {
		r.updatePlayer(p, dt)
	}

	for i := range r.mobs {
		if r.mobs[i].HP > 0 {
			r.updateMob(&r.mobs[i], r.players, dt)
		}
		// Руны (петаль rune) истекают на мобе — сбрасываем стаки.
		if r.mobs[i].RuneStacks > 0 {
			r.mobs[i].RuneT -= dt
			if r.mobs[i].RuneT <= 0 {
				r.mobs[i].RuneStacks = 0
			}
		}
		// «Шиповое тело» игрока (петаль spike): таймер царапания моба.
		if r.mobs[i].SpikeT > 0 {
			r.mobs[i].SpikeT -= dt
		}
		// Замедление (петаль rhombus): истекает — моб снова ходит нормально.
		if r.mobs[i].SlowT > 0 {
			r.mobs[i].SlowT -= dt
			if r.mobs[i].SlowT <= 0 {
				r.mobs[i].SlowT = 0
				r.mobs[i].SlowFrac = 0
			}
		}
	}

	// Способности мобов Chaos (притяжение Black Hole, аура Saturn) — до
	// клампа арены, чтобы тяга не выталкивала за границу (порт родителя).
	r.applyMobChaosAbilities(dt)

	// Волновой режим: директор волн вместо зонного спавна + круглая арена.
	if r.waveMode {
		r.updateWaves(dt)
		for i := range r.mobs {
			if r.mobs[i].HP > 0 {
				r.mobs[i].X, r.mobs[i].Y = r.clampArena(r.mobs[i].X, r.mobs[i].Y, r.mobs[i].Radius)
			}
		}
	} else {
		for i := 0; i < len(zones); i++ {
			r.mobSpawnTimer[i] -= dt
			if r.mobSpawnTimer[i] <= 0 {
				if r.mobsCount(i) < zoneMobLimits[i] {
					r.spawnMob(i)
				}
				r.mobSpawnTimer[i] = 5 + rand.Float64()*10
			}
			// Hexagon-мини-босс (макс 2 на зону): таймер 5-15 минут per L2-зона.
			// Таймер инициализируется только для зон, где редкость 5 разрешена.
			// Пока в зоне кап — таймер ждёт коротким retry (60с).
			if _, has := r.hexTimer[i]; has {
				r.hexTimer[i] -= dt
				if r.hexTimer[i] <= 0 {
					if r.hexInZone(i) < hexPerZoneCap {
						r.spawnHexagonBoss(i)
						r.hexTimer[i] = hexTimerMin + rand.Float64()*(hexTimerMax-hexTimerMin)
					} else {
						r.hexTimer[i] = 60 // кап — ждём освобождения слота
					}
				}
			}
		}

	}
	// Fractal-боссов больше НЕТ по таймеру: они роллятся обычным спавном
	// 1/1500 в верхней полосе L2-зон (см. l2RarityBands / spawnMob).
	// Chaos таймера НЕ имеет: спавнится СТАРЫМ путём — рандомный ролл в самой
	// верхней полосе зон второго уровня (1/5000, см. rarityBands[7]), максимум
	// один живой в мире (кап в spawnMob).

	// Магнит (петаль hex) притягивает дропы к игрокам — до подбора.
	r.applyMagnets(dt)

	// Подбор петалей с земли игроками.
	r.pickupPetals(dt)

	// Дропы из мобов истекают через groundDropLifetime, если их не подобрали,
	// чтобы не копить их бесконечно на земле.
	if len(r.gp) > 0 {
		kept := r.gp[:0]
		for i := range r.gp {
			g := &r.gp[i]
			if g.OwnerID != "" {
				g.Expire -= dt
				if g.Expire <= 0 {
					continue // дроп истёк — выкидываем
				}
			}
			kept = append(kept, *g)
		}
		r.gp = kept
	}

	// Боёвка: петали vs мобы.
	r.fightPetalsVsMobs(dt)
	// Летящие снаряды (миссил-blade, осколки-hiveling).
	r.updateProjectiles(dt)
	// Энергетические шары моба Beam (хоминг до ближайшего игрока).
	r.updateBeamOrbs(dt)
	// Визуальные линии-цепи петеля spark (тик жизни).
	r.updateChains(dt)
	// Диагностика цепи spark (временная): раз в 5с — счёт срабатываний
	// и «пустых» цепей (соседей не нашли), чтобы поймать пропуски в проде.
	r.chainDebugT += dt
	if r.chainDebugT >= 5 {
		if r.chainDebugFired > 0 || r.chainDebugEmpty > 0 {
			extra := ""
			if r.chainDebugMissMin > 0 && r.chainDebugMissMin < 1e18 {
				extra = fmt.Sprintf(", nearestMiss=%.0f (%s)", r.chainDebugMissMin, r.chainDebugMissInfo)
			}
			log.Printf("[chain] 5s: fired=%d empty=%d%s", r.chainDebugFired, r.chainDebugEmpty, extra)
		}
		r.chainDebugT = 0
		r.chainDebugFired = 0
		r.chainDebugEmpty = 0
		r.chainDebugMissMin = 0
		r.chainDebugMissInfo = ""
	}
	// Яд от петалей-«гадюк»: периодический урон отравленным мобам.
	r.applyPoison(dt)
	// Миньоны петалей (роль «стадо»): движение, бой с мобами, респавн.
	r.updateMinions(dt)
	// Тест-режим админа: касание телом ваншотит мобов зоны.
	r.applyBodyOneShots()
	// Боёвка: мобы vs игроки.
	r.fightMobsVsPlayers(dt)
	// Боёвка: орбитальные петали боссов (Fractal/Chaos) против игроков и их петалей.
	r.fightBossPetalsVsPlayers(dt)
	// Тест-ПВП: петали админа вредят другим игрокам и их петалям.
	r.fightPvP(dt)
	// Коллизия мобов между собой и с объектами.
	r.collideMobs()
	r.collideMobsVsPlayers()
	// Дроп-петали коллизий НЕ имеют: мобы и игроки проходят сквозь них,
	// подбираются они только игроком по радиусу (pickupPetals).
	// Коллизия игроков.
	r.collidePlayers()

	// Мёртвые игроки НЕ респавнятся автоматически: только по команде Respawn
	// от клиента (r.RespawnPlayer). До тех пор игрок остаётся мёртвым.

	// Убираем мёртвых мобов из слайса, чтобы не копить их (и таблицы урона/дропы)
	// бесконечно. Живые мобы в state попадают по HP>0.
	alive := r.mobs[:0]
	for i := range r.mobs {
		if r.mobs[i].HP > 0 {
			alive = append(alive, r.mobs[i])
		}
	}
	r.mobs = alive

	// ВАЖНО: лок НЕ отпускаем до конца tick. Ниже мы итерируем r.clients и
	// r.players (buildState + компас), а WS-горутины в это время добавляют/
	// удаляют игроков через AddClient/RemoveClient. Раньше Unlock стоял здесь,
	// и одновременная запись в карту во время итерации роняла сервер фатальной
	// ошибкой «concurrent map iteration and map write».

	// Состояние собирается индивидually для каждого клиента, чтобы приватные
	// дропы (из мобов) были видны и подбираемы только их владельцам.
	for _, c := range r.clients {
		c.deliver(r.buildState(c.playerID))
	}

	// Компас (петаль compass): раз в 1 с игроку ВТОРОГО уровня (зоны 8-15)
	// шлём угол на ближайшего живого моба с Rarity ≥ Fractal в его зоне.
	// Работает, пока компaс жив и игрок во втором тире.
	for _, c := range r.clients {
		p, ok := r.players[c.playerID]
		if !ok || p.Dead || p.zone < secondTierOffset || p.zone >= len(zones) {
			continue
		}
		hasCompass := false
		for i := range p.petals {
			if petalKinds[p.petals[i].Kind].Compass && p.petals[i].HP > 0 && p.petals[i].Recharge <= 0 {
				hasCompass = true
				break
			}
		}
		if !hasCompass {
			continue
		}
		p.compassT -= dt
		if p.compassT > 0 {
			continue
		}
		p.compassT = 1.0
		bestD2 := math.MaxFloat64
		var tx, ty float64
		found := false
		for i := range r.mobs {
			m := &r.mobs[i]
			if m.HP <= 0 || m.Zone != p.zone || m.Rarity < rarityFractal {
				continue
			}
			dx := m.X - p.State.X
			dy := m.Y - p.State.Y
			if d := dx*dx + dy*dy; d < bestD2 {
				bestD2 = d
				tx, ty = m.X, m.Y
				found = true
			}
		}
		msg := CompassMsg{Type: "compass", On: found}
		if found {
			msg.A = math.Atan2(ty-p.State.Y, tx-p.State.X)
			msg.D = math.Sqrt(bestD2)
		}
		if b, err := json.Marshal(msg); err == nil {
			c.deliver(b)
		}
	}

	// Конец критической секции tick — см. комментарий про Unlock выше.
	r.mu.Unlock()
}

func (r *Room) gpCount(zone int) int {
	c := 0
	for _, g := range r.gp {
		// Дропы из мобов (приватные) не считаются: они не мешают респавну
		// обычных петалей мира.
		if g.OwnerID != "" {
			continue
		}
		if g.Zone == zone {
			c++
		}
	}
	return c
}

// buildState собирает состояния игроков, мобов и петалей для конкретного
// клиента (viewerID). Приватные дропы (OwnerID != "") видны только владельцу.
// Мёртвые игроки ПОЛНОСТЬЮ скрываются (и их орбитальные петали тоже).
func (r *Room) buildState(viewerID string) []byte {
	msg := StateMsg{
		Type:    "state",
		Players: make([]PlayerState, 0, len(r.players)),
		Mobs:    make([]MobState, 0, len(r.mobs)),
		Petals:  make([]PetalState, 0),
		Ground:  make([]GroundPetalState, 0, len(r.gp)),
	}

	// Оптимизация Фаза 1: клиент получает только сущности из ЗОНЫ зрителя
	// (сам зритель — всегда). Зоны разделены стенами/лоадингом, поэтому
	// слать весь мир каждому — впустую. viewerZone<0 — зритель не найден,
	// тогда отдаём всё (fallback, чтобы ничего не сломалось).
	viewerZone := -1
	var viewer *Player
	if vp, ok := r.players[viewerID]; ok {
		viewer = vp
		viewerZone = vp.zone
	}

	// Оптимизация Фаза 2: внутри зоны слать только сущности в видимой
	// камере зрителя (прямоугольник, зажатый в границы зоны, как это делает
	// клиент). vx0/vy0/vx1/vy1 — видимый прямоугольник мира с запасом margin.
	// Если размер камеры неизвестен (>0 не пришёл) — срез не применяем.
	const visMargin = 80.0
	hasView := viewer != nil && viewer.viewW > 0 && viewer.viewH > 0 && viewerZone >= 0
	var vx0, vy0, vx1, vy1 float64
	if hasView {
		z := zones[viewerZone]
		cwx := viewer.State.X
		cwy := viewer.State.Y
		if viewer.viewW < z.W {
			cwx = clampf(cwx, z.X+viewer.viewW/2, z.X+z.W-viewer.viewW/2)
		}
		if viewer.viewH < z.H {
			cwy = clampf(cwy, z.Y+viewer.viewH/2, z.Y+z.H-viewer.viewH/2)
		}
		vx0 = cwx - viewer.viewW/2 - visMargin
		vx1 = cwx + viewer.viewW/2 + visMargin
		vy0 = cwy - viewer.viewH/2 - visMargin
		vy1 = cwy + viewer.viewH/2 + visMargin
	}
	visX := func(x float64) bool { return !hasView || (x >= vx0 && x <= vx1) }
	visY := func(y float64) bool { return !hasView || (y >= vy0 && y <= vy1) }
	vis := func(x, y float64) bool { return visX(x) && visY(y) }
	// visR — видимость с учётом РАДИУСА сущности: огромные мобы видны, даже
	// когда их центр за прямоугольником камеры (иначе босс «пропадает»,
	// хотя его тело ещё на экране).
	visR := func(x, y, r float64) bool {
		return !hasView || (x+r >= vx0 && x-r <= vx1 && y+r >= vy0 && y-r <= vy1)
	}

	for _, p := range r.players {
		if p.transferT > 0 || p.Dead {
			continue
		}
		// Невидимый игрок скрыт от всех, кроме самого себя.
		if p.Invisible && p.State.ID != viewerID {
			continue
		}
		// Только игроки из зоны зрителя (плюс сам зритель).
		if viewerZone >= 0 && p.zone != viewerZone && p.State.ID != viewerID {
			continue
		}
		// Внутри зоны — только видимые камерой (сам зритель всегда).
		// Радиус с запасом на орбиту петалей (petalMaxDist = 100).
		if p.State.ID != viewerID && !visR(p.State.X, p.State.Y, p.State.Size+110) {
			continue
		}
		st := p.State
		st.Team = p.Team
		st.ActiveBar = p.activeBar
		if st.ID != viewerID {
			// Другим игрокам не нужен чужой инвентарь/хотбары — только макс.
			// редкость активной линейки (для цвета уровня на клиенте). Это
			// заметно режет размер state-сообщений при нескольких игроках.
			st.Inventory = nil
			st.Hotbars = [2][]PetalDef{}
			hb := p.hotbars[st.ActiveBar&1]
			for i := range hb {
				if int(hb[i].Rarity) > st.MaxHotRar {
					st.MaxHotRar = int(hb[i].Rarity)
				}
			}
		} else {
			st.Inventory = p.inventory
			if st.Inventory == nil {
				st.Inventory = []InvStack{}
			}
			st.Hotbars = p.hotbars
		}
		msg.Players = append(msg.Players, st)
		for _, pet := range p.petals {
			px, py := petalWorldPos(p, pet.Angle, pet.Dist)
			shieldCharge := 0.0
			if petalKinds[pet.Kind].Shield && pet.ShieldChargeT > 0 {
				shieldCharge = 1 - pet.ShieldChargeT/shieldChargeInterval
			}
			msg.Petals = append(msg.Petals, PetalState{
				ID: pet.ID, X: px, Y: py,
				Angle: pet.Angle,
				Size:  pet.Size, Color: pet.Color,
				HP: pet.HP, MaxHP: pet.MaxHP, Rarity: pet.Rarity,
				CD: cdOf(pet), CDMax: pet.CDMax, Kind: pet.Kind,
				Poison:       pet.PoisonT > 0,
				ShieldCharge: shieldCharge,
			})
		}
	}

	for _, m := range r.mobs {
		if m.HP <= 0 {
			continue
		}
		if viewerZone >= 0 && m.Zone != viewerZone {
			continue
		}
		if !visR(m.X, m.Y, m.Radius) {
			continue
		}
		// Подвид: имя с префиксом ("Golden"/"Ring"/...), цвет подвида вместо
		// обычного (у gold — золото).
		mobName := variantPrefix(m.Variant) + mobTypes[m.Type].Name
		mobColor := mobTypes[m.Type].Color
		if vc := variantColor(m.Variant); vc != 0 {
			mobColor = vc
		}
		msg.Mobs = append(msg.Mobs, MobState{
			ID: m.ID, Type: m.Type, Shape: mobTypes[m.Type].Shape,
			Name:   mobName,
			Rarity: rarityNames[m.Rarity], RarityI: m.Rarity,
			X: m.X, Y: m.Y, Size: m.Radius,
			HP: m.HP, MaxHP: m.MaxHP, DMG: m.DMG,
			Color: mobColor, Zone: m.Zone,
			DirX: m.dirX, DirY: m.dirY,
			Speed:    m.Speed,
			Poison:   m.PoisonT > 0,
			Slow:     m.SlowT > 0,
			Showcase: m.Showcase,
			Label:    m.Label,
			Variant:  m.Variant,
		})
		if len(m.petals) > 0 {
			sts := make([]BossPetalState, 0, len(m.petals))
			for i := range m.petals {
				bp := &m.petals[i]
				if bp.HP <= 0 {
					continue // уничтоженная петаль не показывается
				}
				sts = append(sts, BossPetalState{
					Idx: bp.Idx, Angle: bp.Angle, HP: bp.HP, MaxHP: bp.MaxHP, Size: bp.Size, CD: bp.CD,
					Poison: bp.PoisonT > 0,
				})
			}
			if len(sts) > 0 {
				msg.Mobs[len(msg.Mobs)-1].BossPetals = sts
			}
		}
	}

	for _, mn := range r.minions {
		if !mn.Alive {
			continue
		}
		if viewerZone >= 0 && mn.Zone != viewerZone {
			continue
		}
		if !visR(mn.X, mn.Y, mn.Size) {
			continue
		}
		// Миньон шлётся как MobState, но с флагом Ally — клиент рисует его как
		// дружеского моба (зелёная полоска HP, союзное кольцо, свой размер/имя).
		// Чип редкости: усиленные призывы показывают MK-надпись («Square MK-II»,
		// «Pentagon MK-III»), обычные — имя редкости из rarityNames.
		rarLab := rarityNames[mn.Rarity]
		if mn.RarityLabel != "" {
			rarLab = mn.RarityLabel
		}
		msg.Mobs = append(msg.Mobs, MobState{
			ID: mn.ID, Type: mn.Type, Shape: mn.Shape,
			Name:   mn.Name,
			Rarity: rarLab, RarityI: mn.Rarity,
			X: mn.X, Y: mn.Y, Size: mn.Size,
			HP: mn.HP, MaxHP: mn.MaxHP, DMG: mn.DMG,
			Color: mn.Color, Zone: mn.Zone,
			DirX: mn.DirX, DirY: mn.DirY,
			Speed: minionSpeed,
			Ally:  true,
			Owner: mn.OwnerID,
		})
	}

	for _, g := range r.gp {
		// Приватный дроп видят/могут подобрать только его владельцы.
		if g.OwnerID != "" && g.OwnerID != viewerID {
			continue
		}
		if viewerZone >= 0 && g.Zone != viewerZone {
			continue
		}
		if !visR(g.X, g.Y, g.Size) {
			continue
		}
		msg.Ground = append(msg.Ground, GroundPetalState{
			ID: g.ID, Rarity: rarityNames[g.Rarity], RarityI: g.Rarity,
			X: g.X, Y: g.Y, Size: g.Size,
			HP: g.HP, MaxHP: g.MaxHP, Color: g.Color, Zone: g.Zone,
			Kind: g.Kind, Mine: g.OwnerID != "",
		})
	}

	// Летящие снаряды (миссил-blade, осколки-hiveling): только зона зрителя и
	// видимая камера.
	for _, pr := range r.proj {
		if viewerZone >= 0 && pr.Zone != viewerZone {
			continue
		}
		if !vis(pr.X, pr.Y) {
			continue
		}
		msg.Projectiles = append(msg.Projectiles, ProjectileState{
			ID: pr.ID, X: pr.X, Y: pr.Y,
			Angle: math.Atan2(pr.VY, pr.VX),
			Size:  pr.Size, Color: pr.Color, Kind: pr.Kind,
		})
	}

	// Энергетические шары Beam (kind "beamorb"): зона зрителя + камера.
	for _, o := range r.orbs {
		if viewerZone >= 0 && o.Zone != viewerZone {
			continue
		}
		if !vis(o.X, o.Y) {
			continue
		}
		msg.Projectiles = append(msg.Projectiles, ProjectileState{
			ID: o.ID, X: o.X, Y: o.Y,
			Angle: math.Atan2(o.VY, o.VX),
			Size:  o.Size, Color: o.Color, Kind: "beamorb",
		})
	}

	// Линии-цепи петеля spark: только текущая зона зрителя (если она известна).
	for _, ch := range r.chains {
		if viewerZone >= 0 && ch.Zone != viewerZone {
			continue
		}
		msg.Chains = append(msg.Chains, ChainState{Points: ch.Points, Color: ch.Color})
	}

	// Wave bar + ростер живых мобов (волновой режим).
	if r.waveMode {
		switch {
		case r.waveRunning && r.waveToSpawn > 0:
			msg.WavePhase = "spawn"
			if r.waveTotal > 0 {
				msg.WaveF1 = float64(r.waveSpawned) / float64(r.waveTotal)
			}
		case r.waveRunning:
			msg.WavePhase = "hold"
			msg.WaveF1 = 1
			if waveHoldDuration > 0 {
				msg.WaveF2 = 1 - r.waveHoldT/waveHoldDuration
				if msg.WaveF2 < 0 {
					msg.WaveF2 = 0
				}
			}
		}
		msg.WaveNum = r.waveNum
		// Ростер: живые мобы по паре (тип, редкость).
		type rosterKey struct {
			t string
			r int
		}
		counts := map[rosterKey]int{}
		for i := range r.mobs {
			if r.mobs[i].HP > 0 {
				counts[rosterKey{r.mobs[i].Type, r.mobs[i].Rarity}]++
			}
		}
		for k, n := range counts {
			mt := mobTypes[k.t]
			msg.WaveRoster = append(msg.WaveRoster, WaveRosterEntry{
				Type: k.t, Name: mt.Name, Color: mt.Color, Rarity: k.r, Count: n,
			})
		}
	}

	b, _ := json.Marshal(msg)
	return b
}

// clampf ограничивает v в диапазон [lo, hi].
func clampf(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func (r *Room) buildWorld() WorldMsg {
	w := WorldMsg{Type: "world"}
	w.Zones = make([]WorldZoneMsg, 0, len(zones))
	for i, z := range zones {
		// У зон второго уровня Safe Zone и разделительной стены нет.
		safeH, wallTh, wallGap := safeZoneH, zoneWallTh, zoneWallGap
		if !hasSafeZone(&z) {
			safeH, wallTh, wallGap = 0, 0, 0
		}
		zm := WorldZoneMsg{
			Index: z.Index, Biome: z.Biome, Name: z.Name,
			X: z.X, Y: z.Y, W: z.W, H: z.H, Color: z.Color,
			CornerRs: z.crs,
			SafeH:    safeH, WallTh: wallTh, WallGap: wallGap,
			Walls: make([]WorldWallMsg, 0, len(wallLayouts[i])),
		}
		for _, wr := range wallLayouts[i] {
			zm.Walls = append(zm.Walls, WorldWallMsg{X: wr.X, Y: wr.Y, W: wr.W, H: wr.H})
		}
		w.Zones = append(w.Zones, zm)
	}
	// Волновой режим: вместо зон/порталов клиент рисует круглую арену.
	if r.waveMode {
		w.Wave = true
		w.ArenaX, w.ArenaY = r.waveArenaCenter()
		w.ArenaR = waveArenaRadius
		w.ArenaZone = r.waveZone
	}
	w.Portals = make([]WorldPortalMsg, 0, len(portals))
	for _, p := range portals {
		w.Portals = append(w.Portals, WorldPortalMsg{
			ID: p.ID, From: p.From, To: p.To,
			X: p.X, Y: p.Y, W: p.W, H: p.H,
		})
	}
	return w
}

// ---- Обновление игрока ----

func (r *Room) updatePlayer(p *Player, dt float64) {
	// Таймеры перезарядки/восстановления петалей тикают даже когда игрок мёртв,
	// чтобы после респавна петали были готовы (постоянность).
	for i := range p.petals {
		if p.petals[i].Recharge > 0 {
			p.petals[i].Recharge -= dt
			if p.petals[i].Recharge <= 0 {
				p.petals[i].Recharge = 0
				p.petals[i].CDMax = 0              // перезарядка завершена — кулдаун не показывать
				p.petals[i].HP = p.petals[i].MaxHP // восстановилась
			}
		}
	}

	if p.Dead {
		return
	}
	p.noDamageT += dt
	if p.portalCD > 0 {
		p.portalCD -= dt
	}
	if p.mobHitCD > 0 {
		p.mobHitCD -= dt
	}
	if p.invulnT > 0 {
		p.invulnT -= dt
		if p.invulnT < 0 {
			p.invulnT = 0
		}
	}
	if p.transferT > 0 {
		p.transferT -= dt
		if p.transferT <= 0 {
			r.finishTransfer(p)
		}
		return
	}

	// Пассивные механики петалей: реген (orb) и бонус к MaxHP (bulwark).
	r.applyPetalPassives(p, dt)
	// Кольца сатурна (петаль saturn): урон-кольцо-аура вокруг игрока.
	r.applySaturnRings(p, dt)

	// Сдвиг орбиты петалей.
	p.petalDist += p.petalDir * 100 * dt
	if p.petalDist < petalMinDist {
		p.petalDist = petalMinDist
	}
	if p.petalDist > petalMaxDist {
		p.petalDist = petalMaxDist
	}

	// Вращение всей орбиты как единого целого: крутим базовый угол.
	// Петали whirl (Spin) ускоряют вращение, twirl (разворот) меняет направление.
	p.baseAngle += (petalOrbitSpeed + orbitSpinBonus(p)) * orbitDir(p) * dt
	// Глаз (eye) расширяет радиус орбиты ПЛАВНО: ramp-множитель догоняет целевой.
	eyeTarget := eyeOrbitMult(p)
	if p.eyeRamp <= 0 {
		p.eyeRamp = eyeTarget
	}
	p.eyeRamp += (eyeTarget - p.eyeRamp) * clampf(dt*1.2, 0, 1)
	orbMult := p.eyeRamp
	// Аксессуары (eye/line/spike) висят на игроке и НЕ занимают на орбите места:
	// остальные петали раскладываются по кругу плотно, без «дырок».
	var nOrbit int
	for i := range p.petals {
		if !isOrbitAccessory(p.petals[i].Kind) {
			nOrbit++
		}
	}
	if nOrbit == 0 {
		nOrbit = 1
	}
	orbSlot := 0
	for i := range p.petals {
		if isOrbitAccessory(p.petals[i].Kind) {
			p.petals[i].Angle = 0
			p.petals[i].Dist = 0
			continue
		}
		// Равномерное распределение по окружности: baseAngle + i*(360°/N).
		p.petals[i].Angle = p.baseAngle + (2*math.Pi/float64(nOrbit))*float64(orbSlot)
		orbSlot++
		p.petals[i].Dist = p.petalDist * orbMult
	}

	r.movePlayer(p, dt)

	// Волновой режим: круглый «басейн» — прижимаем тело игрока внутрь круга,
	// чекпоинты/порталы не нужны (весь матч идёт в круге).
	if r.waveMode {
		p.State.X, p.State.Y = r.clampArena(p.State.X, p.State.Y, p.State.Size)
	} else {
		// Чекпоинт (версия Т): встав на нижне-центральную площадку сейф-зоны,
		// игрок закрепляет спавнпоинт этой зоны. Только ПЕРВЫЙ уровень (0-7):
		// со второго уровня смерть/релог всегда возвращают в первый.
		// Портал сам по себе чекпоинт НЕ ставит — надо дойти до площадки.
		if p.transferT <= 0 && p.zone < secondTierOffset {
			z := zones[p.zone]
			if p.State.X > z.X+z.W/2-checkpointPadHalfW &&
				p.State.X < z.X+z.W/2+checkpointPadHalfW &&
				p.State.Y > z.Y+z.H-safeZoneH {
				if p.checkpoint != p.zone {
					p.checkpoint = p.zone
					nick := p.State.Name
					cz := p.zone
					go saveCheckpoint(nick, cz)
				}
			}
		}
	}

	if p.portalCD <= 0 && !r.waveMode {
		if portal := portalAt(p.State.X, p.State.Y, p.State.Size); portal != nil {
			p.transferTo = portal.To
			p.transferT = transferTime
			p.portalCD = portalCooldown
			p.input = Vec{X: 0, Y: 0}
			if cl := r.clients[p.State.ID]; cl != nil {
				m := TransferMsg{Type: "transfer", From: portal.From, To: portal.To}
				msg, _ := json.Marshal(m)
				cl.deliver(msg)
			}
		}
	}
}

// applyPetalPassives — пассивные механики активных петалей на орбите:
// реген (orb) лечит игрока % MaxHP/сек, бонус (bulwark) поднимает MaxHP.
// Пересчитывается каждый кадр, поэтому при поломке/снятии петеля MaxHP сразу
// возвращается к базовому, а HP не падает ниже нового капа.
func (r *Room) applyPetalPassives(p *Player, dt float64) {
	base := playerMaxHP
	bonus := 0.0
	regen := 0.0
	regenRar := -1 // orb не стакается: редкость лучшего живого/надетого orb
	shieldCap := 0.0
	// Бафы ПОСТОЯННЫЕ: действуют пока петаль надета в активный хотбар, даже
	// если она сломана/на перезарядке (Regen, MaxHP, ёмкость щита). Полный кап
	// щита считаем ДО цикла по ВСЕМ надетым shield-петали.
	for i := range p.petals {
		pet := &p.petals[i]
		if petalKinds[pet.Kind].Shield {
			shieldCap += shieldCapBase * petalPower[pet.Rarity]
		}
	}
	for i := range p.petals {
		pet := &p.petals[i]
		kn := petalKinds[pet.Kind]
		if kn.MaxHP {
			bonus += base * bulwarkMaxHPFrac * petalPower[pet.Rarity]
		}
		if kn.Regen {
			// НЕ стакается: учитываем только orb максимальной редкости
			// (остальные orb — безделушки).
			if regenRar < pet.Rarity {
				regenRar = pet.Rarity
			}
		}
		if kn.Shield {
			// Заряд щита идёт только от ЖИВОЙ петали (сломанная не заряжает);
			// при ПОЛНОМ щите петаль не тратится: заряд замирает, впитывания
			// и ухода в кулдаун нет (нечего поглощать — не расходуем петаль).
			if pet.HP > 0 && pet.Recharge <= 0 && p.State.Shield < shieldCap {
				pet.ShieldChargeT -= dt
				if pet.ShieldChargeT <= 0 {
					p.State.Shield += shieldChargeBase * petalPower[pet.Rarity]
					if p.State.Shield > shieldCap {
						p.State.Shield = shieldCap
					}
					pet.ShieldChargeT = shieldChargeInterval
					// Петаль «впитывается» в игрока и уходит на перезарядку
					// (cog ускоряет перезарядку всех петалей).
					srch := petalRechargeCD(p, shieldRechargeT)
					pet.Recharge = srch
					pet.CDMax = srch
				}
			}
		}
	}
	p.State.MaxHP = base + bonus
	if p.State.HP > p.State.MaxHP {
		p.State.HP = p.State.MaxHP
	}
	// Реген orb (один, максимальной редкости — без стаков).
	if regenRar >= 0 {
		regen = base * orbRegenFrac * petalPower[minInt(regenRar, len(petalPower)-1)]
	}
	if regen > 0 && p.State.HP < p.State.MaxHP {
		p.State.HP += regen * dt
		if p.State.HP > p.State.MaxHP {
			p.State.HP = p.State.MaxHP
		}
	}
	// Реген вне боя: через outOfCombatDelay секунд без получения урона игрок
	// медленно восстанавливает HP (≈1.5%/с от MaxHP), до полного. Не зависит
	// от орбы и стакается с ней (орба лечит и в бою, эта — только вне боя).
	if p.noDamageT >= outOfCombatDelay && p.State.HP < p.State.MaxHP {
		p.State.HP += p.State.MaxHP * outOfCombatRegenFrac * dt
		if p.State.HP > p.State.MaxHP {
			p.State.HP = p.State.MaxHP
		}
	}
	// Щит: ёмкость из живых shield-петалей; текущий щит не превышает кап.
	p.State.MaxShield = shieldCap
	if p.State.Shield > p.State.MaxShield {
		p.State.Shield = p.State.MaxShield
	}
}

// applySaturnRings — круг-аура сатурна (петаль saturn, макс. 1): вокруг игрока
// ЗАПОЛНЕННЫЙ круг урона (не тонкое кольцо): всех врагов ВНУТРИ радиуса
// saturnRingBase + saturnRingPerRar×редкость — мобов И орбитальные петали
// боссов — тикает saturnDpsBase×petalPower в секунду (по редкости разная
// сила) в Damage-карту игрока. Аура — ЧИСТАЯ зона урона: НИКАКОЙ коллизии и
// отброса (мобов не толкает). Визуал круга — на клиенте. Работает, пока
// петаль жива и не на перезарядке.
func (r *Room) applySaturnRings(p *Player, dt float64) {
	ringR := 0.0
	ringDps := 0.0
	ringRar := 0
	for i := range p.petals {
		pet := &p.petals[i]
		if !petalKinds[pet.Kind].Saturn || pet.HP <= 0 || pet.Recharge > 0 {
			continue
		}
		ringR = saturnRingBase + saturnRingPerRar*float64(pet.Rarity)
		ringDps = saturnDpsBase * petalPower[pet.Rarity]
		ringRar = pet.Rarity
	}
	if ringR <= 0 || ringDps <= 0 {
		return
	}
	dmg := ringDps * dt
	// Мобы: тело пересекает или находится внутри круга — тиковый урон.
	for i := range r.mobs {
		m := &r.mobs[i]
		if m.HP <= 0 || m.Zone != p.zone || m.Showcase {
			continue
		}
		dx := m.X - p.State.X
		dy := m.Y - p.State.Y
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist >= ringR+m.Radius {
			continue // моб целиком за кругом
		}
		// Тик по всей площади круга. БЕЗ отброса.
		// Tier-гейт редкости: Common/Line/Triangle сатурн-петали в 10 раз
		// слабее против мобов редкостью выше своей.
		m.HP -= dmg * petalVsMobMult(ringRar, m.Rarity)
		m.Damage[p.State.ID] += dmg * petalVsMobMult(ringRar, m.Rarity)
		if m.HP <= 0 {
			m.KillerID = p.State.ID
			r.awardMobKill(m)
		}
	}
	// Орбитальные петали боссов (Fractal/Chaos): тоже урон по кругу.
	// Убитая петаль уходит в респавн (как при яде/прямом уроне).
	for i := range r.mobs {
		m := &r.mobs[i]
		if m.HP <= 0 {
			continue
		}
		for j := range m.petals {
			bp := &m.petals[j]
			if bp.HP <= 0 {
				continue
			}
			bx, by := bossPetalPos(m, bp)
			dx := bx - p.State.X
			dy := by - p.State.Y
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist >= ringR+bp.Size {
				continue
			}
			// Орбитальная петаль босса наследует редкость моба — тот же tier-гейт.
			bp.HP -= dmg * petalVsMobMult(ringRar, m.Rarity)
			if bp.HP <= 0 {
				bp.HP = 0
				bp.Respawn = mobPetalsRespawnMin +
					rand.Float64()*(mobPetalsRespawnMax-mobPetalsRespawnMin)
			}
		}
	}
}

// applyMagnets — магнит (петаль hex): притягивает петали-дропы с пола к игроку.
// Радиус растёт с редкостью (hexMagnetBaseRadius + rarity*hexMagnetRadiusPerRar),
// скорость подтягивания постоянна (hexMagnetSpeed). Притягиваются только дропы,
// которые игрок может подобрать (его приватные + публичные, в его зоне).
func (r *Room) applyMagnets(dt float64) {
	for _, p := range r.players {
		if p.Dead || p.transferT > 0 {
			continue
		}
		radius := 0.0
		for i := range p.petals {
			pet := &p.petals[i]
			if pet.Kind == "hex" && pet.HP > 0 && pet.Recharge <= 0 {
				if r := hexMagnetBaseRadius + float64(pet.Rarity)*hexMagnetRadiusPerRar; r > radius {
					radius = r
				}
			}
		}
		if radius <= 0 {
			continue
		}
		for gi := range r.gp {
			g := &r.gp[gi]
			if g.Zone != p.State.Zone {
				continue
			}
			if g.OwnerID != "" && g.OwnerID != p.State.ID {
				continue
			}
			dx := p.State.X - g.X
			dy := p.State.Y - g.Y
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist >= radius || dist < 1 {
				continue
			}
			step := hexMagnetSpeed * dt
			if step > dist {
				step = dist
			}
			g.X += (dx / dist) * step
			g.Y += (dy / dist) * step
		}
	}
}

// pickupPetals - подбор петалей с земли: игрок проходит по петали, она
// становится его орбитальной петалей (заменяя самую слабую, если слоты полны).
// Дропы в небольшом радиусе магнитятся к игроку — подбор становится надёжным.
func (r *Room) pickupPetals(dt float64) {
	for _, p := range r.players {
		if p.Dead || p.transferT > 0 {
			continue
		}
		for gi := 0; gi < len(r.gp); gi++ {
			g := &r.gp[gi]
			if g.Zone != p.State.Zone {
				continue
			}
			// Приватный дроп (из моба) может подобрать только его владелец.
			if g.OwnerID != "" && g.OwnerID != p.State.ID {
				continue
			}
			dx := g.X - p.State.X
			dy := g.Y - p.State.Y
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist < groundPetalPickDist {
				// Добавляем петаль к игроку.
				r.addPetalToPlayer(p, *g)
				// Всемирный анонс подбора Fractal/Chaos-дропа (как анонсы крафта
				// высокого тира: kind "craftX" красиво подсвечивается на клиенте).
				// Только настоящий мобный дроп (OwnerID != ""); редкость Fractal и выше.
				if g.Rarity >= rarityFractal && g.OwnerID != "" {
					petalName := petalKinds[g.Kind].Name
					if petalName == "" {
						petalName = g.Kind
					}
					rarWord := rarityNames[g.Rarity]
					r.broadcastSystemLocked("craft"+rarWord, p.State.Name+
						" has found a "+rarWord+" "+petalName+"!")
				}
				// Фидбек владельцу: «+ вид редкость» (тост подбора).
				if cl := r.clients[p.State.ID]; cl != nil {
					m, _ := json.Marshal(map[string]interface{}{
						"type": "pickedUp", "kind": g.Kind, "rarity": rarityNames[g.Rarity],
					})
					cl.deliver(m)
				}
				// Убираем с земли: swap-remove (O(1) вместо сдвига всего хвоста).
				// Порядок дропов не важен; заново сканируем подменённый элемент.
				r.gp[gi] = r.gp[len(r.gp)-1]
				r.gp = r.gp[:len(r.gp)-1]
				gi--
				break
			}
			// Магнит: подтягиваем дроп к игроку, чтобы «пройти рядом» засчитывалось.
			if dist < groundPetalMagnetDist && dist > 1 {
				pull := groundPetalMagnetSpeed * dt
				if pull > dist-groundPetalPickDist*0.5 {
					pull = dist - groundPetalPickDist*0.5
				}
				if pull > 0 {
					g.X -= dx / dist * pull
					g.Y -= dy / dist * pull
				}
			}
		}
	}
}

// addPetalToPlayer: поднятая с земли петаль попадает в ИНВЕНТАРЬ игрока как стак
// (одинаковые по виду+редкости складываются). Если в активном хотбаре есть место —
// один предмет из стака автоматически ставится на орбиту (неподтверждённый запас
// тратится). Орбита автозаполняется до капа уровня.
func (r *Room) addPetalToPlayer(p *Player, g GroundPetal) {
	kindName := g.Kind
	if kindName == "" {
		kindName = "oval"
	}
	kn := petalKinds[kindName]
	def := PetalDef{
		ID:     "st_" + p.State.ID + "_" + strconv.Itoa(rand.Int()),
		Kind:   kindName,
		Rarity: g.Rarity,
		HP:     g.HP,
		MaxHP:  g.HP,
		DMG:    g.DMG,
		Size:   g.Size,
		Color:  g.Color,
		Mass:   petalDefMass(g.Rarity),
		Name:   kn.Name,
	}
	// 1) Петаль копится в инвентаре (стак по виду+редкости).
	r.addToInventory(p, def)

	// 2) Подобранную петаль ставим на свободный слот хотбара (если место есть).
	//    Сначала смотрим активную линейку, затем вторую. Пустые слоты заполняются.
	r.placePickedPetal(p, def)
}

// placePickedPetal ставит подобранный предмет на первый пустой слот хотбара.
// Сначала в активную линейку; если она полна — во вторую. Затем пересобирает орбиту.
func (r *Room) placePickedPetal(p *Player, def PetalDef) {
	bar := -1
	for _, b := range []int{p.activeBar, 1 - p.activeBar} {
		if slot := firstEmptySlot(p, b); slot >= 0 {
			bar = b
			break
		}
	}
	if bar < 0 {
		// Обе линейки полны — петаль остаётся только в инвентаре.
		return
	}
	r.takeOneFromInventory(p, def.Kind, def.Rarity)
	eq := def // своя копия с уникальным ID для слота хотбара
	eq.ID = "hv_" + p.State.ID + "_" + strconv.Itoa(rand.Int())
	r.setHotbarAt(p, bar, firstEmptySlot(p, bar), &eq)
	r.rebuildOrbit(p)
}

// firstEmptySlot возвращает индекс первого пустого слота хотбара, или -1.
// Учитывает «дырки» (слоты, ставшие пустыми после снятия/перестановки).
func firstEmptySlot(p *Player, bar int) int {
	if bar < 0 || bar > 1 {
		return -1
	}
	hb := p.hotbars[bar]
	// Сначала ищем дырку в уже занятой ёмкости.
	for i := range hb {
		if hb[i].ID == "" {
			return i
		}
	}
	// Иначе — новый слот в конце, если ещё не упёрлись в кап.
	if len(hb) < petalCapForLevel(p.State.Level) {
		return len(hb)
	}
	return -1
}

// addToInventory кладёт предмет в инвентарь-стак. Если стак с таким (вид, редкость)
// уже есть — увеличивает счётчик; иначе добавляет новый стак. После этого сортирует.
func (r *Room) addToInventory(p *Player, def PetalDef) {
	if i := r.findStack(p, def.Kind, def.Rarity); i >= 0 {
		p.inventory[i].Count++
		sortInventory(p)
		return
	}
	s := InvStack{PetalDef: def, Count: 1}
	p.inventory = append(p.inventory, s)
	sortInventory(p)
}

// takeOneFromInventory отнимает один предмет из стака (вид, редкость).
// Если счётчик стал 0 — удаляет стак.
func (r *Room) takeOneFromInventory(p *Player, kind string, rarity int) {
	i := r.findStack(p, kind, rarity)
	if i < 0 {
		return
	}
	if p.inventory[i].Count > 1 {
		p.inventory[i].Count--
		return
	}
	p.inventory = append(p.inventory[:i], p.inventory[i+1:]...)
}

// findStack возвращает индекс стака в инвентаре по (вид, редкость), или -1.
func (r *Room) findStack(p *Player, kind string, rarity int) int {
	for i := range p.inventory {
		if p.inventory[i].Kind == kind && p.inventory[i].Rarity == rarity {
			return i
		}
	}
	return -1
}

// sortInventory сортирует инвентарь по редкости: выше редкость — раньше.
func sortInventory(p *Player) {
	sort.SliceStable(p.inventory, func(a, b int) bool {
		return p.inventory[a].Rarity > p.inventory[b].Rarity
	})
}

// autoEquipFromInventory заполняет активный хотбар из запаса (по одному из стаков)
// до капа уровня. Возвращает true, если хотбар стал непустым.
func (r *Room) autoEquipFromInventory(p *Player) {
	cap := petalCapForLevel(p.State.Level)
	for {
		slot := firstEmptySlot(p, p.activeBar)
		if slot < 0 || slot >= cap {
			break
		}
		i := r.pickStackForEquip(p)
		if i < 0 {
			break
		}
		st := p.inventory[i]
		r.takeOneFromInventory(p, st.Kind, st.Rarity)
		eq := st.PetalDef
		eq.ID = "hv_" + p.State.ID + "_" + strconv.Itoa(rand.Int())
		r.setHotbarAt(p, p.activeBar, slot, &eq)
	}
	if !p.Dead {
		r.rebuildOrbit(p)
	}
}

// pickStackForEquip выбирает первый непустой стак (после сортировки это высшая редкость).
func (r *Room) pickStackForEquip(p *Player) int {
	for i := range p.inventory {
		if p.inventory[i].Count > 0 {
			return i
		}
	}
	return -1
}

// forceIntoInventory кладёт предмет напрямую в запас (для тестов/добавления наград).
func (r *Room) forceIntoInventory(p *Player, kind string, rarity int) {
	kn := petalKinds[kind]
	dmg := kn.BaseDMG * petalPower[rarity]
	hp := kn.BaseHP * petalPower[rarity]
	def := PetalDef{
		ID:     "st_" + p.State.ID + "_" + strconv.Itoa(rand.Int()),
		Kind:   kind,
		Rarity: rarity,
		HP:     hp,
		MaxHP:  hp,
		DMG:    dmg,
		Size:   petalSizeBase,
		Color:  groundPetalColor(rarity),
		Mass:   petalDefMass(rarity),
		Name:   kn.Name,
	}
	r.addToInventory(p, def)
}

// rebuildOrbit пересобирает боевую орбиту (p.petals) из активного хотбара.
func (r *Room) rebuildOrbit(p *Player) {
	bar := p.hotbars[p.activeBar]
	// Сохраняем ЖИВОЕ состояние петалей (текущий HP/износ, перезарядку,
	// кулдауны, заряд щита) по устойчивому ID предмета (OwnerKey): перестановка
	// в хотбаре не должна лечить всех и не должна сбрасывать кулдауны.
	prev := make(map[string]Petal, len(p.petals))
	for _, pet := range p.petals {
		prev[pet.OwnerKey] = pet
	}
	petals := make([]Petal, 0, len(bar))
	for i, d := range bar {
		if d.ID == "" {
			continue // пустой слот — петеля на орбите нет
		}
		if old, ok := prev[d.ID]; ok {
			// Петаль уже была на орбите — сохраняем её состояние, обновляем
			// только слот (ID на орбите), угол и дистанцию.
			old.ID = "pet_" + p.State.ID + "_h" + strconv.Itoa(i)
			old.Angle = p.baseAngle
			old.Dist = p.petalDist
			petals = append(petals, old)
			continue
		}
		petals = append(petals, Petal{
			ID:       "pet_" + p.State.ID + "_h" + strconv.Itoa(i),
			OwnerKey: d.ID, // устойчивый ID предмета — для миньонов (роль «стадо»)
			Angle:    p.baseAngle,
			Dist:     p.petalDist,
			HP:       d.HP,
			MaxHP:    d.HP,
			DMG:      d.DMG,
			Mass:     d.Mass,
			BaseDist: petalBaseDist,
			Rarity:   d.Rarity,
			Color:    d.Color,
			Size:     d.Size,
			Kind:     d.Kind,
			// Щит (shield): первый заряд через ~секунду после появления на орбите.
			ShieldChargeT: shieldChargeInterval,
			// Атака: Charging=false (ноль-значение) = «стратегически готова» —
			// свежая петаль бьёт при первом касании, КД идёт после удара.
		})
	}
	p.petals = petals
}

func (r *Room) finishTransfer(p *Player) {
	// Выход из зоны: мобы полностью забывают этого игрока (урон из Damage-карты,
	// групповое агро) — иначе «память агро» тянет их к игроку в кармане у портала.
	r.resetMobAggroAgainst(p)
	target := p.transferTo
	source := p.zone
	p.transferTo = -1
	r.placeAtPortal(p, target, source)
	// После выхода из портала — 3 секунды бессмертия.
	p.invulnT = respawnInvuln
	p.noDamageT = 0
}

// resetMobAggroAgainst полностью сбрасывает агро всех мобов на игрока:
// он умер или ушёл из зоны. Моб «забывает» нанесённый игроком урон
// (Damage-карта — важно для кармана у портала и Peaceful-мобов вроде Prism),
// сбрасывает групповую агрессию и возвращается в патруль.
func (r *Room) resetMobAggroAgainst(p *Player) {
	for i := range r.mobs {
		m := &r.mobs[i]
		if m.Damage[p.State.ID] > 0 {
			delete(m.Damage, p.State.ID)
			if m.state == "chase" {
				m.state = "patrol"
				m.moveTimer = 1 + rand.Float64()*3
				m.dirX = rand.Float64()*2 - 1
				m.dirY = rand.Float64()*2 - 1
			}
		}
		if m.aggroTo == p.State.ID {
			m.aggroTo = ""
			m.aggroT = 0
		}
	}
}

// placeAtPortal ставит игрока в Safe Zone новой зоны, рядом с порталом, через
// который он вышел (портал в целевой зоне, ведущий обратно в исходную).
func (r *Room) placeAtPortal(p *Player, target, source int) {
	z := zones[target]
	// Порталы парные: в целевой зоне есть портал, ведущий обратно в source.
	for _, pt := range portals {
		if pt.From == target && pt.To == source {
			p.zone = target
			p.State.Zone = target
			// Смещение внутрь зоны, перпендикулярно порталу:
			// боковые порталы (слева/справа) — вбок; порталы «второй уровень»
			// по центру сверху/снизу — вниз/вверх внутрь зоны.
			// Вертикальные порталы высотой vertPortalH: 160 ≈ vertPortalH/2 + запас,
			// чтобы игрок появлялся ЗА пределами прямоугольника обратного портала
			// и его сразу не «затягивало» обратно после окончания portalCooldown.
			dx, dy := 0.0, 0.0
			if pt.X < z.X+z.W*0.25 {
				dx = 90
			} else if pt.X > z.X+z.W*0.75 {
				dx = -90
			} else {
				if pt.Y < z.Y+z.H/2 {
					dy = 160
				} else {
					dy = -160
				}
			}
			p.State.X = pt.X + dx
			p.State.Y = pt.Y + dy
			p.State.X, p.State.Y = clampPlayerInZone(target, p.State.X, p.State.Y, p.State.Size)
			return
		}
	}
	// Фоллбек — в центр Safe Zone.
	r.placeInZone(p, target)
}

func (r *Room) movePlayer(p *Player, dt float64) {
	d := p.input
	if d.X == 0 && d.Y == 0 {
		// Нет ввода — плавная остановка (децелерация).
		p.velX -= p.velX * playerMoveAccel * dt
		p.velY -= p.velY * playerMoveAccel * dt
		if math.Abs(p.velX) < 1 && math.Abs(p.velY) < 1 {
			p.velX, p.velY = 0, 0
		}
	} else {
		length := math.Sqrt(d.X*d.X + d.Y*d.Y)
		dirX := d.X / length
		dirY := d.Y / length
		p.lastDir = Vec{X: dirX, Y: dirY}
		// Плавно ускоряемся к целевой скорости (сглаживает развороты).
		mult := moveSpeedMult(p)
		tx := dirX * p.speed * mult
		ty := dirY * p.speed * mult
		p.velX += (tx - p.velX) * playerMoveAccel * dt
		p.velY += (ty - p.velY) * playerMoveAccel * dt
	}
	p.State.X += p.velX * dt
	p.State.Y += p.velY * dt
	ix, iy := p.State.X, p.State.Y
	p.State.X, p.State.Y = clampPlayerInZone(p.zone, p.State.X, p.State.Y, p.State.Size)
	// Slimy-стены: плавный push от стен. Копится ТОЛЬКО при реальном вдавливании
	// в стену (кламп удерживает игрока, а ввод направлен В стену): быстрое
	// касание стены свободно, а длительное удержание через slimeWallRampTime
	// секунд наращивает push до максимума — стена выталкивает игрока в моб-зону
	// (зажал движение в стену — афк-позиция у стены невозможна).
	ox, oy := p.State.X-ix, p.State.Y-iy
	pressing := (ox != 0 || oy != 0) && (d.X != 0 || d.Y != 0) && (ox*d.X+oy*d.Y) < 0
	if pressing {
		p.wallHold += dt
		if p.wallHold > slimeWallRampTime {
			p.wallHold = slimeWallRampTime
		}
	} else {
		p.wallHold -= slimeWallDecayRate * dt
		if p.wallHold < 0 {
			p.wallHold = 0
		}
	}
	ramp := p.wallHold / slimeWallRampTime
	if ramp > 0 {
		spd := p.speed * moveSpeedMult(p)
		dxS, dyS := wallSlimePush(p.zone, p.State.X, p.State.Y, spd, dt)
		if dxS != 0 || dyS != 0 {
			p.State.X += dxS * ramp
			p.State.Y += dyS * ramp
			p.State.X, p.State.Y = clampPlayerInZone(p.zone, p.State.X, p.State.Y, p.State.Size)
		}
	}
}

func (r *Room) UpdateInput(id string, x, y, viewW, viewH float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p, ok := r.players[id]; ok {
		p.input = Vec{X: x, Y: y}
		if viewW > 0 && viewH > 0 {
			p.viewW = viewW
			p.viewH = viewH
		}
	}
}

func (r *Room) placeInZone(p *Player, zoneIdx int) {
	z := zones[zoneIdx]
	p.zone = zoneIdx
	p.State.Zone = zoneIdx
	p.State.X = z.X + z.W/2
	p.State.Y = z.Y + z.H/2
}

// ---- Боёвка: петали vs мобы ----

// applyBodyOneShots убивает мобов касанием тела в админ-режиме ваншота.
func (r *Room) applyBodyOneShots() {
	for _, p := range r.players {
		if !p.OneShot || p.Dead || p.Passive {
			continue
		}
		for i := range r.mobs {
			m := &r.mobs[i]
			if m.HP <= 0 || m.Zone != p.State.Zone || m.Showcase {
				continue
			}
			dx := m.X - p.State.X
			dy := m.Y - p.State.Y
			if dx*dx+dy*dy < (m.Radius+p.State.Size)*(m.Radius+p.State.Size) {
				m.HP = 0
				m.KillerID = p.State.ID
				r.awardMobKill(m)
			}
		}
	}
}

func (r *Room) fightPetalsVsMobs(dt float64) {
	for _, p := range r.players {
		if p.Dead || p.transferT > 0 {
			continue
		}
		if p.Passive {
			continue // пассивный админ не наносит урона мобам
		}
		if !playerHostileToMob(p.Team) {
			continue // команда 2 (на стороне мобов) не бьёт мобов
		}
		for pi := 0; pi < len(p.petals); pi++ {
			pet := &p.petals[pi]
			// Аксессуары (eye/line/spike) не участвуют в бою и не могут быть
			// повреждены: без кулдаунов, без износа, вечны надет.
			if isOrbitAccessory(pet.Kind) {
				continue
			}
			// Петаль на перезарядке/восстановлении — не атакует.
			if pet.Recharge > 0 || pet.HP <= 0 {
				continue
			}
			// Дробная атака: ready (Charging=false) → удар сразу; после удара —
			// зарядка темпом dt/per-kind КД (cog влияет на ПЕРЕЗАРЯДКУ, не сюда).
			// AtkT ≥ 1 → берст до 3 ударов за тик (низкий КД = больше ударов/тик).
			atkCD := petalAtkCD(pet.Kind)
			strikes := 1
			if pet.Charging {
				if atkCD > 0 {
					pet.AtkT += dt / atkCD
					if pet.AtkT > 3 {
						pet.AtkT = 3
					}
					if pet.AtkT < 1 {
						continue // ещё заряжается
					}
				}
				strikes = int(pet.AtkT)
				if strikes > 3 {
					strikes = 3
				}
				pet.AtkT -= float64(strikes)
			} else {
				// Готова к удару прямо сейчас (свежая или заряд окончен).
				if atkCD > 0 {
					pet.Charging = true
					pet.AtkT = 0
				}
			}
			px, py := petalWorldPos(p, pet.Angle, pet.Dist)
			doPoison := petalKinds[pet.Kind].Poison

			for mi := range r.mobs {
				m := &r.mobs[mi]
				if m.HP <= 0 || m.Zone != p.State.Zone || m.Showcase {
					continue
				}
				dx := m.X - px
				dy := m.Y - py
				dist := math.Sqrt(dx*dx + dy*dy)
				if dist < m.Radius+pet.Size {
					// Берст-урон: strikes = сколько ударов «накопилось» за тик
					// (низкий КД + cog → 2-3 удара в тик — лучший DPS игры).
					// Общий множитель урона всех петалей ×petalDmgMult.
					baseDmg := pet.DMG * petalDmgMult * float64(strikes)
					// Tier-гейт редкости: Common/Line/Triangle петали в 10 раз
					// слабее против мобов редкостью выше своей.
					dmg := baseDmg * petalVsMobMult(pet.Rarity, m.Rarity)
					// Учитываем урон мобу от этого игрока (для дропа по топ-дамагерам).
					m.HP -= dmg
					m.Damage[p.State.ID] += dmg
					// Лайфстил (oval): лечит игрока от нанесённого урона.
					if petalKinds[pet.Kind].Lifesteal {
						p.State.HP += dmg * lifestealPct
						if p.State.HP > p.State.MaxHP {
							p.State.HP = p.State.MaxHP
						}
					}
					// Ядовитый петаль вешает/освежает DoT на мобе (сила по берсту).
					// Tier-гейт: на мобе редкости ВЫШЕ петаля яд ослаблен
					// ×poisonCrossTierMult — square-Viper не выносит Polygon-моба.
					if doPoison {
						pps := dmg * poisonDpsMult
						if m.Rarity > pet.Rarity {
							pps *= poisonCrossTierMult
						}
						poisonMob(m, pps, p.State.ID)
					}
					// Замедление (rhombus): сработает только если редкость моба
					// <= редкости петеля (Fractal Rhombus не замедлит Chaos, но
					// замедлит Fractal и ниже). Свежий удар обновляет таймер.
					if pSlow := petalKinds[pet.Kind].Slow; pSlow && m.Rarity <= pet.Rarity {
						m.SlowFrac = rhombusSlowPct
						m.SlowT += rhombusDur * float64(strikes)
						if m.SlowT > rhombusMaxT {
							m.SlowT = rhombusMaxT
						}
					}
					// Ударили кого-то из стаи Pentagon/Shield — вся ближайшая
					// группа бросается на обидчика.
					r.aggroPack(m, p.State.ID)
					// Износ петеля (за КАЖДЫЙ удар берста); камень — вдвое меньше.
					wear := m.DMG * 0.3 * float64(strikes)
					if petalKinds[pet.Kind].Armor {
						wear *= armorWearMult
					}
					pet.HP -= wear
					// Цепь (spark): после удара бьёт ближайших врагов с визуальной
					// линией-цепью.
					if petalKinds[pet.Kind].Chain {
						r.sparkChain(p.State.ID, m.Zone, m.X, m.Y, dmg, m.ID)
					}
					// Шипы («размен»): моб «сам себя колет» возвращённым износом.
					// Урон идёт игроку в Damage-карту (топ-дамагер/дроп).
					if petalKinds[pet.Kind].Thorns {
						ref := m.DMG * 0.3 * thornsReflectRatio * float64(strikes)
						m.HP -= ref
						m.Damage[p.State.ID] += ref
					}
					// Бомба (ball): мини-АОЕ вокруг точки удара.
					if petalKinds[pet.Kind].Bomb {
						r.bombSplash(p.State.ID, m.Zone, m.X, m.Y, dmg, m.ID)
					}
					// Рана (shard): короткий DoT на цели.
					if petalKinds[pet.Kind].Wound {
						poisonMob(m, dmg*woundDpsMult, p.State.ID)
						m.PoisonT = woundDuration
					}
					// Самоподобие (fractalite): шанс двойного удара (на берст).
					if petalKinds[pet.Kind].Twin && rand.Float64() < twinChance {
						m.HP -= dmg
						m.Damage[p.State.ID] += dmg
					}
					// Руна (rune): удар вешает руну на моба (за каждый удар
					// берста); runeStackNeed стаков → взрыв AoE (runeBoom).
					if petalKinds[pet.Kind].Rune {
						m.RuneStacks += strikes
						m.RuneT = runeStackDur
						if m.RuneStacks >= runeStackNeed {
							m.RuneStacks = 0
							r.runeBoom(p.State.ID, m, dmg)
						}
					}
					// Кулдаун удара учтён в pet.AtkT (дробная накопительная семантика,
					// see луп: доля AtkT накоплена ДО и сброшена ПОСЛЕ удара).
					// Петаль "сломалась" — уходит в ПЕРЕЗАРЯДКУ (cog ускоряет восстановление
					// всех петалей: petalRechargeCD), а НЕ исчезает.
					if pet.HP <= 0 {
						pet.HP = 0
						rch := petalRechargeCD(p, petalRechargeT)
						pet.Recharge = rch
						pet.CDMax = rch
						if petalKinds[pet.Kind].Split {
							r.spawnHivelingShards(p, pet)
						}
						// Слеза (tear): петаль, сломанная мобом, лечит игрока.
						if petalKinds[pet.Kind].Tear {
							p.State.HP += p.State.MaxHP * tearHealFrac(pet.Rarity)
							if p.State.HP > p.State.MaxHP {
								p.State.HP = p.State.MaxHP
							}
						}
					}
					// Hive: каждые 15% HP спавнит случайного моба на 1 редкость ниже.
					if m.Type == "hive" && m.HP > 0 && m.HP <= m.hiveNextSpawn {
						m.hiveNextSpawn -= m.MaxHP * 0.15
						hiveSummonMob(r, m)
					}
					// Моб убит — распределяем награду и дроп.
					if m.HP <= 0 {
						m.KillerID = p.State.ID
						r.awardMobKill(m)
					}
					// Неподвижные мобы (Speed=0: Hive, Block) не отлетают от удара.
					if m.Speed > 0 && dist > 0 {
						// Отброс по массам: лёгкий моб улетает, тяжёлый (редкий /
						// крупный) почти не сдвигается. Чем тяжелее петаль — тем больше.
						// Куб (Push) толкает сильно сильнее обычного отброса.
						mm := mobMass(m)
						if mm > 0 {
							kk := petalKnockbackK
							if petalKinds[pet.Kind].Push {
								kk = cubePushK
							}
							if petalKinds[pet.Kind].Sling {
								kk = pebblePushK
							}
							knockback := kk * orbitPetalMass(*pet) / mm * float64(strikes)
							m.X += (dx / dist) * knockback
							m.Y += (dy / dist) * knockback
							clampMobInZone(m)
						}
					}
					break
				}
			}

			// Петали игрока бьют и орбитальные петали боссов (Fractal/Chaos):
			// удары по телу босса НЕ блокируют бой с его петалем — иначе на коротком
			// кулдауне OnCD (0.3 с) петали часто вообще не успевали бить друг друга.
			for mi := range r.mobs {
				m := &r.mobs[mi]
				if m.HP <= 0 || m.Zone != p.State.Zone || len(m.petals) == 0 {
					continue
				}
				for k := range m.petals {
					bp := &m.petals[k]
					if bp.HP <= 0 {
						continue
					}
					bx, by := bossPetalPos(m, bp)
					ddx := bx - px
					ddy := by - py
					if ddx*ddx+ddy*ddy < (bp.Size+pet.Size)*(bp.Size+pet.Size) {
						dmg := pet.DMG * petalDmgMult * float64(strikes)
						// Tier-гейт редкости — орбитальная петаль босса наследует
						// редкость своего моба (Fractal/Chaos).
						dmg *= petalVsMobMult(pet.Rarity, m.Rarity)
						bp.HP -= dmg
						// Ядовитый петаль травит и орбитальные петали боссов.
						if doPoison {
							poisonBossPetal(bp, dmg*poisonDpsMult)
						}
						// Шипы: возвращённый износ колет и орбитальную петаль босса.
						if petalKinds[pet.Kind].Thorns {
							bp.HP -= bp.DMG * 0.3 * thornsReflectRatio * float64(strikes)
						}
						// Уничтоженная петаль босса респавнится на том же месте
						// орбиты через 3-5 с (как петали игрока на перезарядке).
						if bp.HP <= 0 {
							bp.HP = 0
							bp.Respawn = mobPetalsRespawnMin +
								rand.Float64()*(mobPetalsRespawnMax-mobPetalsRespawnMin)
						}
						// Износ за КАЖДЫЙ удар берста; контакт перезапускает атаку.
						pet.HP -= bp.DMG * 0.3 * float64(strikes)
						pet.Charging = true
						pet.AtkT = 0
						if pet.HP <= 0 {
							pet.HP = 0
							rch := petalRechargeCD(p, petalRechargeT)
							pet.Recharge = rch
							pet.CDMax = rch
							// Разрыв (hiveling): осколки в врагов.
							if petalKinds[pet.Kind].Split {
								r.spawnHivelingShards(p, pet)
							}
							// Слеза (tear): сломанная петаль лечит игрока.
							if petalKinds[pet.Kind].Tear {
								p.State.HP += p.State.MaxHP * tearHealFrac(pet.Rarity)
								if p.State.HP > p.State.MaxHP {
									p.State.HP = p.State.MaxHP
								}
							}
						}
						break
					}
				}
			}
		}
	}
}

// runeBoom — детонация рун (петаль rune): когда на мобе накопилось
// runeStackNeed стаков, вокруг него взрывается полностью: AoE-урон
// runeBoomMult×DMG петеля по всем врагам в runeBoomRadius (сам моб тоже
// получает волну — безвредно, так как детонация наступает сразу после его
// удара и его HP уже списано). Урон идёт в Damage-карту игрока.
func (r *Room) runeBoom(fromID string, src *Mob, petDMG float64) {
	for i := range r.mobs {
		mm := &r.mobs[i]
		if mm.HP <= 0 || mm.Zone != src.Zone || mm.Showcase {
			continue
		}
		dx := mm.X - src.X
		dy := mm.Y - src.Y
		if math.Sqrt(dx*dx+dy*dy) < runeBoomRadius+mm.Radius {
			mm.HP -= petDMG * runeBoomMult
			mm.Damage[fromID] += petDMG * runeBoomMult
			if mm.HP <= 0 {
				mm.KillerID = fromID
				r.awardMobKill(mm)
			}
		}
	}
	// Взрыв руны задевает и энергетические шары Beam рядом (шар не моб —
	// дропа/XP не даёт).
	for oi := range r.orbs {
		o := &r.orbs[oi]
		if o.HP <= 0 || o.Zone != src.Zone {
			continue
		}
		dx := o.X - src.X
		dy := o.Y - src.Y
		if math.Sqrt(dx*dx+dy*dy) < runeBoomRadius+o.Size {
			o.HP -= petDMG * runeBoomMult
		}
	}
}

// hiveSummonMob — Hive спавнит ЗАЩИТНИКА случайного типа зоны на редкость ниже.
// Раньше улей призывал КОПИЮ СЕБЯ (hive -> hive) — и улья росли в геометрической
// прогрессии (каждые 15% HP новый улей, который тоже копирует себя). Теперь
// призывается НЕ-hive моб зоны (по умолчанию Hexagon-солдат), рост линейный.
func hiveSummonMob(r *Room, h *Mob) {
	summonRar := h.Rarity - 1
	if summonRar < 0 {
		summonRar = 0
	}
	tname := "hexagon"
	for _, tn := range mobTypesInZone[h.Zone] {
		if tn != "hive" && tn != "hexagon" {
			tname = tn
			break
		}
	}
	mt := mobTypes[tname]
	angle := rand.Float64() * 2 * math.Pi
	dist := h.Radius + 40
	summoned := Mob{
		ID:        "hive_" + strconv.Itoa(r.mobIDSeq),
		Type:      tname,
		X:         h.X + math.Cos(angle)*dist,
		Y:         h.Y + math.Sin(angle)*dist,
		Zone:      h.Zone,
		Damage:    make(map[string]float64),
		state:     "patrol",
		moveTimer: rand.Float64() * 3,
		dirX:      rand.Float64()*2 - 1,
		dirY:      rand.Float64()*2 - 1,
	}
	// Полное масштабирование редкости (включая боссовые множители HP):
	// саммон Fractal-редкости обязан иметь те же HP, что и обычный Fractal.
	// Hive с подвидом призывает мобов того же подвида (Golden Hexagon и т.д.).
	summoned.Variant = h.Variant
	scaleMobForRarity(&summoned, summonRar)
	summoned.Speed = mt.BaseSpeed * (0.8 + rand.Float64()*0.5)
	clampMobInZone(&summoned)
	r.mobIDSeq++
	r.mobs = append(r.mobs, summoned)
}

// awardMobKill обрабатывает смерть моба: награждает опытом убийцу, раздаёт дроп
// топ-K дамагерам и рассылает всем сообщение о «затухании» моба.
func (r *Room) awardMobKill(m *Mob) {
	// Идемпотентность: урон из нескольких источников в один тик (кольца сатурна,
	// руны, цепь, яд) может триггерить смерть повторно — награждаем один раз.
	if m.DeadAwarded {
		return
	}
	m.DeadAwarded = true

	// Опыт убийце (как было).
	if pk, ok := r.players[m.KillerID]; ok {
		if pk.awardXP(mobXP(m)) {
			pk.ensurePetalCap()
			if cl := r.clients[m.KillerID]; cl != nil {
				cl.deliver([]byte(`{"type":"levelup","level":` +
					strconv.Itoa(pk.State.Level) + `}`))
			}
		}
	}

	// Fractal-босс умер — таймеров больше нет: следующие спавнятся рандомным
	// роллом 1/1500 в верхней полосе L2.

	// Топ-K дамагеров по накопленному урону получают дроп (по 1 петалю каждый).
	// Если кто-то не наносил урона — в топ он не попадает.
	// Anti-lootsteal: порог доли урона от максимального HP моба зависит от
	// редкости: Point..Pentagon (0-4) — 25% (мобов бьют 1-2 игрока), Hexagon..
	// Chaos (5-7) — 5% (боссов бьют большими группами).
	const topK = 4
	type dmgInfo struct {
		id string
		d  float64
	}
	list := make([]dmgInfo, 0, len(m.Damage))
	for id, d := range m.Damage {
		list = append(list, dmgInfo{id, d})
	}
	sort.Slice(list, func(a, b int) bool {
		return list[a].d > list[b].d
	})
	if len(list) > topK {
		list = list[:topK]
	}
	minRatio := 0.25
	if m.Rarity >= 5 {
		minRatio = 0.05
	}
	lootThreshold := m.MaxHP * minRatio

	// Fallback: если НИКТО не набрал порог — лут всё равно уходит:
	// обычные редкости — одному топ-дамагеру, глубокие (5-7) — топ-4 по урону
	// без порога (случай толпы по 3-4%: четверо лучших получают дроп).
	anyQualified := false
	for _, d := range list {
		if d.d >= lootThreshold {
			anyQualified = true
			break
		}
	}
	if !anyQualified && len(list) > 0 {
		if m.Rarity < 5 {
			list = list[:1]
		}
		lootThreshold = 0 // получателям fallback порог не применяется
	}

	// Золотой подвид: дроп ровно ×10 (по goldLootRolls попыток на каждого
	// топ-дамагера вместо одной); прочие подвиды бустят не число, а редкость
	// (см. makeDropPetal).
	goldRolls := 1
	if m.Variant == "gold" {
		goldRolls = goldLootRolls
	}

	// Уведомление о смерти Chaos/Fractal-моба с именами топ-4 игроков, нанёсших урон.
	if m.Rarity >= rarityFractal {
		var names []string
		for _, d := range list {
			if d.d <= 0 {
				break
			}
			if p, ok := r.players[d.id]; ok {
				names = append(names, p.State.Name)
			}
			if len(names) >= 4 {
				break
			}
		}
		if len(names) > 0 {
			label := "Fractal"
			if m.Rarity == rarityChaos {
				label = "Chaos"
			}
			// Подвид: «A Golden Chaos Arrow has been defeated by ...».
			label = variantPrefix(m.Variant) + label
			r.broadcastSystemLocked("super", "A "+label+" "+mobTypes[m.Type].Name+
				" has been defeated by "+strings.Join(names, ", ")+"!")
		}
	}

	if r.waveMode {
		// Командный режим: дроп с мба падает ДЛЯ КАЖДОГО члена команды —
		// у каждого своя персональная копия (OwnerID), все могут подобрать.
		// Рецепт дропа считается один раз (по мобу), а не по вкладу игроков.
		if m.Type == "blackhole" && rand.Float64() >= blackHoleDropChance {
			// Singularity не выпал — дропа нет ни у кого.
		} else {
			extra := 0
			switch m.Rarity {
			case rarityFractal:
				if rand.Float64() < 0.35 {
					extra = 1
				}
			case rarityChaos:
				if rand.Float64() < 0.5 {
					extra = 1
				}
				if extra == 1 && rand.Float64() < 0.2 {
					extra = 2
				}
			}
			total := (1 + extra) * goldRolls
			for _, p := range r.players {
				owner := p.State.ID
				for n := 0; n < total; n++ {
					r.gpIDSeq++
					r.gp = append(r.gp, makeDropPetal(m, owner, r.gpIDSeq))
				}
			}
		}
	} else {
		for _, d := range list {
			if d.d < lootThreshold {
				continue // не нанёс порог от HP моба — лут не положен (анти-лутстил)
			}
			// Black Hole: Singularity падает с 20% шансом (per-mob roll).
			if m.Type == "blackhole" && rand.Float64() >= blackHoleDropChance {
				continue
			}
			// Боссы Fractal/Chaos кидают «бонусную пачку»: Fractal — 35% шанс
			// на +1 петаль, Chaos — 50% на +1 и ещё 20% на +2.
			extra := 0
			switch m.Rarity {
			case rarityFractal:
				if rand.Float64() < 0.35 {
					extra = 1
				}
			case rarityChaos:
				if rand.Float64() < 0.5 {
					extra = 1
				}
				if extra == 1 && rand.Float64() < 0.2 {
					extra = 2
				}
			}
			total := (1 + extra) * goldRolls
			for n := 0; n < total; n++ {
				// Петаль-дроп падает возле трупа моба, принадлежит этому игроку.
				r.gpIDSeq++
				r.gp = append(r.gp, makeDropPetal(m, d.id, r.gpIDSeq))
			}
		}
	}

	// Всем клиентам — сигнал «моб умер и затухает» (id, позиция, вид, цвет).
	// Моб с подвидом и мёртвым остаётся окрашенным: цвет в сообщении такой же,
	// как у живого в buildState (иначе труп затухал бы в натуральном цвете).
	mt := mobTypes[m.Type]
	diedColor := mt.Color
	if vc := variantColor(m.Variant); vc != 0 {
		diedColor = vc
	}
	died := map[string]interface{}{
		"type": "mobDied", "id": m.ID,
		"x": m.X, "y": m.Y, "size": m.Radius,
		"shape": mt.Shape, "color": diedColor,
	}
	msg, _ := json.Marshal(died)
	for _, c := range r.clients {
		if p, ok := r.players[c.playerID]; ok && p.State.Zone == m.Zone {
			c.deliver(msg)
		}
	}

	// Фрактальная механика: actal при смерти спавнит 2-3 мини-копии.
	// Только для Line-редкости и выше (Point не размножается — иначе бесконечный фарм).
	// Мини-копии сами НЕ размножаются (noClone), иначе рост был бы экспоненциальным.
	if m.Type == "actal" && m.Rarity >= 1 && !m.noClone {
		miniCount := 2 + rand.Intn(2) // 2-3 штуки
		for i := 0; i < miniCount; i++ {
			mt := mobTypes[m.Type]
			angle := rand.Float64() * 2 * math.Pi
			dist := 30.0 + rand.Float64()*50.0
			szMult := 0.3
			mini := Mob{
				ID:        "fractal_" + strconv.Itoa(r.mobIDSeq),
				Type:      m.Type,
				X:         m.X + math.Cos(angle)*dist,
				Y:         m.Y + math.Sin(angle)*dist,
				Zone:      m.Zone,
				Damage:    make(map[string]float64),
				state:     "patrol",
				moveTimer: rand.Float64() * 3,
				dirX:      rand.Float64()*2 - 1,
				dirY:      rand.Float64()*2 - 1,
				noClone: true,
				// Actal с подвидом распадается на мини-копий того же подвида.
				Variant: m.Variant,
			}
			// Полное масштабирование редкости (включая боссовые ×10 Fractal
			// и ×32 Chaos), затем мини-версия — 30% от полных статов.
			scaleMobForRarity(&mini, m.Rarity)
			mini.HP *= szMult
			mini.MaxHP = mini.HP
			mini.DMG *= szMult
			mini.Radius *= szMult
			mini.Mass = mini.Radius
			mini.Speed = mt.BaseSpeed * (0.8 + rand.Float64()*0.4)
			clampMobInZone(&mini)
			r.mobIDSeq++
			r.mobs = append(r.mobs, mini)
		}
	}

	// Hive: при смерти спавнится Hexagon той же редкости, преследующий игрока.
	if m.Type == "hive" && !m.hiveDeathSpawned {
		m.hiveDeathSpawned = true
		hmt := mobTypes["hexagon"]
		hm := Mob{
			ID:        "hive_hex_" + strconv.Itoa(r.mobIDSeq),
			Type:      "hexagon",
			X:         m.X,
			Y:         m.Y,
			Zone:      m.Zone,
			Damage:    make(map[string]float64),
			state:     "chase",
			moveTimer: 9999,
			dirX:      0,
			dirY:      0,
			// Hive с подвидом выпускает Hexagon того же подвида и редкости.
			Variant: m.Variant,
		}
		// Полное масштабирование редкости (включая боссовые множители HP).
		scaleMobForRarity(&hm, m.Rarity)
		hm.Speed = hmt.BaseSpeed * 1.5
		clampMobInZone(&hm)
		r.mobIDSeq++
		r.mobs = append(r.mobs, hm)
	}
}

// ---- Боёвка: мобы vs игроки ----

func (r *Room) fightMobsVsPlayers(dt float64) {
	for _, p := range r.players {
		if p.Dead || p.transferT > 0 || p.mobHitCD > 0 {
			continue
		}
		if p.Passive {
			continue // пассивный админ неуязвим для мобов
		}
		if !playerHostileToMob(p.Team) {
			continue // команда 2 не получает урона от мобов
		}
		// Каждый моб бьёт игрока только вне общего кулдауна игрока.
		for i := range r.mobs {
			m := &r.mobs[i]
			if m.HP <= 0 || m.Zone != p.State.Zone {
				continue
			}
			dx := p.State.X - m.X
			dy := p.State.Y - m.Y
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist < m.Radius+p.State.Size {
				// «Шиповое тело» (петаль spike): при контакте тело игрока колет моба.
				if m.SpikeT <= 0 {
					if bd := bodySpikeDmg(p); bd > 0 {
						m.HP -= bd
						m.Damage[p.State.ID] += bd
						m.SpikeT = spikeGrindCD
						if m.HP <= 0 {
							m.KillerID = p.State.ID
							r.awardMobKill(m)
						}
					}
				}
				if m.HP <= 0 {
					break // моб добит шипами — не даём ему ударить в этот же тик
				}
				// Иммунитет после респавна/портала — урона нет, только отталкивание.
				if p.invulnT > 0 || p.GodMode {
					if dist > 0 {
						p.State.X += (dx / dist) * 20
						p.State.Y += (dy / dist) * 20
						p.State.X, p.State.Y = clampPlayerInZone(p.zone, p.State.X, p.State.Y, p.State.Size)
					}
					break
				}
				applyPlayerDamage(p, m.DMG*damageTakenMult(p))
				p.mobHitCD = playerHitCD
				if dist > 0 {
					p.State.X += (dx / dist) * 20
					p.State.Y += (dy / dist) * 20
					p.State.X, p.State.Y = clampPlayerInZone(p.zone, p.State.X, p.State.Y, p.State.Size)
				}
				if p.State.HP <= 0 {
					p.State.HP = 0
					p.Dead = true
					if cl := r.clients[p.State.ID]; cl != nil {
						cl.deliver([]byte(`{"type":"died"}`))
					}
				}
				break
			}
		}
	}
}

// fightBossPetalsVsPlayers — атаки орбитальных петалей боссов по игроку и его
// петалем: каждая петаль имеет свой HP, урон и кулдаун CD. Бьёт и тело игрока,
// и его петали на орбите (те изнашиваются/ломаются, как от мобов).
func (r *Room) fightBossPetalsVsPlayers(dt float64) {
	for _, p := range r.players {
		if p.Dead || p.transferT > 0 || p.Passive {
			continue
		}
		if !playerHostileToMob(p.Team) {
			continue // команда 2 не получает урона от босс-петалей
		}
		if p.State.Y > mobZoneBottom(&zones[p.State.Zone]) {
			continue // игрок в Safe Zone — петали его не касаются
		}
		for i := range r.mobs {
			m := &r.mobs[i]
			if m.HP <= 0 || m.Zone != p.State.Zone || len(m.petals) == 0 {
				continue
			}
			for k := range m.petals {
				bp := &m.petals[k]
				if bp.HP <= 0 || bp.CD > 0 {
					continue
				}
				bx, by := bossPetalPos(m, bp)

				// 1) По телу игрока.
				dx := p.State.X - bx
				dy := p.State.Y - by
				if dx*dx+dy*dy < (bp.Size+p.State.Size)*(bp.Size+p.State.Size) {
					if p.mobHitCD > 0 || p.invulnT > 0 || p.GodMode {
						continue
					}
					applyPlayerDamage(p, bp.DMG*damageTakenMult(p))
					p.mobHitCD = playerHitCD
					bp.CD = mobPetalCD
					if p.State.HP <= 0 {
						p.State.HP = 0
						p.Dead = true
						if cl := r.clients[p.State.ID]; cl != nil {
							cl.deliver([]byte(`{"type":"died"}`))
						}
					}
					continue
				}

				// 2) По петалем игрока на орбите.
				for pi := range p.petals {
					pet := &p.petals[pi]
					if pet.HP <= 0 || isOrbitAccessory(pet.Kind) {
						continue
					}
					px, py := petalWorldPos(p, pet.Angle, pet.Dist)
					ddx := px - bx
					ddy := py - by
					if ddx*ddx+ddy*ddy < (bp.Size+pet.Size)*(bp.Size+pet.Size) {
						pet.HP -= bp.DMG
						pet.AtkT = 0 // контакт босс-петаля перезапускает атаку
						if pet.HP <= 0 {
							pet.HP = 0
							rch := petalRechargeCD(p, petalRechargeT)
							pet.Recharge = rch
							pet.CDMax = rch
							// Слеза (tear): сломанная петаль лечит игрока.
							if petalKinds[pet.Kind].Tear {
								p.State.HP += p.State.MaxHP * tearHealFrac(pet.Rarity)
								if p.State.HP > p.State.MaxHP {
									p.State.HP = p.State.MaxHP
								}
							}
						}
						bp.CD = mobPetalCD
						break
					}
				}
			}
		}
	}
}

// ---- Тест-ПВП: петали админа против других игроков ----
// fightPvP — бой между враждебными игроками (команда 1 или 2). Активные петали
// враждебного игрока вредят другим игрокам (их тело и петали), которые ему
// враждебны. Взаимный износ петалей. Дружелюбные (команда 0) не участвуют.
func (r *Room) fightPvP(dt float64) {
	for _, ap := range r.players {
		if ap.Dead || ap.transferT > 0 || ap.Team == 0 {
			continue
		}
		for pi := 0; pi < len(ap.petals); pi++ {
			pPet := &ap.petals[pi]
			// Аксессуары не участвуют в ПВП (вечны и не бьют).
			if isOrbitAccessory(pPet.Kind) {
				continue
			}
			// Атака в ПВП: та же AtkT-семантика, что у мобов (Charging=false = готова,
			// удар; затем зарядка с берстом 1-3 удара за тик при низком КД).
			if pPet.Recharge > 0 || pPet.HP <= 0 {
				continue
			}
			atkCD := petalAtkCD(pPet.Kind)
			strikes := 1
			if pPet.Charging {
				if atkCD > 0 {
					pPet.AtkT += dt / atkCD
					if pPet.AtkT > 3 {
						pPet.AtkT = 3
					}
					if pPet.AtkT < 1 {
						continue // ещё заряжается
					}
				}
				strikes = int(pPet.AtkT)
				if strikes > 3 {
					strikes = 3
				}
				pPet.AtkT -= float64(strikes)
			} else if atkCD > 0 {
				// Готова — удар мгновенно, затем перезарядка.
				pPet.Charging = true
				pPet.AtkT = 0
			}
			apx, apy := petalWorldPos(ap, pPet.Angle, pPet.Dist)

			for _, op := range r.players {
				if op == ap || op.Dead || op.transferT > 0 || op.State.Zone != ap.State.Zone {
					continue
				}
				if !playersAreEnemies(ap, op) {
					continue
				}

				// Петаль враждебного игрока бьёт самого другого игрока.
				dx := op.State.X - apx
				dy := op.State.Y - apy
				if math.Sqrt(dx*dx+dy*dy) < op.State.Size+pPet.Size {
					if op.GodMode || op.invulnT > 0 {
						continue // неуязвимого/только что заспавненного не трогаем
					}
					applyPlayerDamage(op, pPet.DMG*petalDmgMult*damageTakenMult(op)*float64(strikes))
					// КД удара учтён зарядом pet.AtkT (Charging/AtkT-семантика).
					pPet.AtkT = 0
					if op.State.HP <= 0 {
						op.State.HP = 0
						op.Dead = true
						if cl := r.clients[op.State.ID]; cl != nil {
							cl.deliver([]byte(`{"type":"died"}`))
						}
					}
					break
				}

				// Петаль враждебного игрока против петалей другого игрока.
				for oi := 0; oi < len(op.petals); oi++ {
					oPet := &op.petals[oi]
					if oPet.HP <= 0 || isOrbitAccessory(oPet.Kind) {
						continue
					}
					ox, oy := petalWorldPos(op, oPet.Angle, oPet.Dist)
					dd := ox - apx
					dyy := oy - apy
					if math.Sqrt(dd*dd+dyy*dyy) < oPet.Size+pPet.Size {
						oPet.HP -= pPet.DMG * petalDmgMult * float64(strikes)
						// Петаль-«гадюка» травит петали других игроков (ПВП).
						if petalKinds[pPet.Kind].Poison {
							poisonPlayerPetal(oPet, pPet.DMG*petalDmgMult*poisonDpsMult)
						}
						pPet.HP -= oPet.DMG * 0.3
						// Износ ПВП: обе петали перезапускают атаку (AtkT-семантика).
						pPet.Charging = true
						pPet.AtkT = 0
						oPet.Charging = true
						oPet.AtkT = 0
						if oPet.HP <= 0 {
							oPet.HP = 0
							orch := petalRechargeCD(op, petalRechargeT)
							oPet.Recharge = orch
							oPet.CDMax = orch
							// Слеза (tear): сломанная петаль лечит её владельца.
							if petalKinds[oPet.Kind].Tear {
								op.State.HP += op.State.MaxHP * tearHealFrac(oPet.Rarity)
								if op.State.HP > op.State.MaxHP {
									op.State.HP = op.State.MaxHP
								}
							}
						}
						if pPet.HP <= 0 {
							pPet.HP = 0
							pch := petalRechargeCD(ap, petalRechargeT)
							pPet.Recharge = pch
							pPet.CDMax = pch
							// Слеза (tear): сломанная петаль лечит её владельца.
							if petalKinds[pPet.Kind].Tear {
								ap.State.HP += ap.State.MaxHP * tearHealFrac(pPet.Rarity)
								if ap.State.HP > ap.State.MaxHP {
									ap.State.HP = ap.State.MaxHP
								}
							}
						}
						break
					}
				}
			}
		}
	}
}

// ---- Коллизия игроков ----

func (r *Room) collidePlayers() {
	ids := make([]string, 0, len(r.players))
	for id := range r.players {
		ids = append(ids, id)
	}
	for i := 0; i < len(ids); i++ {
		a := r.players[ids[i]]
		if a.Dead || a.transferT > 0 {
			continue
		}
		for j := i + 1; j < len(ids); j++ {
			b := r.players[ids[j]]
			if b.Dead || b.transferT > 0 {
				continue
			}
			if a.State.Zone != b.State.Zone {
				continue
			}
			dx := b.State.X - a.State.X
			dy := b.State.Y - a.State.Y
			dist := math.Sqrt(dx*dx + dy*dy)
			minDist := a.State.Size + b.State.Size
			if dist < minDist && dist > 0 {
				overlap := (minDist - dist) / 2
				nx := dx / dist
				ny := dy / dist
				a.State.X -= nx * overlap
				a.State.Y -= ny * overlap
				b.State.X += nx * overlap
				b.State.Y += ny * overlap
				a.State.X, a.State.Y = clampToZone(a.zone, a.State.X, a.State.Y, a.State.Size)
				b.State.X, b.State.Y = clampToZone(b.zone, b.State.X, b.State.Y, b.State.Size)
			}
		}
	}
}

// ---- Респавн игрока (после смерти) ----

func (r *Room) respawnPlayer(p *Player) {
	// Смерть: мобы полностью забывают этого игрока (сброс агро — см. выше).
	r.resetMobAggroAgainst(p)
	p.Dead = false
	p.State.Dead = false
	p.State.HP = playerMaxHP
	p.noDamageT = 0
	p.input = Vec{X: 0, Y: 0}
	p.petalDist = petalBaseDist
	p.Dead = false
	p.State.Dead = false
	// Постоянность: орбита строится из активного хотбара.
	// Если хотбар пуст — заполняем базовыми Point-петлями (вид oval).
	cap := petalCapForLevel(p.State.Level)
	if len(p.hotbars[p.activeBar]) == 0 {
		if len(p.inventory) == 0 {
			// Совсем пусто — выдаём базовые Point-петали (вид oval).
			for i := 0; i < cap; i++ {
				p.hotbars[0] = append(p.hotbars[0], basePetalDef(p.State.ID, i))
			}
		} else {
			// Есть запас в инвентаре — снаряжаем из него активный хотбар.
			r.autoEquipFromInventory(p)
		}
	}
	r.rebuildOrbit(p)
	rchAll := petalRechargeCD(p, petalRechargeT)
	for i := range p.petals {
		p.petals[i].HP = 0
		p.petals[i].Recharge = rchAll // перезарядка после смерти (cog ускоряет)
		p.petals[i].CDMax = rchAll
	}
	// Иммунитет на 3 секунды после респавна.
	p.invulnT = respawnInvuln
	// Респавн в Point (зона 0), слева посередине — начало карты.
	p.transferTo = -1
	p.portalCD = 0
	p.transferT = 0
	// Респавн: волновой режим — в центре круглой арены; обычный — на чекпоинте.
	if r.waveMode {
		spawnWavePlayer(r, p)
	} else {
		spawnAtCheckpoint(p, p.checkpoint)
	}
}

// basePetal создаёт базовую Point-петаль игрока (вид oval).
func basePetal(pid string, i int) Petal {
	k := petalKinds["oval"]
	return Petal{
		ID:       "pet_" + pid + "_b" + strconv.Itoa(i),
		Angle:    (2 * math.Pi / float64(3)) * float64(i),
		Dist:     petalBaseDist,
		HP:       k.BaseHP,
		MaxHP:    k.BaseHP,
		DMG:      k.BaseDMG,
		BaseDist: petalBaseDist,
		Rarity:   0,
		Color:    petalColor,
		Size:     petalSizeBase,
		Kind:     "oval",
	}
}

// basePetalDef создаёт базовый Point-предмет петеля (вид oval) для хотбара.
func basePetalDef(pid string, i int) PetalDef {
	k := petalKinds["oval"]
	def := PetalDef{
		ID:     "inv_" + pid + "_b" + strconv.Itoa(i),
		Kind:   "oval",
		Name:   k.Name,
		Rarity: 0,
		HP:     k.BaseHP,
		MaxHP:  k.BaseHP,
		DMG:    k.BaseDMG,
		Size:   petalSizeBase,
		Color:  petalColor,
	}
	normalizePetalDef(&def)
	return def
}

// spawnCommonSpawn ставит игрока в начало карты (Point-зона), внизу по центру
// Safe Zone (нижняя полоса безопасности).
func spawnCommonSpawn(p *Player) {
	spawnInZone(p, 0)
}

// spawnAtCheckpoint ставит игрока в его закреплённый чекпоинт. Чекпоинт бывает
// только на первом уровне (0-7): из второго уровня смерть/релог всегда ведут
// в первый. Нет чекпоинта — Point.
func spawnAtCheckpoint(p *Player, zone int) {
	if zone < 0 || zone >= secondTierOffset {
		zone = 0
	}
	spawnInZone(p, zone)
}

// spawnInZone ставит игрока внизу по центру сейф-зоны указанной зоны.
func spawnInZone(p *Player, zone int) {
	z := zones[zone]
	p.zone = zone
	p.State.Zone = zone
	// Внизу-посередине Safe Zone, небольшая случайность, чтобы не оказаться в стене.
	p.State.X = z.X + z.W/2 + rand.Float64()*80 - 40
	p.State.Y = z.Y + z.H - safeZoneH/2 + rand.Float64()*60 - 30
}

// ---- Подключение / отключение ----

// connTakeoverAfter — сколько соединение может молчать, прежде чем повторный
// вход с тем же ником заберёт слот. После обрыва сети (сон ноутбука, Wi-Fi)
// старый readPump держит игрока до 60с pong-таймаута, и честный реконнект
// получал «This nickname is already in use!» с вылетом в меню.
const connTakeoverAfter = 15 * time.Second

// detachSnapshot — снимок игрока для сохранения профиля ВНЕ лока комнаты.
type detachSnapshot struct {
	name      string
	level     int
	xp        float64
	inv       []InvStack
	hotbars   [2][]PetalDef
	activeBar int
	pending   []PetalDef
}

// detachPlayerLocked снимает игрока id с карт комнаты и возвращает снимок
// для persistDetach. ВЫЗЫВАТЬ ПОД r.mu. usedNiks ведётся счётчиком: одно
// подключение = один ++, одно отключение = один --. Иначе отложенный
// RemoveClient «молчащего» соединения освободил бы ник, уже занятый новым
// игроком (на map[string]bool такое было возможно).
func (r *Room) detachPlayerLocked(id string) (detachSnapshot, bool) {
	p, ok := r.players[id]
	if !ok {
		return detachSnapshot{}, false
	}
	s := detachSnapshot{
		name:      p.State.Name,
		level:     p.State.Level,
		xp:        p.State.XP,
		inv:       append([]InvStack(nil), p.inventory...),
		hotbars:   [2][]PetalDef{nil, nil},
		activeBar: p.activeBar,
		pending:   append([]PetalDef(nil), p.pendingCraft...),
	}
	s.hotbars[0] = append([]PetalDef(nil), p.hotbars[0]...)
	s.hotbars[1] = append([]PetalDef(nil), p.hotbars[1]...)
	delete(r.players, id)
	delete(r.clients, id)
	r.decUsedNick(s.name)
	return s, true
}

// persistDetach пишет профиль отключившегося игрока в БД (вне лока комнаты,
// чтобы игровой цикл не блокировался на медленной MongoDB).
func (r *Room) persistDetach(s detachSnapshot) {
	saveProfile(s.name, s.level, s.xp, s.inv, s.hotbars, s.activeBar)
	savePendingCraft(s.name, s.pending)
}

func (r *Room) incUsedNick(name string) { r.usedNiks[name]++ }

func (r *Room) decUsedNick(name string) {
	if n := r.usedNiks[name]; n <= 1 {
		delete(r.usedNiks, name)
	} else {
		r.usedNiks[name] = n - 1
	}
}

// playerIDByName ищет id подключённого игрока по нику.
func (r *Room) playerIDByName(name string) (string, bool) {
	for id, p := range r.players {
		if p.State.Name == name {
			return id, true
		}
	}
	return "", false
}

// hasClientNamed — есть ли сейчас живой WS-клиент с этим ником (нужен reaper'у
// лобби: игрок в матче не опрашивает lobby-API, но его соединение активно).
func (r *Room) hasClientNamed(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, c := range r.clients {
		if p, ok := r.players[c.playerID]; ok && p.State.Name == name {
			return true
		}
	}
	return false
}

// takeOverStaleSession — если ник занят «молчащим» соединением (нет данных
// дольше connTakeoverAfter), снимает старую сущность с сохранением профиля и
// закрывает её коннект: новый игрок входит бесшовно. Живое соединение не
// трогаем — в этом случае возвращается ошибка. Так вторая вкладка не умеет
// мгновенно отбирать сессию у активной (last-wins только после молчания),
// и нет ping-pong войны двух вкладок за коннект.
func (r *Room) takeOverStaleSession(name string) string {
	var saved detachSnapshot
	has := false
	var oldConn *websocket.Conn
	r.mu.Lock()
	if oldID, ok := r.playerIDByName(name); ok {
		oldC := r.clients[oldID]
		if oldC != nil && oldC.lastRecvAge() < connTakeoverAfter {
			r.mu.Unlock()
			return "This nickname is already in use!"
		}
		saved, _ = r.detachPlayerLocked(oldID)
		has = true
		if oldC != nil {
			oldConn = oldC.conn
		}
	}
	r.mu.Unlock()
	if has {
		r.persistDetach(saved)
		if oldConn != nil {
			oldConn.Close() // readPump старого коннекта дочистит за собой
		}
		log.Printf("Ник %q: перезахват молчащего (> %v) соединения", name, connTakeoverAfter)
	}
	return ""
}

func (r *Room) AddClient(c *Client, name string, ip string, color uint32) string {
	// Забаненные аккаунты не пускаем, даже если токен валиден.
	if userBanned(name) {
		return "This account is banned"
	}

	// MongoDB-вызовы ДЕЛАЕМ ДО захвата r.mu: раньше loadProfile/loadCheckpoint
	// шли ПОД локом, и при подтормаживающей БД (3с таймаут ×2 запроса) весь
	// игровой цикл замирал на секунды. Игроки видели «в 3 раза медленнее игры»
	// и огромный пинг; из-за фризов соединения рвались, клиент переподключался
	// раз в 2с (новый AddClient = респавн на чекпоинте = «резет к спавну»).
	lvl, xp, savedInv, savedHotbars, savedActiveBar := loadProfile(name)
	checkpoint := loadCheckpoint(name)

	// Перезахват «молчащего» соединения с тем же ником (до захвата r.mu:
	// внутри возможен сейв профиля в БД).
	if msg := r.takeOverStaleSession(name); msg != "" {
		return msg
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.usedNiks[name] > 0 {
		return "This nickname is already in use!"
	}

	id := r.nextID
	r.nextID++
	idStr := strconv.Itoa(id)
	if savedActiveBar < 0 || savedActiveBar > 1 {
		savedActiveBar = 0
	}
	cap := petalCapForLevel(lvl)

	// Хотбары: берём сохранённые; если не сохранились (новый игрок) — выдаём
	// базовые Point-петали (вид oval) в линейке 0.
	hot0 := make([]PetalDef, 0, cap)
	if len(savedHotbars[0]) > 0 || len(savedHotbars[1]) > 0 {
		// Сохранённая раскладка — используем её, но обрезаем до текущего капа,
		// чтобы не было «фантомных» слотов за пределами капа (клиент не даст их
		// редактировать, а сервер не отдаст на орбиту больше капа).
		hot0 = savedHotbars[0]
		if len(hot0) > cap {
			hot0 = hot0[:cap]
		}
	}
	if len(hot0) == 0 {
		for i := 0; i < cap; i++ {
			hot0 = append(hot0, basePetalDef(idStr, i))
		}
	}
	hot1 := savedHotbars[1]
	if len(hot1) > cap {
		hot1 = hot1[:cap]
	}

	p := &Player{
		State: PlayerState{
			ID: idStr, Name: name,
			Size: playerSize, Color: color,
			HP: playerMaxHP, MaxHP: playerMaxHP,
			Level: lvl, XP: xp, XPMax: xpForLevel(lvl),
		},
		speed:     playerSpeed,
		zone:      0,
		inventory: savedInv,
		hotbars:   [2][]PetalDef{hot0, hot1},
		activeBar: savedActiveBar,
		petalDist: petalBaseDist,
		invulnT:   respawnInvuln, // короткая защита при первом входе в игру
	}
	r.rebuildOrbit(p)
	// Убеждаемся, что XP не выше порога текущего уровня (из-за округлений).
	if p.State.XP >= p.State.XPMax {
		p.State.XP = 0
	}

	// Спавнпоинт: волновой режим — центр арены; обычный — закреплённый чекпоинт.
	p.checkpoint = checkpoint
	if r.waveMode {
		spawnWavePlayer(r, p)
	} else {
		spawnAtCheckpoint(p, p.checkpoint)
	}

	r.players[idStr] = p
	r.clients[idStr] = c
	r.incUsedNick(name)
	c.playerID = idStr
	c.room = r

	log.Printf("Игрок %q (id=%s) в зоне %s. HP: %.0f. Всего: %d",
		name, idStr, zones[p.zone].Name, p.State.HP, len(r.players))
	return ""
}

func (r *Room) RemoveClient(c *Client, ip string) {
	r.mu.Lock()
	id := c.playerID
	if id == "" {
		r.mu.Unlock()
		return
	}
	s, ok := r.detachPlayerLocked(id)
	remaining := len(r.players)
	r.mu.Unlock()
	if !ok {
		return
	}
	log.Printf("Игрок (id=%s) отключился. Осталось: %d", id, remaining)
	// Сейвы — ВНЕ лока: игровой цикл не блокируем, даже если БД отвечает
	// медленно. Соединение закрывается только после readPump (RemoveClient
	// вызывается из его defer), так что реконнект того же ника не успеет
	// прочитать прежний (не актуальный) профиль раньше этой записи.
	r.persistDetach(s)
}

func (r *Room) RespawnPlayer(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	// В волновом режиме ручного респавна нет: мёртвые возвращаются на старте
	// следующей волны (см. startWave).
	if p, ok := r.players[id]; ok && p.Dead && !r.waveMode {
		r.respawnPlayer(p)
	}
}

func (r *Room) ReturnToMenu(id string, ip string) {
	r.mu.Lock()
	s, ok := r.detachPlayerLocked(id)
	remaining := len(r.players)
	r.mu.Unlock()
	if !ok {
		return
	}
	log.Printf("Игрок %q вернулся в меню. Осталось: %d", s.name, remaining)
	r.persistDetach(s)
}

// SwitchBar переключает активный хотбар (0/1) и пересобирает орбиту.
func (r *Room) SwitchBar(id string, bar int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if bar < 0 {
		bar = 0
	}
	if bar > 1 {
		bar = 1
	}
	p, ok := r.players[id]
	if !ok || p.Dead {
		return
	}
	if p.activeBar == bar {
		return
	}
	p.activeBar = bar
	r.rebuildOrbit(p)
	// После смены хотбара новые петали на короткой перезарядке — не мгновенно активны.
	r.rechargeOrbit(p, switchRechargeT)
}

// rechargeOrbit ставит все петали активной орбиты на короткую перезарядку
// (HP=0, Recharge=t) — используется после смены хотбара или изменения его
// раскладки, чтобы «свежие» петали не вступали в бой мгновенно.
// Аксессуары (eye/line/spike) НЕ трогаем: они вечны и без «загрузки».
func (r *Room) rechargeOrbit(p *Player, t float64) {
	// cog ускоряет ПЕРЕЗАРЯДКУ всех петалей (в т.ч. смену хотбара).
	tc := petalRechargeCD(p, t)
	for i := range p.petals {
		if isOrbitAccessory(p.petals[i].Kind) {
			continue
		}
		p.petals[i].HP = 0
		p.petals[i].Recharge = tc
		p.petals[i].CDMax = tc
	}
}

// clampBarSlot ограничивает бар и слот хотбара допустимыми значениями.
// Возвращает (bar, slot) и валидность слота (slot < cap уровня).
func (r *Room) clampBarSlot(p *Player, bar, slot int) (int, int, bool) {
	if bar < 0 {
		bar = 0
	}
	if bar > 1 {
		bar = 1
	}
	cap := petalCapForLevel(p.State.Level)
	if slot < 0 || slot >= cap {
		return bar, slot, false
	}
	return bar, slot, true
}

// EquipPetal ставит один предмет из стака инвентаря (вид, редкость) в слот хотбара.
// Если слот занят — старый предмет возвращается в инвентарь-стак.
func (r *Room) EquipPetal(id string, bar, slot int, kind string, rarity int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.players[id]
	if !ok || p.Dead {
		return
	}
	bar, slot, ok = r.clampBarSlot(p, bar, slot)
	if !ok {
		return
	}
	// Нужен непустой стак.
	i := r.findStack(p, kind, rarity)
	if i < 0 || p.inventory[i].Count <= 0 {
		return
	}
	st := p.inventory[i]

	// Уникальные аксессуары (eye/line/saturn): максимум 1 на линейке.
	if isUniquePetal(st.Kind) && r.barHasKindExcept(p, bar, slot, st.Kind) {
		return
	}

	// Если слот занят — вернуть старый предмет в запас.
	barDef, _ := r.hotbarAt(p, bar, slot)

	// Снимаем один из стака.
	r.takeOneFromInventory(p, st.Kind, st.Rarity)
	eq := st.PetalDef
	eq.ID = "hv_" + p.State.ID + "_" + strconv.Itoa(rand.Int())

	// Возвращаем старый предмет в запас.
	if barDef != nil {
		r.addToInventory(p, *barDef)
	}
	r.setHotbarAt(p, bar, slot, &eq)
	if bar == p.activeBar {
		// Только новая петаль свежая; остальные сохраняют HP/кулдауны.
		r.rebuildOrbit(p)
	}
}

// UnequipPetal снимает петаль со слота хотбара обратно в инвентарь-стак.
func (r *Room) UnequipPetal(id string, bar, slot int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.players[id]
	if !ok || p.Dead {
		return
	}
	bar, slot, ok = r.clampBarSlot(p, bar, slot)
	if !ok {
		return
	}
	def, present := r.hotbarAt(p, bar, slot)
	if !present {
		return
	}
	r.setHotbarAt(p, bar, slot, nil)
	// Уплотняем хотбар (не оставляем «дырок» до капа) не обязателен — слот станет пустым.
	r.addToInventory(p, *def)
	if bar == p.activeBar {
		r.rebuildOrbit(p)
	}
}

// MovePetal переносит/меняет петаль между слотом хотбара и/или стаком инвентаря.
// from: {src:"inv", kind, rarity} — из инвентаря; {src:"hot", bar, slot} — из слота.
// to:  {bar, slot} — целевой слот.
// Если целевой слот занят, предметы меняются местами (надетый уходит на источник
// хотбара либо возвращается в инвентарь, если источником был инвентарь).
func (r *Room) MovePetal(id string, from map[string]interface{}, toBar, toSlot int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.players[id]
	if !ok || p.Dead {
		return
	}
	toBar, toSlot, ok = r.clampBarSlot(p, toBar, toSlot)
	if !ok {
		return
	}

	// Определяем переносимый предмет и источник.
	var carried *PetalDef
	fromHot := false // true, если источник — слот хотбара
	var fromBar, fromSlot int

	src := "inv"
	if v, has := from["src"]; has {
		src, _ = v.(string)
	}
	if src == "hot" {
		fb, _ := from["bar"].(float64)
		fs, _ := from["slot"].(float64)
		fromBar = int(fb)
		fromSlot = int(fs)
		if def, ok := r.hotbarAt(p, fromBar, fromSlot); ok {
			carried = def
			fromHot = true
		}
	} else {
		kind, _ := from["kind"].(string)
		rarity := int(from["rarity"].(float64))
		i := r.findStack(p, kind, rarity)
		if i < 0 || p.inventory[i].Count <= 0 {
			return
		}
		d := p.inventory[i].PetalDef
		carried = &d
	}

	if carried == nil {
		return
	}

	// Уникальные аксессуары (eye/line/saturn): нельзя надевать второй на линейку.
	if isUniquePetal(carried.Kind) && r.barHasKindExcept(p, toBar, toSlot, carried.Kind) {
		return
	}

	// Предмет в целевом слоте (если есть).
	targetDef, targetOccupied := r.hotbarAt(p, toBar, toSlot)

	// Если двигаем на тот же слот из него же — ничего не делаем.
	if fromHot && fromBar == toBar && fromSlot == toSlot {
		return
	}

	// Снимаем источник.
	if fromHot {
		r.setHotbarAt(p, fromBar, fromSlot, nil)
	}

	// Если целевой слот занят другим предметом:
	//   — источник хотбара -> меняем местами (заменяемый едет на from-слот)
	//   — источник инвентаря -> заменяемый возвращается в инвентарь
	if targetOccupied && targetDef.ID != "" {
		if fromHot {
			r.setHotbarAt(p, fromBar, fromSlot, targetDef)
		} else {
			r.addToInventory(p, *targetDef)
		}
	}

	// Ставим переносимый в целевой слот. При переносе слот→слот сохраняем
	// ИСХОДНЫЙ ID предмета — иначе rebuildOrbit сочтёт петаль новой и сбросит
	// её живое состояние (HP/износ/кулдаун). Новый ID нужен только при
	// надевании из инвентаря (ID стака общий на весь стак).
	eq := *carried
	if !fromHot {
		eq.ID = "hv_" + p.State.ID + "_" + strconv.Itoa(rand.Int())
	}
	r.setHotbarAt(p, toBar, toSlot, &eq)

	// Если источник был инвентарём — списываем один из стака.
	if !fromHot {
		kind, _ := from["kind"].(string)
		rarity := int(from["rarity"].(float64))
		r.takeOneFromInventory(p, kind, rarity)
	}

	// Перестановка не должна лечить и сбрасывать кулдауны остальных петалей
	// (rebuildOrbit сохраняет их состояние), НО обе задействованные в обмене
	// петали уходят на короткую перезарядку — иначе перестановками в бою
	// можно «жонглировать» петали и обходить их износ.
	if toBar == p.activeBar || (fromHot && fromBar == p.activeBar) {
		r.rebuildOrbit(p)
		if fromHot {
			r.rechargePetal(p, carried.ID, switchRechargeT)
			if targetOccupied && targetDef != nil {
				r.rechargePetal(p, targetDef.ID, switchRechargeT)
			}
		} else {
			// Надевание из ИНВЕНТАРЯ: новая петаль тоже уходит на короткую
			// перезарядку (иначе надеванием «свежих» можно обходить износ).
			r.rechargePetal(p, eq.ID, switchRechargeT)
			if targetOccupied && targetDef != nil {
				r.rechargePetal(p, targetDef.ID, switchRechargeT)
			}
		}
	}
}

// LiveCraft — внутрипленной крафт (та же механика, что и в главном меню):
// statelessProfile.craft над ЖИВЫМ инвентарём игрока. Результаты попадают
// в p.pendingCraft (отложенные до нажатия «Take»). При выходе в меню/дисконнекте
// они сохраняются в БД. Уведомление о Fractal/Chaos-крафте — общее.
func (r *Room) LiveCraft(id, kind string, rarity, invest int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.players[id]
	if !ok || p.Dead {
		return
	}
	sp := &statelessProfile{
		inv: p.inventory, hotbars: p.hotbars, activeBar: p.activeBar,
		nick: p.State.Name, level: p.State.Level, pending: p.pendingCraft,
	}
	status, successes, lost, results, _ := sp.craft(kind, rarity, invest)
	p.inventory = sp.inv
	p.hotbars = sp.hotbars
	p.pendingCraft = sp.pending
	if status == "error" {
		b, _ := json.Marshal(map[string]interface{}{"type": "craftLive", "ok": false, "error": "Need5"})
		if c := r.clients[id]; c != nil {
			c.deliver(b)
		}
		return
	}
	ck := map[string]interface{}{
		"type": "craftLive", "ok": true,
		"status": status, "successes": successes, "lost": lost,
		"kind": kind, "rarity": rarity,
	}
	if successes > 0 {
		ck["next"] = rarity + 1
	}
	if p.pendingCraft != nil {
		ck["pending"] = p.pendingCraft
	}
	ck["inventory"] = sp.inv
	// Всемирный анонс крафта Fractal/Chaos — ТОЛЬКО при успехе.
	// Задержка 6с: катушка крутится ~5с (владелец видит результат) + 1с.
	// ГОРУТИНА через AfterFunc: она сама возьмёт r.mu (BroadcastSystem).
	// ВАЖНО: нельзя звать BroadcastSystem прямо здесь — LiveCraft уже держит
	// r.mu, повторный захват заблокировал бы весь сервер навсегда.
	if successes > 0 && rarity+1 >= rarityFractal {
		rarWord := rarityNames[rarity+1]
		petalName := "shape"
		if len(results) > 0 && results[0].Name != "" {
			petalName = results[0].Name
		}
		msgTxt := craftCelebrationText(p.State.Name, rarWord, petalName, successes)
		if msgTxt != "" {
			time.AfterFunc(6*time.Second, func() {
				theRoom.BroadcastSystem("craft"+rarWord, msgTxt)
			})
		}
	}
	b, _ := json.Marshal(ck)
	if c := r.clients[id]; c != nil {
		c.deliver(b)
	}
}

// LiveCraftTake — забрать отложенные результаты крафта в инвентарь (в игре).
func (r *Room) LiveCraftTake(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.players[id]
	if !ok || len(p.pendingCraft) == 0 {
		return
	}
	for _, d := range p.pendingCraft {
		r.addToInventory(p, d)
	}
	p.pendingCraft = nil
	ack, _ := json.Marshal(map[string]interface{}{"type": "craftLiveTake", "ok": true})
	if c := r.clients[id]; c != nil {
		c.deliver(ack)
	}
}

// rechargePetal ставит ОДНУ петаль активной орбиты на короткую перезарядку
// (ищет по устойчивому ID предмета — OwnerKey).
func (r *Room) rechargePetal(p *Player, ownerKey string, t float64) {
	for i := range p.petals {
		if p.petals[i].OwnerKey != ownerKey {
			continue
		}
		// Аксессуары (eye/line/spike) не «загружаются»: всегда на орбите.
		if isOrbitAccessory(p.petals[i].Kind) {
			return
		}
		// cog ускоряет ПЕРЕЗАРЯДКУ всех петалей.
		tc := petalRechargeCD(p, t)
		p.petals[i].HP = 0
		p.petals[i].Recharge = tc
		p.petals[i].CDMax = tc
		return
	}
}

// DropPetal выбрасывает один предмет из стака инвентаря — предмет уничтожается,
// игрок получает часть ценности как опыт.
func (r *Room) DropPetal(id, kind string, rarity int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.players[id]
	if !ok || p.Dead {
		return
	}
	i := r.findStack(p, kind, rarity)
	if i < 0 || p.inventory[i].Count <= 0 {
		return
	}
	st := p.inventory[i]
	// XP за выброс: часть «силы» петаля.
	xpGain := petalDropXP(st.PetalDef)
	r.takeOneFromInventory(p, st.Kind, st.Rarity)
	if xpGain > 0 {
		_ = p.awardXP(xpGain)
	}
}

// DropHotbar выбрасывает петаль со слота хотбара — уничтожается, даёт опыт.
func (r *Room) DropHotbar(id string, bar, slot int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.players[id]
	if !ok || p.Dead {
		return
	}
	bar, slot, ok = r.clampBarSlot(p, bar, slot)
	if !ok {
		return
	}
	def, present := r.hotbarAt(p, bar, slot)
	if !present {
		return
	}
	r.setHotbarAt(p, bar, slot, nil)
	if bar == p.activeBar {
		r.rebuildOrbit(p)
	}
	xpGain := petalDropXP(*def)
	if xpGain > 0 {
		_ = p.awardXP(xpGain)
	}
}

// hotbarAt возвращает предмет в слоте хотбара (nil, если пусто/слот за пределами).
func (r *Room) hotbarAt(p *Player, bar, slot int) (*PetalDef, bool) {
	if bar < 0 || bar > 1 {
		return nil, false
	}
	if slot < 0 || slot >= len(p.hotbars[bar]) {
		return nil, false
	}
	d := p.hotbars[bar][slot]
	if d.ID == "" {
		return nil, false
	}
	cp := d
	return &cp, true
}

// barHasKindExcept — есть ли в хотбаре bar (КРОМЕ слота slot) предмет с видом kind.
// Используется для уникальных аксессуаров (isUniquePetal): максимум 1 на линейку.
func (r *Room) barHasKindExcept(p *Player, bar, slot int, kind string) bool {
	if bar < 0 || bar > 1 || slot < 0 {
		return false
	}
	for i := range p.hotbars[bar] {
		if i == slot {
			continue
		}
		if p.hotbars[bar][i].ID != "" && p.hotbars[bar][i].Kind == kind {
			return true
		}
	}
	return false
}

// setHotbarAt кладёт предмет в слот (nil — очищает). Расширяет список до нужной длины.
func (r *Room) setHotbarAt(p *Player, bar, slot int, def *PetalDef) {
	if bar < 0 || bar > 1 || slot < 0 {
		return
	}
	hb := p.hotbars[bar]
	if slot >= len(hb) {
		grow := make([]PetalDef, slot-len(hb)+1)
		hb = append(hb, grow...)
	}
	if def == nil {
		hb[slot] = PetalDef{}
	} else {
		hb[slot] = *def
	}
	p.hotbars[bar] = hb
}

// petalDropXP — опыт за выброс петаля: почти полная «сила», но меньше, чем её
// ценность при носении; растёт с редкостью и статами.
func petalDropXP(d PetalDef) float64 {
	power := d.DMG*2 + d.MaxHP
	if power <= 0 {
		return 0
	}
	base := power * 0.6
	base *= 1 + petalPower[minInt(d.Rarity, len(petalPower)-1)]*0.1
	return base
}

// minInt возвращает меньшее из двух целых (для ограничения индексов).
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// BroadcastChat рассылает сообщение чата ВСЕМ подключённым игрокам.
// В тексте сообщения держим ник отправителя (глобальный чат).
// Сообщения, начинающиеся с "/", — команды: /give <ник> <вид> <редкость> (админ).
func (r *Room) BroadcastChat(id, text string) {
	r.mu.Lock()
	p, ok := r.players[id]
	if !ok {
		r.mu.Unlock()
		return
	}
	trimmed := strings.TrimSpace(text)
	// Слэш-команды (админ-функции): /help, "/", /give ... Иначе — обычный чат.
	if strings.HasPrefix(trimmed, "/") {
		nick := p.State.Name
		id := p.State.ID
		isAdm := isAdmin(nick)
		r.mu.Unlock()
		switch {
		case trimmed == "/" || trimmed == "/help":
			if isAdm {
				r.chatServerReply(id,
					"Admin commands: /give <nick> <petal> <rarity 0-7>, /help — this help")
			} else {
				r.chatServerReply(id, "Slash commands are for admins only")
			}
		case strings.HasPrefix(trimmed, "/give"):
			r.chatGiveCommand(id, nick, trimmed)
		default:
			if isAdm {
				r.chatServerReply(id, "Unknown command: "+trimmed+" (try /help)")
			} else {
				r.chatServerReply(id, "Slash commands are for admins only")
			}
		}
		return
	}
	if strings.HasPrefix(trimmed, "/give") {
		nick := p.State.Name
		r.mu.Unlock()
		r.chatGiveCommand(id, nick, trimmed)
		return
	}
	name := p.State.Name
	// Лимит длины сообщения: режем до 120 символов (как maxlength у клиента).
	if runes := []rune(text); len(runes) > 120 {
		text = string(runes[:120])
	}
	zone := ""
	if p.zone >= 0 && p.zone < len(zones) {
		zone = zones[p.zone].Name
	}
	msg, _ := json.Marshal(map[string]string{"type": "chat", "id": id, "name": name, "text": text, "zone": zone})
	clients := make([]*Client, 0, len(r.clients))
	for _, c := range r.clients {
		clients = append(clients, c)
	}
	r.mu.Unlock()
	for _, c := range clients {
		c.deliver(msg)
	}
}

// chatServerReply — ответ сервера В ЧАТ конкретному игроку (успех/ошибка
// слэш-команд /help / /give). Также дублируется как sys-тост на клиенте.
func (r *Room) chatServerReply(targetID, text string) {
	r.mu.Lock()
	cl := r.clients[targetID]
	r.mu.Unlock()
	if cl == nil {
		return
	}
	if msg, err := json.Marshal(map[string]string{"type": "chat", "name": "Server", "text": text, "zone": ""}); err == nil {
		cl.deliver(msg)
	}
}

// chatGiveCommand — админ-команда чата: /give <ник> <вид> <редкость>.
// Кладёт петаль фиксированных статов в инвентарь указанного игрока.
func (r *Room) chatGiveCommand(adminID, adminName, cmd string) {
	adminSend := func(text string) {
		// Успех/ошибка — в чат админу (и дублем в админ-панель).
		r.chatServerReply(adminID, text)
		r.mu.Lock()
		cl := r.clients[adminID]
		r.mu.Unlock()
		if cl != nil {
			msg, _ := json.Marshal(map[string]interface{}{"type": "adminAck", "text": text})
			cl.deliver(msg)
		}
	}
	sysTo := func(targetID, text string) {
		r.mu.Lock()
		cl := r.clients[targetID]
		r.mu.Unlock()
		if cl != nil {
			msg, _ := json.Marshal(map[string]interface{}{"type": "chat", "name": "Server", "text": text, "zone": ""})
			cl.deliver(msg)
		}
	}

	if !isAdmin(adminName) {
		adminSend("Only the admin can use /give")
		return
	}
	fields := strings.Fields(cmd)
	if len(fields) != 4 {
		adminSend("Usage: /give <nick> <petal> <rarity 0-7>")
		return
	}
	targetNick := fields[1]
	kind := strings.ToLower(fields[2])
	rar, err := strconv.Atoi(fields[3])
	if err != nil || rar < 0 || rar > len(rarityNames)-1 {
		adminSend("Rarity must be 0-7 (Point..Chaos)")
		return
	}
	kind = strings.ToLower(kind)
	if _, ok := petalKinds[kind]; !ok {
		adminSend("Unknown petal kind: " + fields[2])
		return
	}

	r.mu.Lock()
	var target *Player
	for _, pl := range r.players {
		if strings.EqualFold(pl.State.Name, targetNick) {
			target = pl
			break
		}
	}
	if target == nil {
		r.mu.Unlock()
		adminSend("Player \"" + targetNick + "\" is not online")
		return
	}
	def, ok := petalDefForAdmin(kind, rar)
	if ok {
		r.addToInventory(target, def)
		tName := target.State.Name
		tID := target.State.ID
		r.mu.Unlock()
		adminSend("Gave 1x " + def.Name + " " + rarityNames[rar] + " to " + tName)
		sysTo(tID, "Admin gave you: "+def.Name+" "+rarityNames[rar])
		return
	}
	r.mu.Unlock()
	adminSend("Unknown petal kind: " + fields[2])
}

// BroadcastSystem рассылает системное уведомление ВСЕМ игрокам (не от игрока).
// kind = "super" помечается специальным цветом на клиенте.
func (r *Room) BroadcastSystem(kind, text string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.broadcastSystemLocked(kind, text)
}

// broadcastSystemLocked рассылает системное уведомление ВСЕМ игрокам, НЕ беря
// r.mu (вызывающий обязан уже удерживать её). Используется из контекстов, где
// блокировка уже взята (tick -> spawnMob / awardMobKill), чтобы избежать
// повторного захвата не-реентерабельного мьютекса и связанного с ним дедлока.
func (r *Room) broadcastSystemLocked(kind, text string) {
	msg, _ := json.Marshal(map[string]string{"type": "sys", "kind": kind, "text": text})
	for _, c := range r.clients {
		c.deliver(msg)
	}
}

// broadcastSystemZoneLocked — то же, но только игрокам одной зоны (например,
// зональное оповещение о спавне Hexagon-мини-босса). Вызывающий обязан
// удерживать r.mu.
func (r *Room) broadcastSystemZoneLocked(kind string, zone int, text string) {
	msg, _ := json.Marshal(map[string]string{"type": "sys", "kind": kind, "text": text})
	for _, c := range r.clients {
		if p, ok := r.players[c.playerID]; ok && p.State.Zone == zone {
			c.deliver(msg)
		}
	}
}
