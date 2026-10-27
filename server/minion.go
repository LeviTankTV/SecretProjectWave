package main

import (
	"math"
	"math/rand"
	"strconv"
)

// ---- Миньоны петалей: роль «стадо» (Entropy) ----

// Миньон — союзный «дружеский энтропий-моб» игрока, призванный петелем Entropy
// (или Black Hole — петаль singularity). От вражеского моба Entropy он
// отличается: заметно меньшим размером (~60%), дружественной (зелёной)
// расцветкой и жёсткой коллизией по границам — миньоны НЕ входят в тела
// вражеских мобов (выталкиваются по краю, как игроки у мобов). Миньоны НЕ
// коллизируют с игроками (включая чужих) и их петлями: единственная «физика»
// для них — контакт с враждебными мобами (урон как у петеля), распихивание
// между собой (в т.ч. с чужими миньонами), выталкивание из мобов и коллизия
// с разделительной стеной (миньон не проходит через неё, только через проём).
//
// Статы миньона — по редкости призванного МОБА (summonSpec), не петеля:
// Point..Square петали призывают ту же редкость моба, Pentagon — Square ×2
// («Square MK-II»), Hexagon — редкость Polygon («Pentagon»), Fractal —
// Polygon ×2 с интервалом призыва 12с («Pentagon MK-II»), Chaos — Polygon ×5
// («Pentagon MK-III»). HP и урон миньона равны статам ВРАЖДЕБНОГО моба того
// же типа и редкости (mobStatsForRarity), умноженным на множитель усиления.
//
// Тайминг спавна (слот 0): как только петаль активен, через интервал спавна
// (7 с; у Fractal-петеля — 12 с) появляется первый миньон, ещё через такой же
// интервал — второй. При смерти миньона (в т.ч. если погибли оба) каждый новый
// появляется через свой интервал после смерти, в кап 2 миньона на петаль.
// Всё происходит только пока петаль жив и не на CD.
type Minion struct {
	ID           string
	OwnerID      string
	OwnerPetalID string // устойчивый ID петеля Entropy (PetalDef.ID)
	Slot         int    // 0..entropyMinionsPerPetal-1

	SpawnIn float64 // таймер: у слота 0 — до первого спавна 7с, далее 7с на смерти
	Alive   bool
	HitCD   float64

	X, Y       float64
	DirX, DirY float64

	HP     float64
	MaxHP  float64
	DMG    float64
	Size   float64
	Rarity int
	Zone   int
	Color  uint32
	Name   string
	Shape  string
	Type   string
	RarityLabel string // override для чипа редкости (MK-II/III), пусто = rarityNames[]
	SpawnInterval float64 // интервал призыва этого миньона (7с; Fractal — 12с)
}

const (
	entropyMinionsPerPetal = 2
	minionFollowDist       = 90.0  // дистанция следа за игроком (ближе к нему)
	minionAttackRange      = 260.0 // радиус поиска цели миньоном
	minionSpeed            = 170.0 // скорость миньона (замедлена)
	minionSpawnInterval    = 7.0  // сек: слот 0 — до первого спавна, после смерти — до нового
	minionWear             = 0.3  // доля урона моба, которую выдерживает миньон
	minionSizeFactor       = 0.6  // миньон заметно меньше вражеского моба той же редкости
	minionHitBuffer        = 6.0  // запас контакта у границы: удар засчитывается вдоль края моба
	minionColor            = 0x2ecc71 // дружелюбная зелёная расцветка
	// Интервал призыва Fractal-петеля: НЕ 7с, а 12с (увеличенный кулдаун призыва).
	minionSpawnIntervalFractal = 12.0
)

// summonSpec — сводная таблица призыва миньонов по редкости ПЕТАЛЯ (общая для
// петалей Entropy и Singularity/Black Hole). Возвращает редкость призванного
// моба, множитель усиления (HP и урона), надпись чипа редкости и интервал
// спавна миньона. Point..Square (0-3) призывают моба той же редкости, что и
// петаль; Pentagon (4) — Square ×2 («Square MK-II»); Hexagon (5) — редкость
// Polygon («Pentagon»); Fractal (6) — Polygon ×2 с интервалом призыва 12с
// («Pentagon MK-II»); Chaos (7) — Polygon ×5 («Pentagon MK-III»).
func summonSpec(petalRar int) (mobRar int, mult float64, label string, spawnInterval float64) {
	mult = 1
	label = ""
	spawnInterval = minionSpawnInterval
	switch petalRar {
	case 4:
		mobRar, mult, label = 3, 2, "Square MK-II"
	case 5:
		mobRar, label = 4, "Pentagon"
	case 6:
		mobRar, mult, label = 4, 2, "Pentagon MK-II"
		spawnInterval = minionSpawnIntervalFractal
	case 7:
		mobRar, mult, label = 4, 5, "Pentagon MK-III"
	default:
		if petalRar < 0 {
			petalRar = 0
		}
		if petalRar > 3 {
			petalRar = 3
		}
		mobRar = petalRar
	}
	if mobRar > len(petalPower)-1 {
		mobRar = len(petalPower) - 1
	}
	return
}

