package main

import "math"

// chain.go — цепной урон петеля spark и визуальная линия-цепь.
// При ударе по мобу цепь бьёт ближайших врагов в радиусе (sparkChainRange),
// урон падает на sparkChainFalloff за звено. Точки цепочки записываются в
// r.chains и отправляются клиенту для отрисовки линии.

type Chain struct {
	Points []Vec
	Life   float64
	Color  uint32
	Zone   int
}

type ChainState struct {
	Points []Vec  `json:"points"`
	Color  uint32 `json:"color"`
}

// sparkChain — цепной урон от удара петеля spark. startX/startY — точка удара
// (обычно моб или шар Beam), dmg — урон основного удара, skipID — сущность,
// которая уже получила основной удар (её цепь не перебивает). Идёт до
// sparkChainCount звеньев. Цели цепи — живые мобы И энергетические шары Beam.
func (r *Room) sparkChain(ownerID string, zone int, startX, startY, dmg float64, skipID string) {
	points := []Vec{{X: startX, Y: startY}}
	curX, curY := startX, startY
	curDmg := dmg
	used := map[string]bool{skipID: true}
	for i := 0; i < sparkChainCount; i++ {
		// Ближайший живой враг (моб или шар Beam) той же зоны в радиусе,
		// ещё не задетый цепью.
		var bestM *Mob
		var bestO *BeamOrb
		bestD := sparkChainRange
		for mi := range r.mobs {
			m := &r.mobs[mi]
			if m.HP <= 0 || m.Zone != zone || m.Showcase || used[m.ID] {
				continue
			}
			dx := m.X - curX
			dy := m.Y - curY
			if d := math.Sqrt(dx*dx + dy*dy); d < bestD {
				bestD = d
				bestM = m
				bestO = nil
			}
		}
		for oi := range r.orbs {
			o := &r.orbs[oi]
			if o.HP <= 0 || o.Zone != zone || used[o.ID] {
				continue
			}
			dx := o.X - curX
			dy := o.Y - curY
			if d := math.Sqrt(dx*dx + dy*dy); d < bestD {
				bestD = d
				bestO = o
				bestM = nil
			}
		}
		if bestM == nil && bestO == nil {
			break
		}
		curDmg *= (1 - sparkChainFalloff)
		if bestO != nil {
			// Шар Beam: просто отнимаем HP (шар не моб — убийству/дропу нет).
			bestO.HP -= curDmg
			used[bestO.ID] = true
			points = append(points, Vec{X: bestO.X, Y: bestO.Y})
			curX, curY = bestO.X, bestO.Y
			continue
		}
		bestM.HP -= curDmg
		if bestM.Damage == nil {
			bestM.Damage = map[string]float64{}
		}
		bestM.Damage[ownerID] += curDmg
		if bestM.HP <= 0 {
			bestM.KillerID = ownerID
			r.awardMobKill(bestM)
		}
		used[bestM.ID] = true
		points = append(points, Vec{X: bestM.X, Y: bestM.Y})
		curX, curY = bestM.X, bestM.Y
	}
	if len(points) > 1 {
		r.chains = append(r.chains, Chain{Points: points, Life: sparkChainLife, Color: 0xffd75e, Zone: zone})
		r.chainDebugFired += 1
	} else {
		// Диагностика: удар был, но ни одного соседа не нашлось (нет целей).
		// Запоминаем дистанцию до БЛИЖАЙШЕГО живого врага той же зоны — покажет,
		// насколько реальная плотность ниже радиуса цепи.
		r.chainDebugEmpty += 1
		bD := math.MaxFloat64
		var bT, bR string
		for mi := range r.mobs {
			n := &r.mobs[mi]
			if n.HP <= 0 || n.Zone != zone || n.Showcase || used[n.ID] {
				continue
			}
			dx := n.X - curX
			dy := n.Y - curY
			if d := math.Sqrt(dx*dx + dy*dy); d < bD {
				bD = d
				bT = n.Type
				if n.Rarity <= 7 && n.Rarity >= 0 {
					bR = rarityNames[n.Rarity]
				} else {
					bR = "?"
				}
			}
		}
		for oi := range r.orbs {
			n := &r.orbs[oi]
			if n.HP <= 0 || n.Zone != zone || used[n.ID] {
				continue
			}
			dx := n.X - curX
			dy := n.Y - curY
			if d := math.Sqrt(dx*dx + dy*dy); d < bD {
				bD = d
				bT = "beamorb"
				if n.Rarity <= 7 && n.Rarity >= 0 {
					bR = rarityNames[n.Rarity]
				} else {
					bR = "?"
				}
			}
		}
		if bD < r.chainDebugMissMin {
			r.chainDebugMissMin = bD
			r.chainDebugMissInfo = bT + "/" + bR
		}
	}
}

