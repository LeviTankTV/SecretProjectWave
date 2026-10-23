package main

import (
	"math"
	"math/rand"
	"strconv"
)

// projectile.go — летящие снаряды петалей: миссил-blade (выстрел по пробелу) и
// осколки-hiveling (разрыв петеля на части, летящие в врага).
// Это независимые сущности (не орбитальные петали): имеют свою позицию, скорость,
// урон и время жизни, бьют мобов на своём пути и исчезают.

type Projectile struct {
	ID      string
	X, Y    float64
	VX, VY  float64
	DMG     float64
	HP      float64 // «здоровье» снаряда (0 = нет механики HP, живёт по Life)
	Size    float64
	Color   uint32
	Kind    string // "blade" | "shard"
	OwnerID string
	OwnerTeam int // команда владельца в момент выстрела (для PvP-взаимодействия)
	Zone    int
	Life    float64
	HitCD   float64
	Pierce  bool // снаряд пролетает сквозь мобов (не исчезает на первом попадании)
	Rarity  int  // редкость петеля-источника (для tier-гейта урона)
}

type ProjectileState struct {
	ID    string  `json:"id"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Angle float64 `json:"angle"`
	Size  float64 `json:"size"`
	Color uint32  `json:"color"`
	Kind  string  `json:"kind"`
}

// newProjectile — создаёт снаряд с вектором скорости, временем жизни, HP
// (0 = нет механики HP) и флагом пробивания (pierce).
func (r *Room) newProjectile(ownerID string, ownerTeam int, zone int, x, y, vx, vy, dmg, hp, size float64, color uint32, kind string, life float64, pierce bool) Projectile {
	r.projIDSeq++
	return Projectile{
		ID:      "proj_" + strconv.Itoa(r.projIDSeq),
		X:       x, Y: y,
		VX:      vx, VY: vy,
		DMG:     dmg,
		HP:      hp,
		Size:    size,
		Color:   color,
		Kind:    kind,
		OwnerID: ownerID,
		OwnerTeam: ownerTeam,
		Zone:    zone,
		Life:    life,
		Pierce:  pierce,
	}
}

// updateProjectiles двигает все снаряды, проверяет попадания по мобам и убирает
// отработавшие. Урон записывается владельцу (топ-дамагер/дроп), убийство — как
// у петалей (awardMobKill).
func (r *Room) updateProjectiles(dt float64) {
	if len(r.proj) == 0 {
		return
	}
	kept := r.proj[:0]
	for i := range r.proj {
		pr := &r.proj[i]
		pr.Life -= dt
		pr.X += pr.VX * dt
		pr.Y += pr.VY * dt
		// Снаряд попал в моба (той же зоны) — урон владельцу, снаряд исчезает.
		hit := false
		for mi := range r.mobs {
			m := &r.mobs[mi]
			if m.HP <= 0 || m.Zone != pr.Zone || m.Showcase {
				continue
			}
			// Снаряды команды мобов (залп Bud) своих не задевают.
			if pr.OwnerTeam == 2 {
				break
			}
			dx := m.X - pr.X
			dy := m.Y - pr.Y
			if math.Sqrt(dx*dx+dy*dy) < m.Radius+pr.Size {
				if pr.HitCD > 0 {
					continue // пробивающий снаряд ещё «в перезарядке» по этому мобу
				}
				// Tier-гейт редкости: Common/Line/Triangle клинки и осколки в 10 раз
				// слабее против мобов редкостью выше своей. Локальный множитель —
				// базовый DMG снаряда не трогаем (пробивной клинок летит дальше,
				// PvP-урон от него не должен быть «загейчен» мобами).
				hitDmg := pr.DMG * petalVsMobMult(pr.Rarity, m.Rarity)
				m.HP -= hitDmg
				if m.Damage == nil {
					m.Damage = map[string]float64{}
				}
				m.Damage[pr.OwnerID] += hitDmg
				if m.HP <= 0 {
					m.KillerID = pr.OwnerID
					r.awardMobKill(m)
				}
				// Пробивающий снаряд с HP изнашивается от удара и исчезает, когда HP кончится.
				if pr.HP > 0 {
					pr.HP -= m.DMG * 0.3
					if pr.HP <= 0 {
						pr.HP = 0
						hit = true // изношен — больше не летит
					}
				}
				if !hit && pr.Pierce {
					pr.HitCD = 0.3 // летит дальше, но не бьёт того же моба каждый кадр
				} else {
					hit = true
				}
				break
			}
		}
		// ПВП: снаряд враждебного игрока (команда 1/2) бьёт вражеских игроков и
		// их петали. Дружелюбный (команда 0) снаряд игроков игнорирует.
		if pr.OwnerTeam != 0 && !hit {
			for _, op := range r.players {
				if op.Dead || op.transferT > 0 || op.State.Zone != pr.Zone || op.State.ID == pr.OwnerID {
					continue
				}
				if !teamAreEnemies(pr.OwnerTeam, op.Team) {
					continue
				}
				// Бьёт тело вражеского игрока.
				dx := op.State.X - pr.X
				dy := op.State.Y - pr.Y
				if math.Sqrt(dx*dx+dy*dy) < op.State.Size+pr.Size {
					if op.GodMode || op.invulnT > 0 {
						continue
					}
					applyPlayerDamage(op, pr.DMG*damageTakenMult(op))
					if pr.HP > 0 {
						pr.HP -= pvpMissileBodyWear
						if pr.HP <= 0 {
							pr.HP = 0
							hit = true
						}
					}
					if !hit && pr.Pierce {
						pr.HitCD = 0.3
					} else {
						hit = true
					}
					if op.State.HP <= 0 {
						op.State.HP = 0
						op.Dead = true
						if cl := r.clients[op.State.ID]; cl != nil {
							cl.deliver([]byte(`{"type":"died"}`))
						}
					}
					break
				}
				// Бьёт петали вражеского игрока (взаимный износ).
				for oi := 0; oi < len(op.petals); oi++ {
					oPet := &op.petals[oi]
					// Аксессуары (eye/line/spike) неуязвимы и не бьются.
					if oPet.HP <= 0 || isOrbitAccessory(oPet.Kind) {
						continue
					}
					ox, oy := petalWorldPos(op, oPet.Angle, oPet.Dist)
					if math.Sqrt((ox-pr.X)*(ox-pr.X)+(oy-pr.Y)*(oy-pr.Y)) < oPet.Size+pr.Size {
						oPet.HP -= pr.DMG
						if pr.HP > 0 {
							pr.HP -= oPet.DMG * 0.3
							if pr.HP <= 0 {
								pr.HP = 0
								hit = true
							}
						}
						if !hit && pr.Pierce {
							pr.HitCD = 0.3
						} else {
							hit = true
						}
						if oPet.HP <= 0 {
							oPet.HP = 0
							orch := petalRechargeCD(op, oPet.Kind, petalRechargeT)
							oPet.Recharge = orch
							oPet.CDMax = orch
						}
						break
					}
				}
				if hit {
					break
				}
			}
		}
		if pr.HitCD > 0 {
			pr.HitCD -= dt
		}
		if !hit && pr.Life > 0 {
			kept = append(kept, *pr)
		}
	}
	r.proj = kept
}

// fireBlades — выстрел всех активных миссил-blade по пробелу: каждый летит по
// касательной к своей орбите в момент нажатия и уходит в перезарядку (кулдаун).
func (r *Room) fireBlades(pid string) {
	p, ok := r.players[pid]
	if !ok || p.Dead || p.transferT > 0 {
		return
	}
	for i := range p.petals {
		pet := &p.petals[i]
		if pet.Kind != "blade" || pet.HP <= 0 || pet.Recharge > 0 {
			continue
		}
		px, py := petalWorldPos(p, pet.Angle, pet.Dist)
		// Клинок направлен «от игрока» (по радиусу орбиты): летит в ту сторону,
		// куда он смотрит в момент нажатия пробела (рукоять у игрока, остриё наружу).
		dirX := math.Cos(pet.Angle)
		dirY := math.Sin(pet.Angle)
		sp := bladeFlySpeed
		r.proj = append(r.proj, r.newProjectile(
			p.State.ID, p.Team, p.State.Zone, px, py,
			dirX*sp, dirY*sp,
			pet.DMG*petalDmgMult, pet.HP, pet.Size*1.4, pet.Color, "blade", bladeFlyLife, true))
		// Петаль САМ вылетает: уходит в полную перезарядку (исчезает с орбиты),
		// а летящий снаряд — это и есть вылетевший клинок. Клинок несёт своё HP
		// и урон, пробивает мобов и изнашивается, пока не кончится HP.
		// cog ускоряет перезарядку всех петалей (в т.ч. способности).
		bidx := len(r.proj) - 1
		r.proj[bidx].Rarity = pet.Rarity
		bch := petalRechargeCD(p, "blade", bladeCooldown)
		pet.Recharge = bch
		pet.CDMax = bch
	}
}

// spawnHivelingShards — при уничтожении петеля-hiveling выпускает шарики,
// разлетающиеся ВО ВСЕ СТОРОНЫ (равномерно по кругу), как и положено овалу.
func (r *Room) spawnHivelingShards(p *Player, pet *Petal) {
	px, py := petalWorldPos(p, pet.Angle, pet.Dist)
	dmg := pet.DMG * petalDmgMult * hivelingShardDmgFrac
	// Базовое направление — случайное, осколки равномерно по 360°.
	base := rand.Float64() * 2 * math.Pi
	for i := 0; i < hivelingShardCount; i++ {
		a := base + (float64(i)/float64(hivelingShardCount))*2*math.Pi
		si := len(r.proj)
		r.proj = append(r.proj, r.newProjectile(
			p.State.ID, p.Team, p.State.Zone, px, py,
			math.Cos(a)*hivelingShardSpeed, math.Sin(a)*hivelingShardSpeed,
			dmg, 0, pet.Size*1.2, pet.Color, "shard", hivelingShardLife, true))
		r.proj[si].Rarity = pet.Rarity
	}
}

// spawnBudBurst (порт родителя) — Bud выпускает 8 лепестков-снарядов строго
// по 360°. Снаряды команды 2 (мобы): бьют игроков, своих мобов не задевают.
func (r *Room) spawnBudBurst(m *Mob) {
	const burstCount = 8
	dmg := m.DMG * budBurstDmgFrac
	hp := m.DMG * budShotHPFrac
	for i := 0; i < burstCount; i++ {
		a := (float64(i) / float64(burstCount)) * 2 * math.Pi
		si := len(r.proj)
		r.proj = append(r.proj, r.newProjectile(
			m.ID, 2, m.Zone, m.X, m.Y,
			math.Cos(a)*budBurstSpeed, math.Sin(a)*budBurstSpeed,
			dmg, hp, m.Radius*0.55, mobTypes[m.Type].Color, "budshot", budBurstLife, true))
		r.proj[si].Rarity = m.Rarity
	}
}

// nearestMob — ближайший живой моб зоны к точке (x, y).
func nearestMob(mobs []Mob, zone int, x, y float64) *Mob {
	var best *Mob
	bestD := math.MaxFloat64
	for i := range mobs {
		m := &mobs[i]
		if m.HP <= 0 || m.Zone != zone {
			continue
		}
		dx := m.X - x
		dy := m.Y - y
		d := dx*dx + dy*dy
		if d < bestD {
			bestD = d
			best = m
		}
	}
	return best
}
