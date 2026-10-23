package main

import "math/rand"

// poison.go — Яд (механика «гадюка»). Петали-виды с PetalKind.Poison при каждом
// ударе вешают на цель ЕДИНЫЙ DoT-пул: урон раз в тик, пока не вышел таймер.
// Пул НЕ стакается: повторный удар освежает таймер (до poisonDuration с) и
// УДЕРЖИВАЕТ максимальную силу из всех наложений (take-max), не складывая ДПС
// от нескольких гадюк и не ослабляя яд слабым свежим ударом.
//
// Яд действует на три класса целей:
//   - мобы: убийство от яда засчитывается отравителю (XP, дроп по топ-4, увед.),
//   - орбитальные петали боссов (ПВЕ): убитая ядом петаль уходит в Respawn,
//   - петали других игроков (ПВП): убитая ядом петаль уходит в Recharge.

// poisonMob вешает/освежает яд на мобе от петеля-«гадюки».
func poisonMob(m *Mob, dps float64, by string) {
	if dps <= 0 {
		return
	}
	// Take-max: невозможно усилить стак добавкой слабого удара; держим
	// сильнейшую из наложенных сил. Убийство/дроп — на последнего отравителя.
	if dps > m.PoisonDps {
		m.PoisonDps = dps
	}
	m.PoisonT = poisonDuration
	m.PoisonBy = by
}

// poisonBossPetal вешает/освежает яд на орбитальной петали босса.
func poisonBossPetal(bp *BossPetal, dps float64) {
	if bp.HP <= 0 || dps <= 0 {
		return
	}
	if dps > bp.PoisonDps {
		bp.PoisonDps = dps
	}
	bp.PoisonT = poisonDuration
}

// poisonPlayerPetal вешает/освежает яд на петали другого игрока (ПВП).
func poisonPlayerPetal(pet *Petal, dps float64) {
	// Аксессуары не токсифицируются.
	if pet.HP <= 0 || dps <= 0 || isOrbitAccessory(pet.Kind) {
		return
	}
	if dps > pet.PoisonDps {
		pet.PoisonDps = dps
	}
	pet.PoisonT = poisonDuration
}

// poisonSteps уменьшает таймер яда на dt и возвращает урон за этот тик.
// Когда таймер вышел — сила яда обнуляется (пул «иссяк»).
func poisonSteps(dps *float64, t *float64, dt float64) float64 {
	if *t <= 0 || *dps <= 0 {
		return 0
	}
	dmg := *dps * dt
	*t -= dt
	if *t <= 0 {
		*t = 0
		*dps = 0
	}
	return dmg
}

// applyPoison тикает яд всех отравленных сущностей за кадр: мобов (урон пишется
// в Damage-карту для дропа/наград), орбитальных петалей боссов и петалей игроков.
func (r *Room) applyPoison(dt float64) {
	// Мобы.
	for i := range r.mobs {
		m := &r.mobs[i]
		if m.HP <= 0 || m.Showcase {
			continue
		}
		dmg := poisonSteps(&m.PoisonDps, &m.PoisonT, dt)
		if dmg <= 0 {
			continue
		}
		m.HP -= dmg
		if m.PoisonBy != "" {
			m.Damage[m.PoisonBy] += dmg
		}
		if m.HP <= 0 {
			m.HP = 0
			if m.PoisonBy != "" {
				m.KillerID = m.PoisonBy
			}
			r.awardMobKill(m)
		}
	}
	// Орбитальные петали боссов.
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
			dmg := poisonSteps(&bp.PoisonDps, &bp.PoisonT, dt)
			if dmg <= 0 {
				continue
			}
			bp.HP -= dmg
			if bp.HP <= 0 {
				bp.HP = 0
				bp.PoisonDps = 0
				bp.PoisonT = 0
				bp.Respawn = mobPetalsRespawnMin +
					rand.Float64()*(mobPetalsRespawnMax-mobPetalsRespawnMin)
			}
		}
	}
	// Петали игроков (яд от чужих «гадюк» в ПВП).
	for _, p := range r.players {
		for i := range p.petals {
			pet := &p.petals[i]
			// Аксессуары (eye/line/spike) не токсифицируются и не ломаются.
			if pet.HP <= 0 || isOrbitAccessory(pet.Kind) {
				continue
			}
			dmg := poisonSteps(&pet.PoisonDps, &pet.PoisonT, dt)
			if dmg <= 0 {
				continue
			}
			pet.HP -= dmg
			if pet.HP <= 0 {
				pet.HP = 0
				pet.PoisonDps = 0
				pet.PoisonT = 0
				pet.Recharge = petalRechargeCD(p, petalRechargeT)
				pet.CDMax = pet.Recharge
				// Разрыв (hiveling): петаль, добитая ядом, тоже выстреливает осколки.
				if petalKinds[pet.Kind].Split {
					r.spawnHivelingShards(p, pet)
				}
				// Слеза (tear): разрушенная петаль лечит игрока.
				if petalKinds[pet.Kind].Tear {
					p.State.HP += p.State.MaxHP * tearHealFrac(pet.Rarity)
					if p.State.HP > p.State.MaxHP {
						p.State.HP = p.State.MaxHP
					}
				}
			}
		}
	}
}