// minionAnchor — активный петаль Entropy/Black Hole на орбите и всё, что нужно
// миньонам: владелец, зона, точка следа, статы миньонов (редкость по таблице
// summonSpec, размер и цвет как у вражеского моба той же редкости).
type minionAnchor struct {
	owner  string
	zone   int
	ox, oy float64
	ang    float64
	hp     float64 // MaxHP и HP миньона (оба из них)
	dmg    float64
	size   float64
	rarity int
	rarityLabel string // override для чипа редкости (MK-II/III)
	spawnInterval float64 // интервал спавна слотов (7с, для Fractal — 12с)
	color  uint32
	name   string
	shape  string
	mtype  string
}

// collectMinionAnchors собирает якоря со всех живых/активных петалей Entropy
// на орбитах игроков.
func (r *Room) collectMinionAnchors() map[string]*minionAnchor {
	anchors := make(map[string]*minionAnchor)
	for _, p := range r.players {
		if p.Dead || p.transferT > 0 || p.Passive {
			continue
		}
		for i := range p.petals {
			pet := &p.petals[i]
			if pet.HP <= 0 || pet.Recharge > 0 {
				continue
			}
			pk := petalKinds[pet.Kind]
			if pk.MinionType == "" {
				continue
			}
			mobRar, mult, label, spIn := summonSpec(pet.Rarity)
			mt := mobTypes[pk.MinionType]
			// Статы миньона = статы враждебного моба того же типа (entropy/
			// blackhole) и той же редкости (summonMobStats — без боссовых
			// множителей HP). Петали глубоких редкостей (Hexagon/Fractal/
			// Chaos, 5+) дополнительно ослаблены ×0.5 — иначе призывы Black
			// Hole неубиваемы. MK-версии умножают HP и урон на множитель.
			baseHP, baseDMG := summonMobStats(pk.MinionType, mobRar)
			if pet.Rarity >= 5 {
				baseHP *= summonDeepDebuff
				baseDMG *= summonDeepDebuff
			}
			anchors[pet.OwnerKey] = &minionAnchor{
				owner: p.State.ID,
				zone:  p.State.Zone,
				ox:    p.State.X,
				oy:    p.State.Y,
				ang:   pet.Angle,
				hp:    baseHP * mult,
				dmg:   baseDMG * mult,
				size:  mt.BaseRadius * raritySize[mobRar] * minionSizeFactor,
				rarity: mobRar,
				rarityLabel: label,
				spawnInterval: spIn,
				color: minionColor,
				name:  mt.Name,
				shape: mt.Shape,
				mtype: pk.MinionType,
			}
		}
	}
	return anchors
}

// newMinion создаёт слот миньона под петаль. Слот 0 запускает таймер первого
// спавна (7 с), слот 1 — через 7 с после первого (итого 7 и 14 с после
// активации петеля, как задумано: «через 7 секунд одного, через 7 другого»).
func (r *Room) newMinion(pid string, slot int, a *minionAnchor) Minion {
	r.minionIDSeq++
	spawn := float64(slot+1) * a.spawnInterval
	return Minion{
		ID:           "mn_" + strconv.Itoa(r.minionIDSeq),
		OwnerID:      a.owner,
		OwnerPetalID: pid,
		Slot:         slot,
		SpawnIn:      spawn,
		X:            a.ox,
		Y:            a.oy,
		HP:           a.hp,
		MaxHP:        a.hp,
		DMG:          a.dmg,
		Size:         a.size,
		Rarity:       a.rarity,
		RarityLabel:  a.rarityLabel,
		SpawnInterval: a.spawnInterval,
		Zone:         a.zone,
		Color:        a.color,
		Name:         a.name,
		Shape:        a.shape,
		Type:         a.mtype,
	}
}

