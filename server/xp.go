package main

import (
	"math"
	"math/rand"
)

// xp.go - система уровней: опыт, прокачка, максимум петалей по уровню.

// petalCapForLevel возвращает максимум орбитальных петалей на данном уровне.
// Уровни набираются порогами:
//   уровень 0  -> 5 петалей
//   уровень 5  -> 6
//   уровень 10 -> 7
//   уровень 16 -> 8
//   уровень 25 -> 9
//   уровень 50 -> 10
func petalCapForLevel(level int) int {
	switch {
	case level >= 50:
		return 10
	case level >= 25:
		return 9
	case level >= 16:
		return 8
	case level >= 10:
		return 7
	case level >= 5:
		return 6
	default:
		return 5
	}
}

// xpForLevel возвращает, сколько опыта нужно, чтобы ПРОКАЧАТЬСЯ с level на level+1.
// Формула растёт с уровнем: база + линейный рост.
func xpForLevel(level int) float64 {
	return math.Floor(30 + float64(level)*12)
}

// mobXP возвращает награду опыта за убитую моба.
// Чем выше редкость и опасность, тем больше опыта.
func mobXP(m *Mob) float64 {
	// База: 10, растёт с редкостью и типом моба.
	mt := mobTypes[m.Type]
	base := 10.0
	// Каждая редкость умножает опыт.
	xp := base * rarityDanger[m.Rarity]
	// Сложный тип даёт чуть больше.
	if mt.BaseHP*mt.BaseDMG > 200 {
		xp *= 1.5
	}
	// Маленький случайный разброс.
	xp *= 0.9 + rand.Float64()*0.2
	return xp
}

// awardXP начисляет опыт игроку. Возвращает true, если уровень повысился.
func (p *Player) awardXP(xp float64) bool {
	p.State.XP += xp
	leveledUp := false
	for {
		need := xpForLevel(p.State.Level)
		if p.State.XP >= need {
			p.State.XP -= need
			p.State.Level++
			leveledUp = true
		} else {
			break
		}
	}
	p.State.XPMax = xpForLevel(p.State.Level)
	return leveledUp
}

// ensurePetalCap обрезает петалей игрока до максимума уровня (если надо).
// Вызывается после повышения уровня или при смене лимита.
func (p *Player) ensurePetalCap() {
	cap := petalCapForLevel(p.State.Level)
	if len(p.petals) > cap {
		p.petals = p.petals[:cap]
	}
}
