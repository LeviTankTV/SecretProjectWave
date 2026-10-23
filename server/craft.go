package main

import (
	"math/rand"
	"strconv"
)

// ---- Крафт петалей (florr.io style) ----
// Из 5 петалей ОДНОЙ редкости и ОДНОГО вида пытаемся получить 1 петаль
// той же редкости+1. Шанс успеха зависит от исходной редкости.
// Индексы редкости: 0 Point, 1 Line, 2 Triangle, 3 Square, 4 Polygon,
// 5 Hexagon, 6 Fractal, 7 Chaos.

// Шанс успеха (%) для перехода rarity -> rarity+1. Из Chaos (индекс 7) крафта нет.
// Все шансы в 2 раза ниже прежних (новая экономика: редкое дороже).
// «Брутально редкий топ»: Point→Line 20%, Hexagon→Fractal 1%, Fractal→Chaos 1%.
var craftChancePct = []float64{20.0, 12.5, 7.5, 4.0, 2.0, 1.0, 1.0}

// craftableRarity сообщает, можно ли крафтить пёталь этой редкости (есть ли следующая).
func craftableRarity(rarity int) bool {
	return rarity >= 0 && rarity < len(craftChancePct)
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
	return PetalDef{
		ID:     "craft_" + strconv.Itoa(rand.Int()),
		Kind:   kind,
		Rarity: next,
		HP:     hp,
		MaxHP:  hp,
		DMG:    dmg,
		Size:   petalSizeBase,
		Color:  groundPetalColor(next),
		Mass:   petalDefMass(next),
		Name:   kn.Name,
	}
}

// CraftBatch — исход одного запуска катушки (одна попытка из 5 петалей).
// Клиент проигрывает эти батчи по очереди, по ~5 секунд каждый.
type CraftBatch struct {
	Success bool `json:"success"`
	Lost    int  `json:"lost"`
}