// updateMinions поддерживает жизнь всех миньонов комнаты: создаёт/удаляет слоты
// под активные петали Entropy, перезапускает таймеры возрождения, двигает,
// даёт бой с мобами и расталкивает миньонов друг с другом.
func (r *Room) updateMinions(dt float64) {
	anchors := r.collectMinionAnchors()

	// Удаляем миньонов, чей петаль больше не активен (снят с орбиты, пошёл на
	// перезарядку, игрок умер/перемещается/пассивный админ).
	kept := r.minions[:0]
	for i := range r.minions {
		if _, ok := anchors[r.minions[i].OwnerPetalID]; ok {
			kept = append(kept, r.minions[i])
		}
	}
	r.minions = kept

	// Обеспечиваем ровно entropyMinionsPerPetal слотов на каждый активный петаль.
	have := make(map[string]int, len(r.minions))
	for i := range r.minions {
		have[r.minions[i].OwnerPetalID]++
	}
	for pid, a := range anchors {
		for slot := have[pid]; slot < entropyMinionsPerPetal; slot++ {
			r.minions = append(r.minions, r.newMinion(pid, slot, a))
		}
	}

	for mi := range r.minions {
		mn := &r.minions[mi]
		a := anchors[mn.OwnerPetalID]
		if a == nil {
			continue
		}
		mn.Zone = a.zone

		if !mn.Alive {
			// Таймер до спавна/возрождения. Первый слот — 7с после активации
			// петеля, второй — 14с; после смерти каждого — снова 7с.
			mn.SpawnIn -= dt
			if mn.SpawnIn > 0 {
				continue
			}
			// Спавн рядом с игроком.
			ang := rand.Float64() * 2 * math.Pi
			mn.X = a.ox + math.Cos(ang)*30
			mn.Y = a.oy + math.Sin(ang)*30
			mn.HP = mn.MaxHP
			mn.HitCD = 0
			mn.Alive = true
			clampMinionInZone(mn)
			continue
		}

		r.moveMinion(mn, a, dt)
		r.fightMinionVsMobs(mn, dt)
	}

	// Миньоны отталкивают друг друга (в т.ч. миньонов других игроков) и
	// выталкиваются из тел вражеских мобов (не входят в них).
	r.collideMinions()
	r.collideMinionsVsMobs()
}

// moveMinion выбирает цель (ближайший вражеский моб в радиусе атаки) или, если
// цели нет, возвращается в точку следа за орбитальной позицией своего петеля.
func (r *Room) moveMinion(mn *Minion, a *minionAnchor, dt float64) {
	var tx, ty float64
	hasTarget := false
	var best float64 = -1
	for i := range r.mobs {
		m := &r.mobs[i]
		if m.HP <= 0 || m.Zone != mn.Zone {
			continue
		}
		dx := m.X - mn.X
		dy := m.Y - mn.Y
		d2 := dx*dx + dy*dy
		if d2 < minionAttackRange*minionAttackRange && (best < 0 || d2 < best) {
			best = d2
			tx, ty = m.X, m.Y
			hasTarget = true
		}
	}

	if hasTarget {
		dx := tx - mn.X
		dy := ty - mn.Y
		ln := math.Sqrt(dx*dx + dy*dy)
		if ln > 0 {
			mn.DirX = dx / ln
			mn.DirY = dy / ln
		}
	} else {
		// Точка следа: миньоны держатся по бокам от ИГРОКА (не от петеля), на
		// фиксированных углах — не вращаются вместе с петалем Entropy.
		ang := 0.0
		if mn.Slot == 1 {
			ang = math.Pi
		}
		fx := a.ox + math.Cos(ang)*minionFollowDist
		fy := a.oy + math.Sin(ang)*minionFollowDist
		dx := fx - mn.X
		dy := fy - mn.Y
		dd := math.Sqrt(dx*dx + dy*dy)
		if dd < 10 {
			mn.DirX = 0
			mn.DirY = 0
		} else {
			mn.DirX = dx / dd
			mn.DirY = dy / dd
		}
	}

	if mn.DirX != 0 || mn.DirY != 0 {
		mn.X += mn.DirX * minionSpeed * dt
		mn.Y += mn.DirY * minionSpeed * dt
		clampMinionInZone(mn)
	}
}

