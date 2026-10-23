package main

import "math/rand"

// mob_variant.go — геометрические подвиды мобов (порт родительского
// SecretProject, адаптация 2.0). Заменяют единственный «Golden»-флаг:
// золотой (gold), кольчатый (ring), звёздный (star) и серповидный (crescent)
// подвиды с собственными множителями HP/урона/размера и бустом редкости дропа.

// variantDefs — геометрические подвиды мобов. "" = обычный моб.
var variantDefs = map[string]struct {
	prefix      string
	color       uint32
	rarityBoost int     // +к редкости дропа (0-2)
	hpMult      float64 // множитель HP
	dmgMult     float64 // множитель DMG
	sizeMult    float64 // множитель радиуса
}{
	"":         {"", 0, 0, 1.0, 1.0, 1.0},
	"gold":     {"Golden ", 0xffd700, 2, 20.0, 1.0, 1.15},
	"ring":     {"Ring ", 0xf1c40f, 1, 1.5, 1.0, 1.2},
	"star":     {"Star ", 0xfffacd, 2, 1.0, 1.3, 1.25},
	"crescent": {"Crescent ", 0xb0c4de, 1, 0.7, 1.4, 0.9},
}

// variantSpawnWeights — веса для натурального спавна подвидов (сумма ≈ 1070).
var variantSpawnWeights = map[string]int{
	"":         100000,
	"gold":     100,
	"ring":     250,
	"star":     150,
	"crescent": 200,
}

// VariantDef возвращает параметры подвида (пустая строка = обычный моб).
func VariantDef(v string) (prefix string, color uint32, rarityBoost int, hpMult, dmgMult, sizeMult float64) {
	if d, ok := variantDefs[v]; ok {
		return d.prefix, d.color, d.rarityBoost, d.hpMult, d.dmgMult, d.sizeMult
	}
	d := variantDefs[""]
	return d.prefix, d.color, d.rarityBoost, d.hpMult, d.dmgMult, d.sizeMult
}

// IsVariant возвращает true, если у моба есть подвид (не обычный).
func IsVariant(v string) bool { return v != "" }

// PickRandomVariant выбирает подвид по весам ("" включительно).
func PickRandomVariant() string {
	total := 0
	for _, w := range variantSpawnWeights {
		total += w
	}
	n := rand.Intn(total)
	for v, w := range variantSpawnWeights {
		if n < w {
			return v
		}
		n -= w
	}
	return ""
}

// variantColor — цвет подвида (0 = обычный цвет вида).
func variantColor(v string) uint32 {
	if d, ok := variantDefs[v]; ok {
		return d.color
	}
	return 0
}

// variantPrefix — префикс имени подвида ("", "Golden ", "Ring ", ...).
func variantPrefix(v string) string {
	if d, ok := variantDefs[v]; ok {
		return d.prefix
	}
	return ""
}
