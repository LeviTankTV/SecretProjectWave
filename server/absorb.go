package main

import "math"

// absorb.go — Абсорб: уничтожение петалей инвентаря за опыт (XP).
// Полная противоположность крафта: петали (несколько видов/редкостей сразу)
// списываются навсегда, а игрок получает за них XP. Чем реже и сильнее петаль,
// тем больше опыта.

type absorbItem struct {
	Kind   string `json:"kind"`
	Rarity int    `json:"rarity"`
	Count  int    `json:"count"`
}

// Абсорб-цена петаля растёт в ГЕОМЕТРИЧЕСКОЙ ПРОГРЕССИИ по редкости:
// каждый шаг редкости даёт ровно absorbXPRatio × больше опыта
// (по умолчанию редкость +1 = вдвое больше XP). Cостав — «сила» вида
// (петаль без учёта редкости: BaseDMG*2 + BaseHP), от неё берём долю
// absorbXPCoeff и умножаем на 2^редкость.
const (
	absorbXPRatio  = 2.0 // геометрический множитель за каждый шаг редкости
	absorbXPCoeff  = 0.6 // доля «силы» за петаль базовой редкости (0)
)

// petalStackXP — XP за ОДИН петаль. Статы (dmg/maxHp) нормализуются обратно в
// «базовые» (без множителя редкости), чтобы цена зависела только от вида и
// редкости в чистом виде: base = (baseDMG*2 + baseHP) * coeff * ratio^rarity.
// Деление на множитель редкости делает формулу невосприимчивой к крафтовым
// «полам» урона — клиентский превью (то же деление) совпадает с фактом.
func petalStackXP(d PetalDef) float64 {
	rar := d.Rarity
	if rar < 0 {
		rar = 0
	}
	mult := petalPower[minInt(rar, len(petalPower)-1)]
	if mult <= 0 {
		mult = 1
	}
	baseDMG := d.DMG / mult
	baseHP := d.MaxHP / mult
	base := (baseDMG*2 + baseHP) * absorbXPCoeff
	if base <= 0 {
		return 0
	}
	return base * math.Pow(absorbXPRatio, float64(rar))
}

// absorb уничтожает указанные петали из инвентаря и возвращает суммарный XP
// (считает только реально списанные, по доступному количеству в стаках).
func (sp *statelessProfile) absorb(items []absorbItem) float64 {
	var total float64
	for _, it := range items {
		i := sp.findStack(it.Kind, it.Rarity)
		if i < 0 || sp.inv[i].Count <= 0 || it.Count <= 0 {
			continue
		}
		n := it.Count
		if n > sp.inv[i].Count {
			n = sp.inv[i].Count
		}
		// Захватываем статы ДО списания: takeN может удалить стак из среза.
		def := sp.inv[i].PetalDef
		sp.takeN(it.Kind, it.Rarity, n)
		total += petalStackXP(def) * float64(n)
	}
	return total
}
