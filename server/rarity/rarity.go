// Package rarity — единый источник истины по редкостям: имена, цвета,
// кривые опасности/силы и шансы крафта. Из него генерируется client/js/rarities.js
// (см. server/cmd/genrarities). Это чистый рефакторинг: значения совпадают
// с прежними (server/world.go, server/mob_rarity.go, server/craft.go,
// client/js/ui_core.js, client/js/chat_tips_admin.js).
package rarity

import (
	"strconv"
	"strings"
)

// Names — 16 редкостей: 0..7 — зонная лестница Point..Chaos,
// 8..15 — «истинная» лестница Dot..Origin (параллельная, чуть сильнее парного
// тира: Dot между Point и Line, Vector между Line и Triangle, …,
// Origin выше Chaos). Индекс 4 — Pentagon.
var Names = []string{"Point", "Line", "Triangle", "Square", "Pentagon", "Hexagon", "Fractal", "Chaos",
	"Dot", "Vector", "Prism", "Frame", "Lattice", "Spiral", "Mirror", "Origin"}

// Colors — HEX-цвета редкостей (#rrggbb) в том же порядке. Истинные редкости —
// чуть темнее парных (решение пользователя).
var Colors = []string{"#ecf0f1", "#2ecc71", "#f1c40f", "#e74c3c", "#3498db", "#1abc9c", "#9b59b6", "#ff2d95",
	"#bcc8cb", "#23a55a", "#d0a308", "#bd3d2e", "#2c7cb8", "#169a81", "#7d4794", "#cd2378"}

// Danger — множитель опасности мобов (HP и урон). Истинные тиры 8..14 —
// геометрическая интерполяция между парными ступенями (×√1.5), Origin (15) —
// ×1.5 над Chaos. Кривая монотонна: каждое следующее значение больше.
var Danger = []float64{1.0, 1.5, 2.25, 3.375, 5.0625, 7.59375, 11.390625, 17.0859375,
	1.2247448713915892, 1.8371173070873836, 2.7556759606310758, 4.133513940946612, 6.200270911419919, 9.300406367049879, 13.950609550574818, 25.62890625}

// Power — кривая силы петаль (HP/DMG/эффекты) = клиентский RAR_MULT.
// Истинные тиры — интерполяция по тем же правилам.
var Power = []float64{1.0, 1.5, 2.25, 3.375, 6.0, 12.0, 27.0, 81.0,
	1.2247448713915892, 1.8371173070873836, 2.7556759606310758, 4.5, 8.485281374238571, 18.0, 46.765371804359685, 121.5}

// CraftChancePct — шанс успеха крафта (%) для перехода rarity -> rarity+1.
// Длина 15 = len(Names)-1. Истинные переходы: Chaos→Dot 4% (межлестничный),
// дальше — те же шансы, что в зонной лестнице.
var CraftChancePct = []float64{20.0, 12.5, 7.5, 4.0, 2.0, 1.0, 1.0, 4.0, 20.0, 12.5, 7.5, 4.0, 2.0, 1.0, 1.0}

// RenderJS возвращает содержимое client/js/rarities.js.
func RenderJS() string {
	var b strings.Builder
	b.WriteString("// AUTO-GENERATED from server/rarity (go generate). DO NOT EDIT.\n")
	b.WriteString("// Единый реестр редкостей клиента: имена, цвета, сила петаль, шансы крафта.\n\n")
	b.WriteString("const rarityNames = " + jsStrings(Names) + ";\n")
	b.WriteString("const rarityColors = " + jsStrings(Colors) + ";\n")
	b.WriteString("const RAR_MULT = " + jsFloats(Power) + ";\n")
	b.WriteString("const RARITY_DANGER = " + jsFloats(Danger) + ";\n")
	b.WriteString("const CRAFT_CHANCE_PCT = " + jsFloats(CraftChancePct) + ";\n\n")
	b.WriteString("// Имя редкости по индексу; '?' при выходе за диапазон.\n")
	b.WriteString("//\n// Локализация: английское имя — только фолбэк. Живая подпись берётся из\n")
	b.WriteString("// словаря клиента (ключ rarity.<индекс>, client/js/i18n.js), поэтому виды\n")
	b.WriteString("// редкости в хотбаре/крафте/галерее русские в русском UI. Тестовые данные\n")
	b.WriteString("// (const rarityNames выше) остаются основой сравнения — их не трогаем.\n")
	b.WriteString("function rarityName(r) {\n")
	b.WriteString("\tconst en = rarityNames[r];\n")
	b.WriteString("\tif (en == null) return '?';\n")
	b.WriteString("\treturn (typeof window !== 'undefined' && typeof window.tr === 'function')\n")
	b.WriteString("\t\t? window.tr('rarity.' + r, en) : en;\n")
	b.WriteString("}\n")
	return b.String()
}

func jsStrings(xs []string) string {
	parts := make([]string, len(xs))
	for i, s := range xs {
		parts[i] = "'" + s + "'"
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func jsFloats(xs []float64) string {
	parts := make([]string, len(xs))
	for i, v := range xs {
		parts[i] = strconv.FormatFloat(v, 'g', -1, 64)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}
