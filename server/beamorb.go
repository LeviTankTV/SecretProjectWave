package main

import (
	"math"
	"strconv"
)

// beamorb.go — энергетический шар моба Beam (зона Line): летящий снаряд со
// СВОИМ HP, который «доводится» (лёгкий хоминг) до ближайшего враждебного
// игрока. Шар: наносит урон телу игрока при касании (и разбивается), при этом
// сам получает урон от петалей игрока, его снарядов, миньонов и «шипового
// тела» (spike). Клиент видит его как ProjectileState с Kind "beamorb".

const (
	beamOrbSpeed  = 190.0  // скорость полёта шара
	beamOrbSize   = 16.0   // радиус шара (ребаланс: крупный энергетический шар)
	beamOrbLife   = 6.0    // время жизни (сек)
	beamOrbTurn   = 2.6    // скорость поворота вектора к цели (рад/с) — «доведение»
	beamOrbHPBase = 6.0    // HP шара r0 (далее × rarityDanger редкости Beam)
	beamShootBase = 4.0    // средняя пауза между выстрелами Beam (случайная ±)
)

// BeamOrb — летящий энергетический шар Beam.
type BeamOrb struct {
	ID        string
	X, Y      float64
	VX, VY    float64
	DMG       float64
	HP, MaxHP float64
	Size      float64
	Color     uint32
	Zone      int
	Rarity    int // редкость родителя-Beam (для tier-гейта яда и т.п.)
	Life      float64
	SpikeT    float64 // тик «шипового тела» игрока против шара
	PoisonDPS float64 // единый DoT-пул яда (viper/shard) на шаре
	PoisonT   float64
}

// spawnBeamOrb — Beam выпускает шар в сторону цели (шар потом сам доводится).
func (r *Room) spawnBeamOrb(m *Mob, target *Player) {
	hp := beamOrbHPBase * rarityDanger[minInt(m.Rarity, len(rarityDanger)-1)]
	r.orbIDSeq++
	r.orbs = append(r.orbs, BeamOrb{
		ID: "orb_" + strconv.Itoa(r.orbIDSeq),
		X:  m.X, Y: m.Y,
		VX: math.Cos(math.Atan2(target.State.Y-m.Y, target.State.X-m.X)) * beamOrbSpeed,
		VY: math.Sin(math.Atan2(target.State.Y-m.Y, target.State.X-m.X)) * beamOrbSpeed,
		DMG:    m.DMG,
		HP:     hp, MaxHP: hp,
		Size:   beamOrbSize,
		Color:  mobTypes[m.Type].Color,
		Zone:   m.Zone,
		Rarity: m.Rarity,
		Life:   beamOrbLife,
	})
}

// beamHomingTarget — ближайший живой враждебный игрок зоны (вне Safe Zone),
// к которому «доводится» шар. nil — цель пропала, шар летит по прямой.
func (r *Room) beamHomingTarget(o *BeamOrb, zoneBottom float64) *Player {
	var tp *Player
	best := math.MaxFloat64
	for _, p := range r.players {
		if p.State.Zone != o.Zone || p.Dead || p.transferT > 0 || p.Passive || !mobAggrosPlayer(p.Team) {
			continue
		}
		if p.State.Y > zoneBottom {
			continue
		}
		dx := p.State.X - o.X
		dy := p.State.Y - o.Y
		if d := dx*dx + dy*dy; d < best {
			best = d
			tp = p
		}
	}
	return tp
}