// updateChains тикает время жизни линий-цепей и убирает отжившие.
func (r *Room) updateChains(dt float64) {
	if len(r.chains) == 0 {
		return
	}
	kept := r.chains[:0]
	for i := range r.chains {
		r.chains[i].Life -= dt
		if r.chains[i].Life > 0 {
			kept = append(kept, r.chains[i])
		}
	}
	r.chains = kept
}

// bombSplash — мини-АОЕ петеля ball: наносит bombDmgFrac × урона всем живым
// мобам И шарам Beam в радиусе bombRadius вокруг точки удара (кроме самого
// моба/шара skipID).
func (r *Room) bombSplash(ownerID string, zone int, cx, cy, dmg float64, skipID string) {
	for mi := range r.mobs {
		m := &r.mobs[mi]
		if m.HP <= 0 || m.Zone != zone || m.ID == skipID {
			continue
		}
		dx := m.X - cx
		dy := m.Y - cy
		if math.Sqrt(dx*dx+dy*dy) < bombRadius {
			s := dmg * bombDmgFrac
			m.HP -= s
			if m.Damage == nil {
				m.Damage = map[string]float64{}
			}
			m.Damage[ownerID] += s
			if m.HP <= 0 {
				m.KillerID = ownerID
				r.awardMobKill(m)
			}
		}
	}
	// Энергетические шары Beam тоже задевает мини-АОЕ (шар не моб — дропа нет).
	for oi := range r.orbs {
		o := &r.orbs[oi]
		if o.HP <= 0 || o.Zone != zone || o.ID == skipID {
			continue
		}
		dx := o.X - cx
		dy := o.Y - cy
		if math.Sqrt(dx*dx+dy*dy) < bombRadius {
			o.HP -= dmg * bombDmgFrac
		}
	}
}

// Бафы ПОСТОЯННЫЕ: действуют, пока петаль надета в активный хотбар, даже если
// она сломана/на перезарядке (ломается только её «боевая» функция).

// orbitSpinBonus — суммарная прибавка к скорости вращения орбиты от надетых
// петалей cog (Spin): каждый даёт cogSpinBase + cogSpinPerRar×редкость, стакается.
func orbitSpinBonus(p *Player) float64 {
	bonus := 0.0
	for i := range p.petals {
		pet := &p.petals[i]
		if petalKinds[pet.Kind].Spin {
			bonus += cogSpinBase + cogSpinPerRar*float64(pet.Rarity)
		}
	}
	return bonus
}

// orbitDir — направление вращения орбиты: +1 (по часовой) или -1 (против).
// Каждый надетый Twirl (петаль «вертушка») ПЕРЕВОРАЧИВАЕТ направление, как
// только появляется в активном хотбаре: один — в обратную сторону, два —
// снова по часовой (парность). Действует постоянно, пока петаль надета,
// даже если она сломана/на перезарядке (как остальные пассивные бафы).
func orbitDir(p *Player) float64 {
	n := 0
	for i := range p.petals {
		if petalKinds[p.petals[i].Kind].Twirl {
			n++
		}
	}
	if n%2 == 1 {
		return -1
	}
	return 1
}

