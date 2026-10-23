package main

import (
	"math"
)

// xp.go - система уровней: опыт, прокачка, максимум петалей по уровню.

// petalCapForLevel возвращает максимум орбитальных петалей на данном уровне
// (кривая родителя, порт): ранние капы наступают быстрее — волны короткие,
// второй слот должен открываться в первом же забеге.
//   уровень 0  -> 5 петалей
//   уровень 2  -> 6
//   уровень 4  -> 7
//   уровень 8  -> 8
//   уровень 16 -> 9
//   уровень 25 -> 10
func petalCapForLevel(level int) int {
	switch {
	case level >= 25:
		return 10
	case level >= 16:
		return 9
	case level >= 8:
		return 8
	case level >= 4:
		return 7
	case level >= 2:
		return 6
	default:
		return 5
	}
}

// Кривая опыта (порт родителя, XP_DESIGN): геометрия 45×1.11^L. Раньше была
// линейная 30+12×L — с волновым режимом (короткие забеги, боссы каждые 10
// волн) линейка исчерпывалась к 20-му уровню и дальше «не чувствовалась».
const (
	xpCurveBase     = 45.0
	xpCurveGrowth   = 1.11
	xpLevelSanityCap = 1000
	// xpMaxLevel — МАКСИМУМ уровня (мягкий кап): 8 тиров × 8 уровней под
	// зонную лестницу редкостей. После него уровень не растёт.
	xpMaxLevel = 60
	// Тиры опыта: 8 уровней на тир, 8 тиров — под редкости Point..Chaos.
	xpLevelsPerTier = 8
	xpTiers         = 8
	// Штраф за каждый тир разрыва между тиром игрока и редкостью моба.
	xpTierGapStep = 0.65
	// Опыт за моба БАЗОВОЙ редкости (Point), до множителей.
	baseKillXP = 8.0
	// «Сложный» тип моба (толстый и бьющий) даёт больше.
	mobTypeXPMult = 1.5
	mobToughPower = 200.0
)

// xpForLevel возвращает, сколько опыта нужно, чтобы ПРОКАЧАТЬСЯ с level на
// level+1 (порт родителя).
func xpForLevel(level int) float64 {
	if level < 0 {
		level = 0
	}
	if level > xpLevelSanityCap {
		level = xpLevelSanityCap
	}
	return math.Round(xpCurveBase * math.Pow(xpCurveGrowth, float64(level)))
}

// totalXPForLevel — сколько опыта накоплено к моменту первого достижения
// уровня level. Используется миграцией на новую кривую.
func totalXPForLevel(level int) float64 {
	if level <= 0 {
		return 0
	}
	if level > xpLevelSanityCap {
		level = xpLevelSanityCap
	}
	var total float64
	for l := 0; l < level; l++ {
		total += xpForLevel(l)
	}
	return total
}

// levelForTotalXP — обратная к totalXPForLevel: (уровень, остаток XP) по
// накопленному опыту. Уровень не превышает мягкий кап.
func levelForTotalXP(total float64) (int, float64) {
	if total <= 0 {
		return 0, 0
	}
	level := 0
	for level < xpMaxLevel && level < xpLevelSanityCap {
		need := xpForLevel(level)
		if total < need {
			break
		}
		total -= need
		level++
	}
	return level, total
}

// applyXPGain — начисление прироста к паре (level, xp) с капом 60 (для
// меню-путей, где нет живого Player).
func applyXPGain(level int, xp, gain float64) (int, float64) {
	total := totalXPForLevel(level) + xp + gain
	return levelForTotalXP(total)
}

// killXPBase — «сырой» опыт за моба редкости rar (без штрафа тиров).
func killXPBase(rarity int) float64 {
	if rarity < 0 {
		rarity = 0
	}
	if rarity >= len(rarityDanger) {
		rarity = len(rarityDanger) - 1
	}
	return baseKillXP * rarityDanger[rarity]
}

// killXPRounded — эталонная цена «убийства моба редкости rar» (для потолка
// абсорба): сырая база, округлённая.
func killXPRounded(rarity int) float64 {
	return math.Round(killXPBase(rarity))
}

// tierGap — множитель анти-фарма (порт родителя): редкости ниже тира игрока
// дают ×0.65 за каждый тир разрыва (пол 0.02). Тир игрока = level/8.
func tierGap(level, mobRarity int) float64 {
	myTier := level / xpLevelsPerTier
	if myTier < 0 {
		myTier = 0
	}
	if myTier > xpTiers-1 {
		myTier = xpTiers - 1
	}
	if mobRarity >= myTier {
		return 1.0
	}
	return math.Max(0.02, math.Pow(xpTierGapStep, float64(myTier-mobRarity)))
}

// killXPBeforeTierGap — опыт за убийство до штрафа тиров: база по редкости ×
// множитель «сложного» типа.
func killXPBeforeTierGap(m *Mob) float64 {
	if m == nil {
		return 0
	}
	xp := killXPBase(m.Rarity)
	mt := mobTypes[m.Type]
	if mt.BaseHP*mt.BaseDMG > mobToughPower {
		xp *= mobTypeXPMult
	}
	return xp
}

// killXPFor — опыт за убийство моба для игрока данного уровня
// (порт родителя): детерминированный, без случайного разброса. Минимум 1 XP.
func killXPFor(m *Mob, level int) float64 {
	if m == nil {
		return 0
	}
	return math.Max(1, math.Round(killXPBeforeTierGap(m)*tierGap(level, m.Rarity)))
}



// awardXP начисляет опыт игроку (порт родителя): мягкий кап 60 — после него
// уровень не растёт (XP на капе не копится в прогресс-баре; источник истины
// xp_total появится с магазином/осколками, этап 9). Возвращает true при левелапе.
func (p *Player) awardXP(xp float64) bool {
	if p.State.Level >= xpMaxLevel {
		p.State.XPMax = xpForLevel(p.State.Level)
		return false
	}
	p.State.XP += xp
	leveledUp := false
	for p.State.Level < xpMaxLevel {
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