// updateBeamOrbs двигает шары, включает хоминг, обрабатывает все столкновения:
// с телом игрока (шар наносит урон и разбивается), с петалем игрока (игрок
// режет шар), с миньонами и снарядами игрока (шар получает урон), с шиповым
// телом (spike). Шар гибнет о границу Safe Zone (мобы туда не агрятся).
//
// Шары уязвимы ВСЕМ У игрока: каждый петаль бьёт шар по СВОЕЙ каденции удара
// (OrbT = petalAtkCD вида, без затрагивания боевого AtkT/Charging), ядовитые
// петали вешают на шар DoT, ball бомбит вокруг, spark получает шанс цепи,
// миньоны и снаряды бьют шар, как моба. Убитый (HP<=0) шар гаснет в следующий
// тик. Плюс на шары действуют сплэш-эффекты от ударов по МОБАМ (bombSplash,
// sparkChain, runeBoom) — см. chain.go/room.go.
func (r *Room) updateBeamOrbs(dt float64) {
	if len(r.orbs) == 0 {
		return
	}
	// Общая каденция ударов петалей по шарам: тикаем OrbT ВСЕХ петалей раз за
	// тик (независимо от числа орбов/контактов), чтобы каждый петаль мог
	// ударить во время контакта не чаще petalAtkCD.
	for _, p := range r.players {
		for i := range p.petals {
			if p.petals[i].OrbT > 0 {
				p.petals[i].OrbT -= dt
				if p.petals[i].OrbT < 1e-6 {
					p.petals[i].OrbT = 0
				}
			}
		}
	}
	kept := r.orbs[:0]
	for i := range r.orbs {
		o := &r.orbs[i]
		o.Life -= dt
		if o.Life <= 0 || o.HP <= 0 {
			continue
		}
		z := zones[o.Zone]
		zoneBottom := mobZoneBottom(&z)

		// Хоминг: плавное доведение вектора скорости до ближайшего игрока.
		if tp := r.beamHomingTarget(o, zoneBottom); tp != nil {
			dx := tp.State.X - o.X
			dy := tp.State.Y - o.Y
			ln := math.Sqrt(dx*dx + dy*dy)
			if ln > 0 {
				wx, wy := dx/ln, dy/ln
				spd := math.Sqrt(o.VX*o.VX + o.VY*o.VY)
				if spd == 0 {
					spd = beamOrbSpeed
				}
				ca := math.Atan2(o.VY, o.VX)
				na := math.Atan2(wy, wx)
				da := na - ca
				for da > math.Pi {
					da -= 2 * math.Pi
				}
				for da < -math.Pi {
					da += 2 * math.Pi
				}
				step := beamOrbTurn * dt
				if da > step {
					da = step
				}
				if da < -step {
					da = -step
				}
				oa := ca + da
				o.VX = math.Cos(oa) * beamOrbSpeed
				o.VY = math.Sin(oa) * beamOrbSpeed
			}
		}

		o.X += o.VX * dt
		o.Y += o.VY * dt
		o.X, o.Y = clampZoneRounded(&z, o.X, o.Y, o.Size)
		// Шар не залетает в Safe Zone (как мобы): гаснет у границы.
		if o.Y > zoneBottom-o.Size {
			continue
		}
		dead := false

		for _, p := range r.players {
			if p.Dead || p.transferT > 0 || p.State.Zone != o.Zone || !mobAggrosPlayer(p.Team) {
				continue
			}
			if p.State.Y > zoneBottom {
				continue
			}
			dx := p.State.X - o.X
			dy := p.State.Y - o.Y
			dBody := math.Sqrt(dx*dx + dy*dy)

			// Петали игрока (не аксессуары) режут шар: каждый контактирующий
			// петаль бьёт по своей каденции OrbT. Ядовитые (viper/shard) вешают
			// на шар DoT, ball бомбит вокруг точки касания, spark — с шара может
			// проскочить цепь на ближайшего живущего врага.
			if !dead {
				for pi := range p.petals {
					pet := &p.petals[pi]
					if pet.HP <= 0 || pet.Recharge > 0 || isOrbitAccessory(pet.Kind) {
						continue
					}
					px, py := petalWorldPos(p, pet.Angle, pet.Dist)
					edx := px - o.X
					edy := py - o.Y
					if math.Sqrt(edx*edx+edy*edy) >= pet.Size+o.Size || pet.OrbT > 0 {
						continue
					}
					pet.OrbT = petalAtkCD(pet.Kind)
					dmg := pet.DMG * petalDmgMult
					// Tier-гейт редкости: Common/Line/Triangle петали в 10 раз
					// слабее против шаров Beam редкостью выше своей.
					dmg *= petalVsMobMult(pet.Rarity, o.Rarity)
					o.HP -= dmg
					if o.HP <= 0 {
						dead = true
						break
					}
					// Лайфстил (oval): лечит игрока от урона по шару.
					if petalKinds[pet.Kind].Lifesteal {
						p.State.HP += dmg * lifestealPct
						if p.State.HP > p.State.MaxHP {
							p.State.HP = p.State.MaxHP
						}
					}
					// Яд (viper): tier-гейт по редкости Beam (шар её унаследовал).
					if petalKinds[pet.Kind].Poison && pet.DMG > 0 {
						pps := dmg * poisonDpsMult
						if o.Rarity > pet.Rarity {
							pps *= poisonCrossTierMult
						}
						if pps > o.PoisonDPS {
							o.PoisonDPS = pps
						}
						o.PoisonT = poisonDuration
					}
					// Рана (shard): короткий DoT на шаре.
					if petalKinds[pet.Kind].Wound && pet.DMG > 0 {
						if s := dmg * woundDpsMult; s > o.PoisonDPS {
							o.PoisonDPS = s
						}
						o.PoisonT = woundDuration
					}
					// Цепь (spark): цепь может прыгнуть и с самого шара.
					if petalKinds[pet.Kind].Chain {
						r.sparkChain(p.State.ID, o.Zone, o.X, o.Y, dmg, o.ID)
					}
					// Бомба (ball): мини-АОЕ вокруг точки касания шара.
					if petalKinds[pet.Kind].Bomb {
						r.bombSplash(p.State.ID, o.Zone, o.X, o.Y, dmg, o.ID)
					}
				}
			}
			if dead {
				break
			}
			// Тело игрока: шар бьёт игрока (и разбивается); шиповое тело игрока
			// попутно колет шар.
			if dBody < p.State.Size+o.Size {
				if o.SpikeT <= 0 {
					if sd := bodySpikeDmg(p); sd > 0 {
						o.HP -= sd
						o.SpikeT = spikeGrindCD
					}
				}
				if o.HP <= 0 {
					dead = true
				}
				if !dead && p.invulnT <= 0 && !p.GodMode && p.mobHitCD <= 0 {
					applyPlayerDamage(p, o.DMG*damageTakenMult(p))
					p.mobHitCD = playerHitCD
					if p.State.HP <= 0 {
						p.State.HP = 0
						p.Dead = true
						if cl := r.clients[p.State.ID]; cl != nil {
							cl.deliver([]byte(`{"type":"died"}`))
						}
					}
					dead = true // шар разбивается о тело
				}
			}
			if dead {
				break
			}
		}
		// Миньоны игрока (entropy) бьют шар, как мобов.
		if !dead {
			for mi := range r.minions {
				mn := &r.minions[mi]
				if !mn.Alive || mn.Zone != o.Zone {
					continue
				}
				dx := mn.X - o.X
				dy := mn.Y - o.Y
				if math.Sqrt(dx*dx+dy*dy) < mn.Size+o.Size {
					if mn.HitCD <= 0 {
						mn.HitCD = petalDamageCD
						o.HP -= mn.DMG
						if o.HP <= 0 {
							dead = true
						}
					}
					break
				}
			}
		}
		// Снаряды игрока (миссил-blade, осколки и т.д.) стреляют по шару.
		if !dead {
			for pi := range r.proj {
				pr := &r.proj[pi]
				if pr.Zone != o.Zone || pr.DMG <= 0 {
					continue
				}
				dx := pr.X - o.X
				dy := pr.Y - o.Y
				if math.Sqrt(dx*dx+dy*dy) < pr.Size+o.Size {
					o.HP -= pr.DMG
					if o.HP <= 0 {
						dead = true
					}
					break
				}
			}
		}
		// Яд (viper/shard) тикает на шаре: единый DoT-пул, гаснет с таймером.
		if !dead {
			if dmg := poisonSteps(&o.PoisonDPS, &o.PoisonT, dt); dmg > 0 {
				o.HP -= dmg
				if o.HP <= 0 {
					dead = true
				}
			}
		}
		if o.SpikeT > 0 {
			o.SpikeT -= dt
		}
		if dead {
			continue
		}
		kept = append(kept, *o)
	}
	r.orbs = kept
}
