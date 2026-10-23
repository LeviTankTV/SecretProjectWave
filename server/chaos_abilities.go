package main

import (
	"math"
)

// chaos_abilities.go — способности мобов Chaos-зоны, тикающие раз в тик
// (порт родительского SecretProject, адаптация 2.0).
//
//	blackhole — ПРИТЯЖЕНИЕ игроков: сдвигает живых игроков к мобу со скоростью
//	    mobHolePullBase + mobHolePullPerRar×редкость (Point 70 px/s → Chaos 420),
//	    в радиусе mobHolePullRBase + mobHolePullRPerRar×редкость. Как гравитация:
//	    тянет всегда (это характер моба), но не трогает пассивных.
//	saturn — аура урона вокруг моба: те же кривые, что у ПЕТАЛИ Saturn
//	    (radius = saturnRingBase + saturnRingPerRar×редкость, DPS =
//	    saturnDpsBase × petalPower редкости). Урон только враждебным игрокам,
//	    без отброса — как у петали.
//
// void (микротелепорт) и entropy (инверсия ввода) живут в updateMob: телепорт
// нужен в момент погони (state chase + цель), инверсия — на цели chase.
func (r *Room) applyMobChaosAbilities(dt float64) {
	for i := range r.mobs {
		m := &r.mobs[i]
		if m.HP <= 0 || m.Showcase {
			continue
		}
		switch m.Type {
		case "blackhole":
			pull := mobHolePullBase + mobHolePullPerRar*float64(m.Rarity)
			rad := mobHolePullRBase + mobHolePullRPerRar*float64(m.Rarity)
			// Гравитация всегда должна ДЕЙСТВОВАТЬ за телом моба: у глубоких
			// редкостей тело огромно (Chaos ~400+px), а кривая радиуса
			// константная — тяги не хватало дотянуться из-за края тела.
			if rad < m.Radius+80 {
				rad = m.Radius + 80
			}
			for _, p := range r.players {
				if p.Dead || p.transferT > 0 || p.Passive || p.State.Zone != m.Zone {
					continue
				}
				if !mobAggrosPlayer(p.Team) {
					continue // команда мобов не притягивается
				}
				dx := m.X - p.State.X
				dy := m.Y - p.State.Y
				d := math.Sqrt(dx*dx + dy*dy)
				if d >= rad || d <= 1e-6 {
					continue
				}
				nx := p.State.X + dx/d*pull*dt
				ny := p.State.Y + dy/d*pull*dt
				if r.waveMode {
					// Круглая арена лобби: держим игрока внутри круга.
					nx, ny = r.clampArena(nx, ny, p.State.Size)
				} else {
					nx, ny = clampPlayerInZone(p.zone, nx, ny, p.State.Size)
				}
				p.State.X, p.State.Y = nx, ny
			}
		case "saturn":
			// Аура: радиус и DPS как у петали Saturn редкости = редкость моба.
			// Кривая петали наследует РАЗМЕР ПЕТАЛИ, а не моба — у глубоких
			// редкостей тело моба больше кольца, поэтому аура обязана
			// выступать ЗА корпус минимум на 70px.
			ringR := saturnRingBase + saturnRingPerRar*float64(m.Rarity)
			if ringR < m.Radius+70 {
				ringR = m.Radius + 70
			}
			dmg := saturnDpsBase * petalPower[minInt(m.Rarity, len(petalPower)-1)] * dt
			for _, p := range r.players {
				if p.Dead || p.transferT > 0 || p.Passive || p.GodMode || p.invulnT > 0 || p.State.Zone != m.Zone {
					continue // пассив/неубиваемый/иммунитет — аура по ним не бьёт
				}
				if !mobAggrosPlayer(p.Team) {
					continue
				}
				dx := p.State.X - m.X
				dy := p.State.Y - m.Y
				if math.Sqrt(dx*dx+dy*dy) >= ringR+p.State.Size {
					continue
				}
				applyPlayerDamage(p, dmg*damageTakenMult(p))
				if p.State.HP <= 0 {
					p.State.HP = 0
					p.Dead = true
					if cl := r.clients[p.State.ID]; cl != nil {
						cl.deliver([]byte(`{"type":"died"}`))
					}
				}
			}
			// Миньоны игрока (дружелюбные entropy/blackhole) — тоже «тела» в
			// ауре: тикаются уроном, зачисленным просто в износ миньона.
			for j := range r.minions {
				mn := &r.minions[j]
				if !mn.Alive || mn.Zone != m.Zone {
					continue
				}
				dx := mn.X - m.X
				dy := mn.Y - m.Y
				if math.Sqrt(dx*dx+dy*dy) >= ringR+mn.Size {
					continue
				}
				mn.HP -= dmg
				if mn.HP <= 0 {
					mn.HP = 0
					mn.Alive = false
					mn.SpawnIn = mn.SpawnInterval
				}
			}
		}
	}
}
