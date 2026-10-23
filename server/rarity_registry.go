package main

import (
	"strconv"
	"strings"

	"secretproject/server/rarity"
)

//go:generate go run ./cmd/genrarities

// Единый реестр редкостей (пакет server/rarity) подключён в package main под
// прежними идентификаторами, чтобы не менять многочисленные вызовы по коду.
// Порт из родительского SecretProject (адаптация 2.0): 16 редкостей —
// зонная лестница Point..Chaos (0..7) + истинная Dot..Origin (8..15).
var (
	rarityNames    = rarity.Names
	rarityColors   = rarity.Colors
	rarityDanger   = rarity.Danger
	craftChancePct = rarity.CraftChancePct
	// petalPower — копия: config-переопределения (этап 10) смогут менять её.
	petalPower = append([]float64(nil), rarity.Power...)
)

// Якоря лестницы: семантика «топ-редкость» у Wave раньше выражалась как
// len(rarityNames)-1/-2 — с 16 редкостями это стало бы Origin/Mirror и сломало
// бы боссов/анонсы. Явные константы сохраняют прежнее поведение.
const (
	rarityFractal = 6 // топ спавнящихся боссов Fractal-тира
	rarityChaos   = 7 // потолок зонной лестницы (Chaos)
)

// rarityWireColor — 24-битный цвет редкости (тот же HEX, что уезжает клиенту в
// сгенерированном client/js/rarities.js). Нужен там, где цвет берётся не из
// groundPetalColor: у той палитры только 8 зонных ступеней, и «истинные»
// редкости 8..15 (Dot..Origin) выпадали в белый.
func rarityWireColor(r int) uint32 {
	if len(rarityColors) == 0 {
		return petalColor
	}
	if r < 0 {
		r = 0
	}
	if r >= len(rarityColors) {
		r = len(rarityColors) - 1
	}
	n, err := strconv.ParseUint(strings.TrimPrefix(rarityColors[r], "#"), 16, 32)
	if err != nil || n == 0 || n > 0xffffff {
		return petalColor
	}
	return uint32(n)
}