// fightMinionVsMobs — контактный бой миньона с враждебными мобами (как у
// петелей): моб получает DMG, миньон изнашивается на долю урона моба. Урон
// записывается владельцу, чтобы дроп и опыт шли ему.
func (r *Room) fightMinionVsMobs(mn *Minion, dt float64) {
	if mn.HitCD > 0 {
		mn.HitCD -= dt
		if mn.HitCD > 0 {
			return
		}
	}
	for i := range r.mobs {
		m := &r.mobs[i]
		if m.HP <= 0 || m.Zone != mn.Zone || m.Showcase {
			continue
		}
		dx := m.X - mn.X
		dy := m.Y - mn.Y
		dist := math.Sqrt(dx*dx + dy*dy)
		// Контакт у самой границы: миньон не входит в моба, а «прижимается» к его
		// краю (коллизия выталкивает его до dist == m.Radius+mn.Size), поэтому
		// бой засчитывается по границе + небольшой запас на шаг тика.
		if dist >= m.Radius+mn.Size+minionHitBuffer {
			continue
		}
		m.HP -= mn.DMG
		if m.Damage == nil {
			m.Damage = make(map[string]float64)
		}
		addMobDamage(m, mn.OwnerID, mn.DMG)
		mn.HitCD = petalDamageCD
		mn.HP -= m.DMG * minionWear
		if mn.HP <= 0 {
			mn.HP = 0
			mn.Alive = false
			// Через интервал призыва после смерти спавнится новый (обычно за это
			// время его место на орбите уже покинуто). Интервал зависит от
			// редкости петеля (Fractal — 12с, остальные — 7с).
			mn.SpawnIn = mn.SpawnInterval
			return
		}
		if m.HP <= 0 {
			m.KillerID = mn.OwnerID
			r.awardMobKill(m)
		}
		return
	}
}

// collideMinions расталкивает живых миньонов (своих и чужих игроков), чтобы
// они не складывались в одну точку, когда следуют за игроком.
func (r *Room) collideMinions() {
	for i := 0; i < len(r.minions); i++ {
		a := &r.minions[i]
		if !a.Alive {
			continue
		}
		for j := i + 1; j < len(r.minions); j++ {
			b := &r.minions[j]
			if !b.Alive || a.Zone != b.Zone {
				continue
			}
			dx := b.X - a.X
			dy := b.Y - a.Y
			dist := math.Sqrt(dx*dx + dy*dy)
			minDist := a.Size + b.Size
			if dist < minDist && dist > 0 {
				overlap := minDist - dist
				nx := dx / dist
				ny := dy / dist
				a.X -= nx * overlap / 2
				a.Y -= ny * overlap / 2
				b.X += nx * overlap / 2
				b.Y += ny * overlap / 2
				clampMinionInZone(a)
				clampMinionInZone(b)
			}
		}
	}
}

// collideMinionsVsMobs не даёт миньонам входить в тела вражеских мобов:
// при перекрытии выталкиваем миньона за границу (как мобы выталкивают игроков
// в collideMobsVsPlayers). Моб «тяжёлый» и остаётся на месте.
func (r *Room) collideMinionsVsMobs() {
	for i := range r.minions {
		mn := &r.minions[i]
		if !mn.Alive {
			continue
		}
		for j := range r.mobs {
			m := &r.mobs[j]
			if m.HP <= 0 || m.Zone != mn.Zone {
				continue
			}
			dx := mn.X - m.X
			dy := mn.Y - m.Y
			dist := math.Sqrt(dx*dx + dy*dy)
			minDist := m.Radius + mn.Size
			if dist < minDist && dist > 0 {
				overlap := minDist - dist
				nx := dx / dist
				ny := dy / dist
				mn.X += nx * overlap
				mn.Y += ny * overlap
				clampMinionInZone(mn)
			}
		}
	}
}

// clampMinionInZone ограничивает миньона ВСЕЙ зоной (и Mob zone, и Safe zone):
// миньоны — «мобы игрока» и ходят за игроком везде. Кастомные стены (walls.json)
// по-прежнему толкают их наружу. Разделительную стену миньон ВЕРХНИМ проходом
// через её центр (как игрок) — через саму стену миньон не проходит: коллизия
// выталкивает его из прямоугольников стены (раньше миньон проходил насквозь).
func clampMinionInZone(mn *Minion) {
	if mn.Zone < 0 || mn.Zone >= len(zones) {
		return
	}
	z := zones[mn.Zone]
	rds := mn.Size
	mn.X, mn.Y = clampZoneRounded(&z, mn.X, mn.Y, rds)
	// Разделительная стена (без прохода посередине): не даём миньону пролезть
	// сквозь стены между Mob Zone и Safe Zone — только через проход.
	segs := wallSegments(&z)
	for _, w := range segs {
		mn.X, mn.Y = pushCircleOutOfRect(mn.X, mn.Y, rds, w)
	}
	for _, w := range zoneWallRects(mn.Zone) {
		mn.X, mn.Y = pushCircleOutOfRect(mn.X, mn.Y, rds, w)
	}
}
