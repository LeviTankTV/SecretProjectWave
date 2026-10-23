package main

import (
	"math/rand"
	"strconv"
	"time"
)

// ---- Крафт петалей (florr.io style) ----
// Из 5 петалей ОДНОЙ редкости и ОДНОГО вида пытаемся получить 1 петаль
// той же редкости+1. Шанс успеха зависит от исходной редкости.
// Индексы редкости: 0 Point, 1 Line, 2 Triangle, 3 Square, 4 Pentagon,
// 5 Hexagon, 6 Fractal, 7 Chaos, 8..15 — истинная лестница Dot..Origin.

// Шанс успеха (%) для перехода rarity -> rarity+1 — единый реестр:
// server/rarity (rarity_registry.go), 15 переходов, включая Chaos→Dot 4%.
// «Брутально редкий топ»: Point→Line 20%, Hexagon→Fractal 1%, Fractal→Chaos 1%.

// craftableRarity сообщает, можно ли крафтить пёталь этой редкости (есть ли следующая).
func craftableRarity(rarity int) bool {
	return rarity >= 0 && rarity <= maxCraftFrom && rarity < len(craftChancePct)
}

// craftSuccessChance возвращает шанс успеха (%) для редкости, или -1 если нельзя.
func craftSuccessChance(rarity int) float64 {
	if !craftableRarity(rarity) {
		return -1
	}
	return craftChancePct[rarity]
}

// makeCraftPetal создаёт петаль следующей редкости (вид сохраняется).
// Статы — как у обычного петаля этой редкости/вида (BaseDMG/BaseHP * petalPower).
// Для высоких тиров (крафт в Hexagon/Fractal/Chaos, next >= 5) действует
// ГАРАНТИРОВАННЫЙ минимум урона: чуть выше МАКСИМУМА предыдущего тира
// (15.75 * petalPower[next-1] > 15 * petalPower[next-1] — максимум урона любого
// вида предыдущего тира). Иначе скрафтивший Chaos из слабого вида (rock, oval…)
// получил бы петаль слабее, чем Fractal сильного вида — и «тир» переставал
// бы доминировать.
func makeCraftPetal(kind string, rarity int) PetalDef {
	return makeCraftPetalBy(kind, rarity, "")
}

// makeCraftPetalBy — с провенансом (ник автора; порт родителя).
func makeCraftPetalBy(kind string, rarity int, by string) PetalDef {
	return makeCraftPetalGraded(kind, rarity, by, -1)
}

// makeCraftPetalGraded — результат крафта с грейдом Pot (порт родителя):
// пол и база считаются БЕЗ множителя, грейд — после. ОДИН грейд на предмет
// множит и HP/DMG, и силу пассивки (Eff = Pot). forceGrade >= 0 — ручной
// грейд из админки (тест PRNG) с тем же потолком тира.
func makeCraftPetalGraded(kind string, rarity int, by string, forceGrade int) PetalDef {
	kn := petalKinds[kind]
	next := rarity + 1
	dmg := kn.BaseDMG * petalPower[next]
	// Пол урона НЕ применяется к видам, у которых нет базового урона вообще
	// (спавн-петали вроде Entropy: призывают миньонов, сами не бьют). Иначе
	// крафт в Hexagon+ «вернул» бы им урон через минимальный порог.
	if kn.BaseDMG > 0 && next >= 5 {
		floor := 15.75 * petalPower[next-1] // чуть выше max урона предыдущего тира
		if dmg < floor {
			dmg = floor
		}
	}
	hp := kn.BaseHP * petalPower[next]
	pot := rollPot(next)
	if forceGrade >= 0 {
		if max := gradeMaxFor(next); forceGrade > max {
			forceGrade = max
		}
		pot = gradePct[forceGrade]
	}
	eff := pot
	k := potMult(pot)
	dmg *= k
	hp *= k
	return PetalDef{
		ID:        "craft_" + strconv.Itoa(rand.Int()),
		Kind:      kind,
		Rarity:    next,
		HP:        hp,
		MaxHP:     hp,
		DMG:       dmg,
		Size:      petalSizeBase,
		Color:     groundPetalColor(next),
		Mass:      petalDefMass(next),
		Name:      kn.Name,
		Pot:       pot,
		Eff:       eff,
		CraftedBy: by,
		CraftedAt: time.Now().Unix(),
	}
}

// CraftBatch — исход одного запуска катушки (одна попытка из 5 петалей).
// Клиент проигрывает эти батчи по очереди, по ~5 секунд каждый.
type CraftBatch struct {
	Success bool `json:"success"`
	Lost    int  `json:"lost"`
}
