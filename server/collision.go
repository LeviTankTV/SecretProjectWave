package main

import "math"

// collision.go - коллизии мобов с мобами, игроками и петалями на земле.

// collideMobs расталкивает мобов одной зоны, чтобы они не наезжали друг на друга
// ни физически, ни визуально. Сдвиг каждого — по массе: тяжёлый моб отталкивает
// лёгкого на большую часть перекрытия, сам едва сдвигается («тяжелее = тяжелее
// пушить» по массе: оба сдвигаются обратно пропорционально своей массе).
func (r *Room) collideMobs() {
	for i := 0; i < len(r.mobs); i++ {
		a := &r.mobs[i]
		if a.HP <= 0 {
			continue
		}
		for j := i + 1; j < len(r.mobs); j++ {
			b := &r.mobs[j]
			if b.HP <= 0 || a.Zone != b.Zone {
				continue
			}
			// Hive игнорирует столкновения с другими мобами.
			// Члены одной группы Trinity друг сквозь друга не толкаются.
			if a.noMobCollide || b.noMobCollide ||
				(a.GroupID != "" && a.GroupID == b.GroupID) {
				continue
			}
			dx := b.X - a.X
			dy := b.Y - a.Y
			dist := math.Sqrt(dx*dx + dy*dy)
			minDist := a.Radius + b.Radius
			if dist < minDist && dist > 0 {
				overlap := minDist - dist
				nx := dx / dist
				ny := dy / dist
				// Неподвижные мобы (Speed=0: Hive, Block) не сдвигаются —
				// толкаем только соседа на полное перекрытие.
				if a.Speed <= 0 && b.Speed <= 0 {
					continue
				} else if a.Speed <= 0 {
					b.X += nx * overlap
					b.Y += ny * overlap
					clampMobInZone(b)
				} else if b.Speed <= 0 {
					a.X -= nx * overlap
					a.Y -= ny * overlap
					clampMobInZone(a)
				} else {
					// Оба подвижны: делим перекрытие по массам — кто тяжелее,
					// тот сдвигается меньше и «пушит» более лёгкого.
					ma, mbw := mobMass(a), mobMass(b)
					total := ma + mbw
					if total <= 0 {
						total = 1
					}
					a.X -= nx * overlap * (mbw / total)
					a.Y -= ny * overlap * (mbw / total)
					b.X += nx * overlap * (ma / total)
					b.Y += ny * overlap * (ma / total)
					clampMobInZone(a)
					clampMobInZone(b)
				}
			}
		}
	}
}

// collideMobsVsPlayers не даёт мобам наезжать на игрока: при перекрытии
// выталкиваем игрока из моба (моб остаётся на месте, он "тяжёлый и несёт урон").
func (r *Room) collideMobsVsPlayers() {
	for _, p := range r.players {
		if p.Dead || p.transferT > 0 {
			continue
		}
		for i := range r.mobs {
			m := &r.mobs[i]
			if m.HP <= 0 || m.Zone != p.State.Zone {
				continue
			}
			dx := p.State.X - m.X
			dy := p.State.Y - m.Y
			dist := math.Sqrt(dx*dx + dy*dy)
			minDist := m.Radius + p.State.Size
			if dist < minDist && dist > 0 {
				// выталкиваем игрока из моба до касания, небольшим запасом
				overlap := minDist - dist
				nx := dx / dist
				ny := dy / dist
				p.State.X += nx * overlap
				p.State.Y += ny * overlap
				p.State.X, p.State.Y = clampPlayerInZone(p.zone, p.State.X, p.State.Y, p.State.Size)
			}
		}
	}
}

// collideMobsVsGround удалена: дроп-петали больше не имеют коллизий ни с мобами,
// ни с игроками. Подбор — только игроком, по радиусу (pickupPetals в room.go).