// moveSpeedMult — множитель скорости передвижения игрока от надетых петалей
// penta (Swift): каждый даёт swiftBasePct + swiftPerRarity×редкость, стакается.
func moveSpeedMult(p *Player) float64 {
	mult := 1.0
	for i := range p.petals {
		pet := &p.petals[i]
		if petalKinds[pet.Kind].Swift {
			mult += swiftBasePct + swiftPerRarity*float64(pet.Rarity)
		}
	}
	return mult
}

// damageTakenMult — множитель входящего урона игрока от надетых петалей
// fragwall (Guard): каждая снижает на guardBasePct + guardPerRarity×редкость,
// стакается, но не ниже 20% от полного урона.
func damageTakenMult(p *Player) float64 {
	mult := 1.0
	for i := range p.petals {
		pet := &p.petals[i]
		if petalKinds[pet.Kind].Guard {
			mult -= guardBasePct + guardPerRarity*float64(pet.Rarity)
		}
	}
	if mult < 0.2 {
		mult = 0.2
	}
	return mult
}

// applyPlayerDamage — наносит урон игроку: сначала щит (броня), затем HP.
// Любой урон обнуляет таймер «без урона» (реген вне боя start с нуля).
func applyPlayerDamage(p *Player, dmg float64) {
	p.noDamageT = 0
	if p.State.Shield > 0 {
		absorbed := math.Min(p.State.Shield, dmg)
		p.State.Shield -= absorbed
		dmg -= absorbed
	}
	if dmg > 0 {
		p.State.HP -= dmg
	}
}

// dashVoid — рывок петеля void (Dash): по пробелу тратит активный void-петаль
// (уходит в перезарядку) и сдвигает игрока на voidDashDist по направлению
// движения (p.lastDir). Если игрок не двигался — рывок вправо.
func (r *Room) dashVoid(pid string) {
	p, ok := r.players[pid]
	if !ok || p.Dead || p.transferT > 0 {
		return
	}
	for i := range p.petals {
		pet := &p.petals[i]
		if pet.Kind != "void" || pet.HP <= 0 || pet.Recharge > 0 {
			continue
		}
		dx, dy := p.lastDir.X, p.lastDir.Y
		if dx == 0 && dy == 0 {
			dx, dy = 1, 0
		}
		// Рывок проходится МЕЛКИМИ ШАГАМИ с клампом на каждом: дистанция рывка
		// (180) больше толщины разделительной стены (300 не толщина, но для
		// безопасного попадания нужен шаг меньше запаса), поэтому одиночный
		// прыжок с клампом после него может «перепрыгнуть» стену целиком и
		// протолкнуть игрока в Safe Zone. Пошаговый кламп этого не допускает:
		// круг, коснувшийся стены, выталкивается обратно, а проход по центру
		// (проём) по-прежнему свободно проходим.
		// Рывок проходится МЕЛКИМИ ШАГАМИ с клампом на каждом: дистанция рывка
		// (180) больше толщины разделительной стены (300 не толщина, но для
		// безопасного попадания нужен шаг меньше запаса), поэтому одиночный
		// прыжок с клампом после него может «перепрыгнуть» стену целиком и
		// протолкнуть игрока в Safe Zone. Пошаговый кламп этого не допускает:
		// круг, коснувшийся стены, выталкивается обратно, а проход по центру
		// (проём) по-прежнему свободно проходим.
		stepDist := voidDashStep // runtime-переменная, чтобы Go не считал константу
		nSteps := int(voidDashDist/stepDist + 0.5)
		if nSteps < 1 {
			nSteps = 1
		}
		sx := dx * voidDashDist / float64(nSteps)
		sy := dy * voidDashDist / float64(nSteps)
		for i := 0; i < nSteps; i++ {
			p.State.X += sx
			p.State.Y += sy
			p.State.X, p.State.Y = clampPlayerInZone(p.zone, p.State.X, p.State.Y, p.State.Size)
		}
		// cog ускоряет перезарядку всех петалей (в т.ч. способности).
		vch := petalRechargeCD(p, voidCooldown)
		pet.Recharge = vch
		pet.CDMax = vch
		return
	}
}
