package main

import (
	"math"
	"math/rand"
	"strconv"
)

// mob.go - мобы и петали-предметы: типы, статы, редкости, спавн, ИИ.

// Rarity-подобные множители.
// Обе прогрессии СТРОГО геометрические:
//
//	dangerMult (HP и урон) — ×1.5 за тир, Chaos (7) = 1.5^7 ≈ 17.09x;
//	sizeMult (радиус) — нарастающая кривая (1, 1.6, 2.5, 3.5, 5, 7, 12, 20):
//	Chaos (7) в 20 раз больше базы (действует лимит limitMobSizeForZone:
//	радиус ≤ 18% ширины зоны). Chaos — самый большой и редкий тир;
//	aggroMult — множитель радиуса агрессии: чем выше редкость, тем дальше
//	моб ЗАМЕЧАЕТ игрока (Chaos видит в 6 раз дальше базового радиуса).
var rarityDanger = []float64{1.0, 1.5, 2.25, 3.375, 5.0625, 7.59375, 11.390625, 17.0859375}

// petalPower — кривая силы ПЕТАЛЕЙ (HP/DMG/эффекты), ОТДЕЛЬНО от кривой
// опасности мобов. Ребаланс (текущий): Polygon/Pentagon (r4) 6.0 и Hexagon
// (r5) 12.0 усилены — раньше Polygon (4.5) был «еле сильнее» Square (3.375),
// теперь заметно сильнее: Square(3.375) → Pentagon(6.0) = ×1.78,
// Pentagon → Hexagon(12.0) = ×2, Hexagon → Fractal(27.0) = ×2.25,
// Fractal → Chaos(81.0) = ×3. Высокие тиры петалей НАМНОГО сильнее:
// фулл фрактал-сет обязан заметно превосходить фулл полигон-сет.
var petalPower = []float64{1.0, 1.5, 2.25, 3.375, 6.0, 12.0, 27.0, 81.0}

// petalDmgMult — ОБЩИЙ множитель урона ВСЕХ петалей (применяется на попадании:
// ближний бой, клинки, осколки hiveling). Плоский ×2 вместо раздельных
// кривых/костылей: каждый удар петаля наносит ровно ×2 от его DMG-стата.
const petalDmgMult = 2.0

// petalVsMobMult — множитель урона ПЕТАЛЯ по мобу в зависимости от редкости.
// Progression-гейт: Common(0)/Line(1)/Triangle(2) петали в 10 раз слабее
// против мобов редкостью ВЫШЕ своей (Common vs Line+, Line vs Triangle+,
// Triangle vs Square+). Петали Square+ (3+) не ограничены (оставлено как есть).
func petalVsMobMult(petalR, mobR int) float64 {
	if petalR <= 2 && mobR > petalR {
		return 0.1
	}
	return 1.0
}
var raritySize = []float64{1.0, 1.6, 2.5, 3.5, 5.0, 7.0, 12.0, 20.0}
var rarityAggro = []float64{1.0, 1.3, 1.6, 2.0, 2.5, 3.2, 4.5, 6.0}

// mobHPDanger — кривая ЗАПАСА HP мобов (переменная, не равная rarityDanger).
// Урон растёт по rarityDanger (×1.5/тир), а HP — по более «толстой» кривой:
// Point..Hexagon получают заметно больше здоровья (Hexagon в 9.5 раз > Point,
// внутри вилки 5-10), чтобы петали Polygon+ не убивали высокие тиры подряд.
// Square (3) и Polygon (4) дополнительно ×1.4 (ребаланс): петали
// square-тира не должны мгновенно сносить Polygon-мобов, а Polygon-петали
// должны реально «держать свой тир».
// Нижние тиры почти не меняются (новичкам не сложнее). Fractal (6) и Chaos (7)
// остаются равными rarityDanger[6], rarityDanger[7] — формулы боссов
// (admin_test, баланс) обязаны совпадать.
var mobHPDanger = []float64{1.0, 1.7, 2.9, 6.44, 9.52, 9.5, 11.390625, 17.0859375}

type rarityWeights = []int

// Шансы редкости по ВЫСОТЕ Mob-зоны (одинаковы во всех зонах).
// Зона делится на 8 горизонтальных полос равной высоты: от Safe Zone (низ)
// до верха зоны. «Все тиры везде»: в каждой зоне по наклону могут встретиться
// все 8 редкостей, но чем выше полоса, тем больше шанс высоких тиров.
// В самой верхней полосе шанс Chaos 1/5000 (=0.02%, только там), Fractal —
// редкий (таймер-босс — главный источник), в нижней Chaos и Hexagon невозможны.
var rarityBands = []rarityWeights{
	{90, 10, 0, 0, 0, 0, 0, 0},               // полоса 0 (низ, у Safe Zone): почти только Point
	{70, 28, 2, 0, 0, 0, 0, 0},               // полоса 1
	{50, 37, 12, 1, 0, 0, 0, 0},              // полоса 2
	{32, 36, 25, 7, 0, 0, 0, 0},              // полоса 3
	{18, 28, 30, 21, 3, 0, 0, 0},             // полоса 4
	{10, 18, 26, 30, 12, 4, 0, 0},            // полоса 5
	{5, 10, 18, 26, 25, 13, 3, 0},            // полоса 6
	{100, 250, 450, 800, 1350, 1850, 199, 1}, // полоса 7 (верх): Chaos 1/5000, Fractal ~4%
}

// bandForY возвращает индекс полосы редкости по Y (0 = низ у Safe Zone).
func bandForY(z *Zone, y float64) int {
	bottom := mobZoneBottom(z)
	top := z.Y
	n := len(rarityBands)
	if bottom <= top {
		return 0
	}
	// y от top..bottom -> band от 0 (верх) до n-1 (у стены).
	t := (y - top) / (bottom - top)
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	b := int(t * float64(n))
	if b < 0 {
		b = 0
	}
	if b >= n {
		b = n - 1
	}
	// Больше y — ближе к Safe Zone — значит меньше редкость. Инверсия.
	return n - 1 - b
}

// ---- Редкость в зонах ВТОРОГО уровня (8-15): «большие прямоугольники» ----
// Второй уровень — жёсткая лестница сверху вниз: ЧЕМ НИЖЕ, тем меньше
// редкость. Зона делится на 4 горизонтальных прямоугольника равной высоты
// (по 1/4 зоны). Hexagon/Fractal/Chaos спавнятся ТОЛЬКО в самой верхней
// четверти, Polygon — в верхних 3/4, квадраты — везде.
// Верхняя полоса (итог 10000): Hexagon 1/200 (кап 2 на зону, зона-оповещение),
// Fractal 1/1500 (ролл, таймеров больше нет), Chaos 1/5000.
var l2RarityBands = []rarityWeights{
	{0, 0, 0, 100, 0, 0, 0, 0},    // 0 (низ, у входа): только Square
	{0, 0, 0, 750, 250, 0, 0, 0},  // 1: Square + Polygon
	{0, 0, 0, 200, 800, 0, 0, 0},  // 2: Polygon при остатке Square
	{0, 0, 0, 800, 6191, 0, 7, 2}, // 3 (верх, итог 10000): Hexagon НЕ роллится (таймер 5-15 мин), Fractal 1/1500, Chaos 1/5000
}

// Chaos II (15) — единственная вторая зона полного спектра (пол 0): та же
// лестница снизу вверх, но низ начинается с Point/Line, а не с Square.
var l2ChaosIIBands = []rarityWeights{
	{750, 250, 0, 0, 0, 0, 0, 0},   // низ: Point + Line
	{0, 550, 400, 50, 0, 0, 0, 0},  // 1: Line + Triangle
	{0, 0, 300, 600, 100, 0, 0, 0}, // 2: Triangle + Square + Polygon
	{0, 0, 0, 800, 6191, 0, 7, 2},  // верх (10000): Hexagon — только таймером, Fractal 1/1500, Chaos 1/5000
}

// pickL2RarityAtY выбирает редкость по высоте Y в зоне ВТОРОГО уровня.
// Игрок входит снизу (портал внизу зоны): чем выше поднялся — тем опаснее.
func pickL2RarityAtY(z *Zone, y float64) int {
	bottom := mobZoneBottom(z)
	top := z.Y
	n := len(l2RarityBands)
	if bottom <= top {
		return pickRarityFromWeights(l2RarityBands[0])
	}
	// t: 0 у самого верха, 1 у входа снизу (та же конвенция, что в bandForY).
	t := (y - top) / (bottom - top)
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	// Интенсивность высоты: 1 = верх (самая редкая полоса), 0 = низ.
	inten := 1 - t
	b := int(inten * float64(n))
	if b < 0 {
		b = 0
	}
	if b >= n {
		b = n - 1
	}
	band := l2RarityBands[b]
	// Chaos II (полный спектр) — единственная вторая зона с полом 0.
	if zoneRarityFloor[z.Index] == 0 {
		band = l2ChaosIIBands[b]
	}
	w := applyRarityClamp(band, zoneRarityFloor[z.Index], zoneRarityCeil[z.Index])
	return pickRarityFromWeights(w)
}

// mobTypes - базовые характеристики типов мобов (БЕЗ умножителей редкости).
// mobHPScale - общий усилитель запаса HP мобов.
const mobHPScale = 4.0

var mobTypes = map[string]MobType{
	// === Zone 0: Point — мир точек ===
	"dot": {
		Name: "Dot", Color: 0x2ecc71,
		BaseHP: 12, BaseDMG: 2, BaseSpeed: 45, BaseRadius: 18,
		AggroRange: 240, Shape: "circle",
	},
	"pebble": {
		Name: "Pebble", Color: 0x27ae60,
		BaseHP: 8, BaseDMG: 4, BaseSpeed: 75, BaseRadius: 14,
		AggroRange: 180, Shape: "pebble",
	},
	// === Zone 1: Line — мир линий ===
	"dart": {
		Name: "Dart", Color: 0xf1c40f,
		BaseHP: 8, BaseDMG: 6, BaseSpeed: 120, BaseRadius: 14,
		AggroRange: 300, Shape: "dart",
	},
	"spinner": {
		Name: "Spinner", Color: 0xf39c12,
		BaseHP: 18, BaseDMG: 4, BaseSpeed: 45, BaseRadius: 22,
		AggroRange: 200, Shape: "spinner",
	},
	// === Zone 2: Triangle — мир треугольников ===
	"triangle": {
		Name: "Triangle", Color: 0xe67e22,
		BaseHP: 14, BaseDMG: 7, BaseSpeed: 55, BaseRadius: 20,
		AggroRange: 280, Shape: "triangle",
	},
	"arrow": {
		Name: "Arrow", Color: 0xd35400,
		BaseHP: 7, BaseDMG: 12, BaseSpeed: 160, BaseRadius: 13,
		AggroRange: 400, Shape: "arrow",
	},
	// === Zone 3: Square — мир квадратов ===
	"square": {
		Name: "Square", Color: 0xe74c3c,
		BaseHP: 22, BaseDMG: 5, BaseSpeed: 40, BaseRadius: 24,
		AggroRange: 350, Shape: "square",
	},
	"block": {
		Name: "Block", Color: 0x8b0000,
		BaseHP: 80, BaseDMG: 8, BaseSpeed: 0, BaseRadius: 35,
		AggroRange: 0, Shape: "block",
	},
	// === Zone 4: Pentagon — мир пятиугольников ===
	"pentagon": {
		Name: "Pentagon", Color: 0x3498db,
		BaseHP: 28, BaseDMG: 7, BaseSpeed: 50, BaseRadius: 26,
		AggroRange: 350, Shape: "pentagon",
	},
	"shield": {
		Name: "Shield", Color: 0x2980b9,
		BaseHP: 40, BaseDMG: 3, BaseSpeed: 30, BaseRadius: 30,
		AggroRange: 200, Shape: "shield",
	},
	// === Zone 5: Hexagon — мир шестиугольников ===
	"hexagon": {
		Name: "Hexagon", Color: 0x1abc9c,
		BaseHP: 34, BaseDMG: 8, BaseSpeed: 50, BaseRadius: 28,
		AggroRange: 380, Shape: "hexagon",
	},
	"hive": {
		Name: "Hive", Color: 0x16a085,
		BaseHP: 60, BaseDMG: 4, BaseSpeed: 0, BaseRadius: 32,
		AggroRange: 300, Shape: "hive",
	},
	// === Zone 6: Fractal — мир фракталов ===
	"actal": {
		Name: "Actal", Color: 0x9b59b6,
		BaseHP: 28, BaseDMG: 10, BaseSpeed: 65, BaseRadius: 22,
		AggroRange: 400, Shape: "triangle",
	},
	"fragspark": {
		Name: "FragSpark", Color: 0xd6a4ff,
		BaseHP: 12, BaseDMG: 13, BaseSpeed: 130, BaseRadius: 14,
		AggroRange: 350, Shape: "spark",
	},
	"fragwall": {
		Name: "FragWall", Color: 0x6c3483,
		BaseHP: 55, BaseDMG: 5, BaseSpeed: 20, BaseRadius: 30,
		AggroRange: 200, Shape: "wall",
	},
	// === Zone 7: Chaos — мир хаоса ===
	"void": {
		Name: "Void", Color: 0x8e44ad,
		BaseHP: 42, BaseDMG: 13, BaseSpeed: 55, BaseRadius: 30,
		AggroRange: 450, Shape: "void",
	},
	"entropy": {
		Name: "Entropy", Color: 0xe74c3c,
		BaseHP: 32, BaseDMG: 16, BaseSpeed: 80, BaseRadius: 20,
		AggroRange: 380, Shape: "entropy",
	},
	// === Реальные мобы-«новобранцы» из витринных мокапов TestRoom ===
	// Shape "mockup_N" переиспользует авторские дизайны витрины: моб рисуется
	// клиентом через drawShowcaseDesign (см. drawMobShape).
	// Zone 0: Point
	"beads": {
		Name: "Beads", Color: 0x1dbf73,
		BaseHP: 8, BaseDMG: 2, BaseSpeed: 60, BaseRadius: 16,
		AggroRange: 200, Shape: "mockup_3",
	},
	"tear": {
		Name: "Tear", Color: 0x27ae60,
		// афк-моб (как Block/Hive): стоит на месте, распластался каплей.
		BaseHP: 14, BaseDMG: 2, BaseSpeed: 0, BaseRadius: 22,
		AggroRange: 220, Shape: "mockup_4",
	},
	// Zone 1: Line
	"comet": {
		Name: "Comet", Color: 0xf9ca24,
		BaseHP: 12, BaseDMG: 8, BaseSpeed: 105, BaseRadius: 16,
		AggroRange: 320, Shape: "mockup_14",
	},
	"beam": {
		Name: "Beam", Color: 0xffd32e,
		BaseHP: 10, BaseDMG: 11, BaseSpeed: 95, BaseRadius: 15,
		AggroRange: 350, Shape: "mockup_15",
	},
	// Zone 2: Triangle
	"spike": {
		Name: "Spike", Color: 0xf6b93b,
		// Цвет адаптирован под Triangle-зону (оранжевая тема зоны).
		BaseHP: 12, BaseDMG: 9, BaseSpeed: 70, BaseRadius: 18,
		AggroRange: 300, Shape: "mockup_1",
	},
	// Zone 3: Square
	"cross": {
		Name: "Cross", Color: 0xc0392b,
		// Цвет адаптирован под Square-зону: тёмно-красный (как просил юзер).
		BaseHP: 25, BaseDMG: 5, BaseSpeed: 45, BaseRadius: 26,
		AggroRange: 340, Shape: "mockup_5",
	},
	"prism": {
		Name: "Prism", Color: 0xe74c3c,
		// Редкий НЕЙТРАЛЬНЫЙ моб Square-зоны (Peaceful): не агрится на игрока,
		// пока игрок сам не ударит его (тогда честно догоняет обидчика).
		// Дропает петаль Rhombus — замедление в пределах редкости петеля.
		BaseHP: 32, BaseDMG: 8, BaseSpeed: 42, BaseRadius: 26,
		AggroRange: 360, Shape: "mockup_16",
		Peaceful: true,
	},
	// Zone 4: Pentagon
	"star": {
		Name: "Star", Color: 0x0abab5,
		BaseHP: 24, BaseDMG: 7, BaseSpeed: 60, BaseRadius: 24,
		AggroRange: 360, Shape: "mockup_2",
	},
	// Zone 5: Hexagon
	"aeon": {
		Name: "Aeon", Color: 0xb07cd6,
		BaseHP: 36, BaseDMG: 8, BaseSpeed: 45, BaseRadius: 26,
		AggroRange: 380, Shape: "mockup_11",
	},
	"rune": {
		Name: "Rune", Color: 0x9b59b6,
		BaseHP: 32, BaseDMG: 9, BaseSpeed: 55, BaseRadius: 24,
		AggroRange: 380, Shape: "mockup_12",
	},
	"cog": {
		Name: "Cog", Color: 0x16a085,
		BaseHP: 40, BaseDMG: 7, BaseSpeed: 40, BaseRadius: 28,
		AggroRange: 350, Shape: "mockup_13",
	},
	// Zone 6: Fractal
	"sun": {
		Name: "Sun", Color: 0x85c1e9,
		BaseHP: 30, BaseDMG: 10, BaseSpeed: 55, BaseRadius: 24,
		AggroRange: 400, Shape: "mockup_7",
	},
	"mote": {
		Name: "Mote", Color: 0x8e44ad,
		BaseHP: 20, BaseDMG: 12, BaseSpeed: 92, BaseRadius: 18,
		AggroRange: 380, Shape: "mockup_10",
	},
	// Zone 7: Chaos
	"saturn": {
		Name: "Saturn", Color: 0x73c6b6,
		BaseHP: 45, BaseDMG: 14, BaseSpeed: 45, BaseRadius: 32,
		AggroRange: 420, Shape: "mockup_6",
	},
	"eye": {
		Name: "Eye", Color: 0xe472a0,
		BaseHP: 38, BaseDMG: 15, BaseSpeed: 60, BaseRadius: 26,
		AggroRange: 450, Shape: "mockup_8",
	},
	"galaxy": {
		Name: "Galaxy", Color: 0x9b59b6,
		BaseHP: 42, BaseDMG: 14, BaseSpeed: 55, BaseRadius: 30,
		AggroRange: 430, Shape: "mockup_9",
	},
	// Black Hole — редкий моб Chaos-зоны. Уникальный дизайн "blackhole" на
	// клиенте: аккреционный диск, фотонное кольцо, горизонт событий.
	// Базовый DMG 8 × mobDmgMult(2.0) = 16 на Point-редкости.
	"blackhole": {
		Name: "Black Hole", Color: 0x5b2c6f,
		BaseHP: 48, BaseDMG: 8, BaseSpeed: 50, BaseRadius: 34,
		AggroRange: 450, Shape: "blackhole",
	},
	// === TestRoom: 16 витринных мокапов (не спавнятся обычным спавнером) ===
	// Shape "mockup_N" рисуется клиентом в drawShowcaseDesign.
	"mockup1":  {Name: "Spike", Color: 0x48c9b0, BaseHP: 1, BaseDMG: 0, BaseSpeed: 0, BaseRadius: 42, AggroRange: 0, Shape: "mockup_1"},
	"mockup2":  {Name: "Star", Color: 0x0abab5, BaseHP: 1, BaseDMG: 0, BaseSpeed: 0, BaseRadius: 42, AggroRange: 0, Shape: "mockup_2"},
	"mockup3":  {Name: "Beads", Color: 0x1dbf73, BaseHP: 1, BaseDMG: 0, BaseSpeed: 0, BaseRadius: 42, AggroRange: 0, Shape: "mockup_3"},
	"mockup4":  {Name: "Tear", Color: 0x27ae60, BaseHP: 1, BaseDMG: 0, BaseSpeed: 0, BaseRadius: 42, AggroRange: 0, Shape: "mockup_4"},
	"mockup5":  {Name: "Cross", Color: 0x52be80, BaseHP: 1, BaseDMG: 0, BaseSpeed: 0, BaseRadius: 42, AggroRange: 0, Shape: "mockup_5"},
	"mockup6":  {Name: "Saturn", Color: 0x73c6b6, BaseHP: 1, BaseDMG: 0, BaseSpeed: 0, BaseRadius: 42, AggroRange: 0, Shape: "mockup_6"},
	"mockup7":  {Name: "Sun", Color: 0x85c1e9, BaseHP: 1, BaseDMG: 0, BaseSpeed: 0, BaseRadius: 42, AggroRange: 0, Shape: "mockup_7"},
	"mockup8":  {Name: "Eye", Color: 0xe472a0, BaseHP: 1, BaseDMG: 0, BaseSpeed: 0, BaseRadius: 42, AggroRange: 0, Shape: "mockup_8"},
	"mockup9":  {Name: "Galaxy", Color: 0x9b59b6, BaseHP: 1, BaseDMG: 0, BaseSpeed: 0, BaseRadius: 42, AggroRange: 0, Shape: "mockup_9"},
	"mockup10": {Name: "Mote", Color: 0x8e44ad, BaseHP: 1, BaseDMG: 0, BaseSpeed: 0, BaseRadius: 42, AggroRange: 0, Shape: "mockup_10"},
	"mockup11": {Name: "Aeon", Color: 0xb07cd6, BaseHP: 1, BaseDMG: 0, BaseSpeed: 0, BaseRadius: 42, AggroRange: 0, Shape: "mockup_11"},
	"mockup12": {Name: "Rune", Color: 0x9b59b6, BaseHP: 1, BaseDMG: 0, BaseSpeed: 0, BaseRadius: 42, AggroRange: 0, Shape: "mockup_12"},
	"mockup13": {Name: "Cog", Color: 0x16a085, BaseHP: 1, BaseDMG: 0, BaseSpeed: 0, BaseRadius: 42, AggroRange: 0, Shape: "mockup_13"},
	"mockup14": {Name: "Comet", Color: 0xf9ca24, BaseHP: 1, BaseDMG: 0, BaseSpeed: 0, BaseRadius: 42, AggroRange: 0, Shape: "mockup_14"},
	"mockup15": {Name: "Beam", Color: 0xffd32e, BaseHP: 1, BaseDMG: 0, BaseSpeed: 0, BaseRadius: 42, AggroRange: 0, Shape: "mockup_15"},
	// Prism (16): красный кристалл-призма из дважды ромба. Отдельный стенд в
	// TestRoom ПОД НОМЕРОМ 16; слоты 17-25 пока свободны (новые мобы появятся).
	"mockup16": {Name: "Prism", Color: 0xe74c3c, BaseHP: 1, BaseDMG: 0, BaseSpeed: 0, BaseRadius: 42, AggroRange: 0, Shape: "mockup_16"},
}

// mobTypesInZone - из каких типов выбирается рандомный моб в зоне.
var mobTypesInZone = [][]string{
	{"dot", "pebble", "beads", "tear"},                        // 0  Point
	{"dart", "spinner", "comet", "beam"},                      // 1  Line
	{"triangle", "arrow", "spike"},                            // 2  Triangle
	{"square", "block", "cross", "prism"},                     // 3  Square (Prism — редкий нейтральный)
	{"pentagon", "shield", "star"},                            // 4  Pentagon
	{"hexagon", "aeon", "rune", "cog", "hive"},                // 5  Hexagon (Hive — редкий)
	{"actal", "fragspark", "fragwall", "sun", "mote"},         // 6  Fractal
	{"void", "entropy", "hexagon", "saturn", "eye", "galaxy", "blackhole"}, // 7  Chaos
	// Второй уровень (те же типы, что у родительской зоны).
	{"dot", "pebble", "beads", "tear"},                        // 8  Point II
	{"dart", "spinner", "comet", "beam"},                      // 9  Line II
	{"triangle", "arrow", "spike"},                            // 10 Triangle II
	{"square", "block", "cross", "prism"},                     // 11 Square II (Prism — редкий нейтральный)
	{"pentagon", "shield", "star"},                            // 12 Pentagon II
	{"hexagon", "aeon", "rune", "cog", "hive"},                // 13 Hexagon II (Hive — редкий)
	{"actal", "fragspark", "fragwall", "sun", "mote"},         // 14 Fractal II
	{"void", "entropy", "hexagon", "saturn", "eye", "galaxy", "blackhole"}, // 15 Chaos II
	nil, // 16 TestRoom — обычные мобы не спавнятся
}

// zoneSpawnWeights — веса типов из mobTypesInZone для ВЗВЕШЕННОГО выбора
// (индекс i в массиве соответствует i-му типу зоны). Более «профильные»
// мобы зоны спавнятся чаще; НОВЫЕ мобы спавнятся Заметно реже классических:
// beads — редкий в Point, spike — намного реже остальных в Triangle,
// eye — исключительно редкий в Chaos (шанс 1/100, было 1/24).
// Star (Pentagon 4/12): вес 2→1 при пропорции pentagon:shield 5:4→7:6 —
// шанс упал с 2/11 ≈ 18% до 1/14 ≈ 7% (в ~2.5 раза).
// Если весов нет (или длина не совпадает) — randMobType падает на равномерный выбор.
var zoneSpawnWeights = map[int][]int{
	0:  {5, 4, 1, 2},
	1:  {5, 4, 2, 2},
	2:  {5, 4, 1},
	3:  {5, 2, 2, 1},
	4:  {7, 6, 1},
	5:  {4, 2, 2, 2, 1},
	6:  {4, 4, 3, 2, 2},
	7:  {33, 33, 12, 10, 1, 11, 1},
	8:  {5, 4, 1, 2},
	9:  {5, 4, 2, 2},
	10: {5, 4, 1},
	11: {5, 2, 2, 1},
	12: {7, 6, 1},
	13: {4, 2, 2, 2, 1},
	14: {4, 4, 3, 2, 2},
	15: {33, 33, 12, 10, 1, 11, 1},
}

// Лимит мобов на зону (карта стала больше — плотность спавна увеличена ~x2).
// Zone 4 (Pentagon) — самая густонаселённая: там живут группы Shield+Pentagon.
// Zone 16 (TestRoom) — 0 обычных мобов: там живут только витринные мокапы.
// Ребаланс: зоны ПЕРВОГО уровня (0-7) не тронуты, ВТОРОЙ уровень (8-15)
// заселён в 2 раза плотнее (Pentagon II (12) — только ×1.5).
var zoneMobLimits = []int{150, 160, 128, 126, 240, 160, 180, 120, 300, 320, 256, 252, 360, 320, 360, 240, 0}

// Кап редкости мобов по зонам:
//   - зоны 0-7 (первый уровень, включая Chaos): мобы максимум Square —
//     безопасная прокачка без риска встретить высокие редкости;
//   - зоны 8-14 (второй уровень): от Square до Chaos — за порталом;
//   - зона 15 (Chaos II): полный спектр — финальный хардкорный уровень.
var zoneRarityFloor = []int{0, 0, 0, 0, 0, 0, 0, 0, 3, 3, 3, 3, 3, 3, 3, 0, 0}
var zoneRarityCeil = []int{3, 3, 3, 3, 3, 3, 3, 3, 7, 7, 7, 7, 7, 7, 7, 7, 0}

// ---- Виды петалей ----

// PetalKind описывает "вид" петеля (роль): у каждого свой базовый стат-профиль,
// форма для отрисовки. Комбинация (Kind, Rarity) определяет предмет петеля.
type PetalKind struct {
	Name      string
	Shape     string
	BaseHP    float64
	BaseDMG   float64
	Poison    bool // яд: удары петеля вешают на моба DoT (см. applyPoison)
	Thorns    bool // шипы: износ петеля отражается мобу (см. fightPetalsVsMobs)
	Regen     bool // реген: пассивно лечит игрока % MaxHP/сек, пока петаль активна (orb)
	MaxHP     bool // MaxHP: пассивно добавляет к макс. HP игрока по редкости (bulwark)
	Magnet    bool // магнит: притягивает петали-дропы с пола к игроку (hex)
	Split     bool // разрыв: при уничтожении выпускает осколки в врагов (hiveling)
	Missile   bool // миссил: по пробелу вылетает по касательной к орбите (blade)
	Push      bool // толчок: усиленный гравитационный отброс моба от удара (cube)
	Chain     bool // цепь: цепной урон ближайшим врагам с визуальной линией (spark)
	Dash      bool // рывок: по пробелу тратит петаль и делает рывок по движению (void)
	Lifesteal bool // лайфстил: при ударе лечит игрока % от нанесённого урона (oval)
	Armor     bool // броня: износ от ударов мобов снижен (rock)
	Bomb      bool // бомба: мини-АОЕ вокруг точки удара (ball)
	Sling     bool // праща: сильный толчок и почти нет кулдауна удара (pebble)
	Wound     bool // рана: короткий DoT на цели при ударе (shard)
	Spin      bool // ускоритель: ускоряет вращение орбиты по редкости, стакается (cog)
	Swift     bool // ускорение: повышает скорость передвижения игрока, стакается (penta)
	Twin      bool // самоподобие: шанс двойного удара (fractalite)
	Guard     bool // барьер: пассивно снижает входящий урон игрока, стакается (fragwall)
	Shield    bool // щит: пополняет полоску щита игрока (броню перед HP) (shield)

	Tear      bool // слеза: при РАЗРУШЕНИИ петеля лечит игрока % MaxHP по редкости (tear)
	Eye       bool // глаз: увеличивает радиус орбиты всех петалей по редкости (eye)
	View      bool // линия: геометрически расширяет радиус обзора игрока (line)
	Compass   bool // компас: сервер шлёт угол на ближайшего Fractal/Chaos-моба (compass)
	Saturn    bool // кольца: урон-кольцо вокруг игрока, враги в кольце тикаются (saturn)
	Rune      bool // руна: 3 стака на мобе → взрыв AoE вокруг него (rune)
	Overdrive bool // заводка: укорачивает кулдаун удара всех петалей, стакается (cog)
	Slow      bool // замедление: удар тормозит моба в пределах своей редкости (rhombus)
	Twirl     bool // разворот: каждый Twirl на орбите переворачивает направление её вращения (twirl)

	MinionType string // тип моба-миньона ("entropy" / "blackhole"); если "" — не спавнит миньонов
}

// Каталог видов петалей. Статы далее умножаются на редкость (rarityDanger).
// Базовые «стартовые» виды (oval/thorns/rock/ball) лежат в земле и на них же
// можно учиться крафтить; тематические (orb..entropy) дропаются с конкретных
// мобов (см. mobDropKind).
var petalKinds = map[string]PetalKind{
	// Стартовые виды
	"oval": {
		Name: "Oval", Shape: "oval",
		// Лайфстил (механика «лепесток»): при ударе лечит игрока lifestealPct
		// от нанесённого урона — састийн в бою.
		BaseHP: 8.0, BaseDMG: 4.0,
		Lifesteal: true,
	},
	"thorns": {
		Name: "Thorns", Shape: "thorns",
		// Шипы («размен»): прямой урон высокий, HP низкое. За каждый износ
		// (моб бьёт шип) моб получает обратно thornsReflectRatio × урон износа.
		// Чем больнее бьёт моб — тем больше «сам колется» о шип.
		BaseHP: 5.0, BaseDMG: 7.0,
		Thorns: true,
	},
	"rock": {
		Name: "Rock", Shape: "rock",
		// Броня (механика «камень»): износ от ударов мобов снижен (armorWearMult),
		// камень живучий.
		BaseHP: 14.0, BaseDMG: 2.5,
		Armor: true,
	},
	"ball": {
		Name: "Ball", Shape: "circle",
		// Бомба (механика «шарик»): при ударе мини-АОЕ вокруг точки удара
		// (bombRadius, урон bombDmgFrac).
		BaseHP: 6.0, BaseDMG: 3.0,
		Bomb: true,
	},
	// Тематические виды (дроп с мобов)
	"orb": {
		Name: "Orb", Shape: "circle",
		// Реген (механика «сфера»): пока петаль на орбите, лечит игрока
		// orbRegenFrac × rarityDanger от макс. HP в секунду. НЕ стакается:
		// работает только ОДИН реген — максимальной редкости среди живых.
		BaseHP: 7.0, BaseDMG: 4.0,
		Regen: true,
	},
	"pebble": {
		Name: "Pebble", Shape: "rock",
		// Праща (механика «галька»): сильный толчок моба и почти без кулдауна
		// удара (pebbleHitCD) — быстрый контроль. «Желе»: микро-пуш и немного HP.
		BaseHP: 9.0, BaseDMG: 3.0,
		Sling: true,
	},
	"shard": {
		Name: "Shard", Shape: "thorns",
		// Рана (механика «осколок»): при ударе оставляет короткий DoT (woundDuration)
		// на цели — «зазубрина», добивает после отхода.
		BaseHP: 5.5, BaseDMG: 9.0,
		Wound: true,
	},
	"cog": {
		Name: "Cog", Shape: "oval",
		// Заводка (механика «шестерёнка»): пока петаль на орбите, укорачивает
		// кулдаун удара ВСЕХ петалей (атаки чаще), величина зависит от редкости,
		// стакается (кап overdriveCapMult). Ускорение вращения орбиты переехало
		// к whirl. Дроп с Cog-моба.
		BaseHP: 10.0, BaseDMG: 4.5,
		Overdrive: true,
	},
	"whirl": {
		Name: "Whirl", Shape: "whirl",
		// Вихрь (механика «спиннер»): маленький круглый спиннер — носитель
		// ускорителя орбиты. Дроп со Spinner-моба: способности прежней
		// «шестерёнки» (Spin) на новом круглом теле.
		BaseHP: 10.0, BaseDMG: 4.5,
		Spin: true,
	},
	"blade": {
		Name: "Blade", Shape: "thorns",
		// Миссил (механика «клинок»): по пробелу вылетает по касательной к
		// орбите (в направлении, куда он направлен в момент нажатия), бьёт
		// мобов на пути и уходит на кулдаун (bladeCooldown).
		BaseHP: 5.0, BaseDMG: 10.0,
		Missile: true,
	},
	"viper": {
		Name: "Viper", Shape: "thorns",
		// Яд (механика «гадюка»): HP нет (1 = «растворяется» при первом же
		// касании), зато каждый удар вешает на цель DoT-яд
		// (poisonDpsMult × DMG, poisonDuration с). Пул яда ЕДИНЫЙ и НЕ
		// стакается: повторные удары освежают таймер и держат максимальную
		// силу (take-max), но НЕ складывают DPS от нескольких гадюк.
		// Роль — жало-носитель яда: прямой урон невелик, весь ДПС — в яде
		// (того же тира держит уровень Spark). Гейтинг по редкости: на моба
		// ВЫШЕ тира петаля яд ослаблен ×poisonCrossTierMult — square-Viper
		// больше не «растворяет» Polygon-мобов.
		BaseHP: 1.0, BaseDMG: 8.0,
		Poison: true,
	},
	"cube": {
		Name: "Cube", Shape: "rock",
		// Гравитационный толчок (механика «грузовой танк»): куб тяжёлый и при
		// ударе сильно отталкивает моба назад; урон минимальный — пуш-петаль.
		BaseHP: 15.0, BaseDMG: 2.0,
		Push: true,
	},
	"bulwark": {
		Name: "Bulwark", Shape: "rock",
		// MaxHP (механика «бастион»): пока петаль на орбите, добавляет к макс.
		// HP игрока bulwarkMaxHPFrac × rarityDanger от базового MaxHP.
		// Урон снижен — это танк-стат, а не дамагер.
		BaseHP: 26.0, BaseDMG: 2.0,
		MaxHP: true,
	},
	"penta": {
		Name: "Penta", Shape: "pentagon",
		// Ускорение (механика «пентаграмма»): повышает скорость передвижения
		// игрока на (swiftBasePct + swiftPerRarity×редкость); стакается.
		BaseHP: 9.0, BaseDMG: 6.5,
		Swift: true,
	},
	"shield": {
		Name: "Shield", Shape: "shield",
		// Щит (механика «барьер»): автоматически пополняет синюю полоску щита
		// игрока (броня перед HP). Каждый активный щит даёт ёмкость MaxShield и
		// периодически (shieldChargeInterval) заряжает её.
		BaseHP: 13.0, BaseDMG: 6.0,
		Shield: true,
	},
	"hex": {
		Name: "Hex", Shape: "hexagon",
		// Магнит (механика «гекс»): притягивает все петали-дропы с пола к
		// игроку в радиусе, растёт с редкостью (hexMagnet*).
		BaseHP: 10.0, BaseDMG: 7.0,
		Magnet: true,
	},
	"hiveling": {
		Name: "Hiveling", Shape: "circle",
		// Разрыв (механика «рой»): HP нет (1 = лопается от первого касания),
		// при уничтожении выпускает ОСКОЛКИ в ближайших врагов — это «условный
		// миссайл»: весь смысл в осколках, не в теле.
		BaseHP: 1.0, BaseDMG: 9.0,
		Split: true,
	},
	"fractalite": {
		Name: "Fractalite", Shape: "thorns",
		// Самоподобие (механика «фрактал»): при ударе с шансом twinChance бьёт
		// дважды. HP нет (1) — «как чип во флорке»: шанс/рнг и высокий урон.
		BaseHP: 1.0, BaseDMG: 15.0,
		Twin: true,
	},
	"spark": {
		Name: "Spark", Shape: "spark",
		// Цепь (механика «молния»): при ударе бьёт ближайших врагов цепью
		// (sparkChainLinkCount звеньев, каждый следующий ×(1-sparkChainFalloff)
		// от предыдущего). Визуальная линия-цепь (см. sparkChain).
		BaseHP: 4.67, BaseDMG: 8.0, // дебаф ÷1.5
		Chain: true,
	},
	"fragwall": {
		Name: "Fragwall", Shape: "rock",
		// Барьер-щит (механика «стена»): пассивно снижает входящий урон игрока
		// на (guardBasePct + guardPerRarity×редкость) за петаль, стакается
		// (но не ниже 20% от полного урона). Задача — ДЕФ, урон минимальный.
		BaseHP: 19.0, BaseDMG: 1.0,
		Guard: true,
	},
	"void": {
		Name: "Void", Shape: "circle",
		// Рывок (механика «бездна»): по пробелу тратит петаль (уходит в перезарядку)
		// и делает рывок на voidDashDist по направлению движения (см. dashVoid).
		BaseHP: 11.0, BaseDMG: 12.0,
		Dash: true,
	},
	"entropy": {
		Name: "Entropy", Shape: "spark",
		// Спавн-петаль: НЕ наносит урона сам (только HP — «выживает» для
		// призыва миньонов). Весь урон дают миньоны.
		// Редкость/сила призыва — по таблице summonSpec (тот же тир моба, что
		// и редкость петеля; Pentagon+ — Square ×2 / Polygon с MK-именами);
		// HP и DMG миньона — ТОЧНЫЕ статы враждебного моба того же типа и
		// редкости (mobStatsForRarity), умноженные на множитель усиления.
		BaseHP: 8.0, BaseDMG: 0.0,
		MinionType: "entropy",
	},
	// === Новые виды из витринных мокапов (дроп с новых мобов) ===
	"singularity": {
		Name: "Singularity", Shape: "circle",
		// Спавн-петаль (как Entropy): сам не бьёт (DMG 0), призывает миньонов
		// Black Hole по той же таблице summonSpec, что и Entropy (редкость моба
		// = редкости петеля для Point..Square, Pentagon+ — усиленные с MK-
		// именами). HP и DMG миньона — ТОЧНЫЕ статы враждебного моба того же
		// типа и редкости (mobStatsForRarity), умноженные на множитель.
		BaseHP: 10.0, BaseDMG: 0.0,
		MinionType: "blackhole",
	},
	"tear": {
		Name: "Tear", Shape: "tear",
		// Слеза (механика «слёзы»): когда моб ЛОМАЕТ петаль, она лечит игрока
		// tearHealBasePct × (1 + 0.5×редкость) от MaxHP. Стакается.
		BaseHP: 9.0, BaseDMG: 3.0,
		Tear: true,
	},
	"line": {
		Name: "Line", Shape: "line",
		// Линия (механика «антенна»): аксессуар (макс. 1, узор реже с ростом
		// редкости). Визуально «антенна» на игроке; радиус обзора клиент
		// считает геометрически: 1500 × 1.1^редкость (потолок lineViewCap).
		// Сервер спец-механик не имеет (viewW/viewH шлёт клиент).
		BaseHP: 10.0, BaseDMG: 0.0,
		View: true,
	},
	"compass": {
		Name: "Compass", Shape: "circle",
		// Компас (механика «компас»): сервер раз в 1 с шлёт игроку угол на
		// ближайшего живого моба с Rarity ≥ Fractal в ЕГО зоне (работает только
		// на втором уровне 8-15). Стрелка на кружке петеля рисуется клиентом.
		BaseHP: 14.0, BaseDMG: 1.0,
		Compass: true,
	},
	"saturn": {
		Name: "Saturn", Shape: "saturn",
		// Кольца (механика «сатурн», макс. 1): кольцо-аура вокруг игрока
		// (радиус saturnRingBase + saturnRingPerRar×редкость), враги в кольце
		// получают тиковый урон saturnDpsBase×petalPower и лёгкий отброс.
		// Бафф: больше HP петали (16→22) и сильнее прямой удар (5→7).
		BaseHP: 22.0, BaseDMG: 7.0,
		Saturn: true,
	},
	"rune": {
		Name: "Rune", Shape: "rune",
		// Руна (механика «руна»): удары вешают руну на моба (макс. runeStackNeed,
		// истекает через runeStackDur). Набрав 3 — взрыв AoE runeBoomMult×DMG
		// вокруг моба (runeBoomRadius). Счётчик рун — на сервере на мобе.
		BaseHP: 13.0, BaseDMG: 7.0,
		Rune: true,
	},
	"eye": {
		Name: "Eye", Shape: "eye",
		// Глаз (механика «глаз», макс. 1): аксессуар. Увеличивает радиус орбиты
		// всех петалей на eyeOrbitPerRar×редкость (r7 ≈ ×1.7), петали разлетаются
		// шире ПЛАВНО (ramp в updatePlayer). Визуально «глаз» в центре игрока.
		BaseHP: 14.0, BaseDMG: 0.0,
		Eye: true,
	},
	"spike": {
		Name: "Spike", Shape: "spike",
		// Шип (механика «шип»): аксессуар — обводка из шипов вокруг игрока,
		// орбиту НЕ занимает, КУЛДАУНА ИМЕЕТ, пассив. Прибавляет УРОН К ТЕЛУ
		// игрока: каждый надетый spike ДОБАВЛЯЕТ spikeBodyBase×rarityDanger
		// к контактному урону тела (складывается, см. bodySpikeDmg).
		BaseHP: 10.0, BaseDMG: 0.0,
	},
	"rhombus": {
		Name: "Rhombus", Shape: "rhombus",
		// Ромб (механика «замедление»): каждый удар замедляет моба на
		// rhombusSlowPct на rhombusDur (макс. SlowT = rhombusMaxT). Работает
		// ТОЛЬКО если редкость моба <= редкости петеля (см. fightPetalsVsMobs).
		// Дроп с редкого нейтрального Prism (Square-зона).
		BaseHP: 14.0, BaseDMG: 6.0,
		Slow: true,
	},
	"twirl": {
		Name: "Twirl", Shape: "twirl",
		// Вертушка (механика «разворот»): вытянутый ромб-двусторонний дротик.
		// ПОКА ПЕТАЛЬ НА ОРБИТЕ — направление вращения всей орбиты мгновенно
		// разворачивается (было по часовой → стало против, и наоборот). Каждый
		// Twirl переворачивает направление (парность): один — в обратную,
		// два — снова по часовой. Дроп с Rune-моба (редкая подмена).
		BaseHP: 10.0, BaseDMG: 6.0,
		Twirl: true,
	},
}

// Виды, из которых случайно выбирается петаль на земле.
var petalKindsPool = []string{"oval", "thorns", "rock", "ball"}

// Спавн петалей-предметов на земле: сколько их и в каких зонах.
var zonePetalCount = []int{5, 5, 5, 5, 5, 4, 4, 3}

// Каждый тип моба при смерти "дропает" конкретный вид петеля.
var mobDropKind = map[string]string{
	"dot":       "orb",
	"pebble":    "pebble",
	"dart":      "shard",
	"spinner":   "whirl",
	"triangle":  "blade",
	"arrow":     "viper",
	"square":    "cube",
	"block":     "bulwark",
	"pentagon":  "penta",
	"shield":    "shield",
	"hexagon":   "hex",
	"hive":      "hiveling",
	"actal":     "fractalite",
	"fragspark": "spark",
	"fragwall":  "fragwall",
	"void":      "void",
	"entropy":   "entropy",
	// Витринные мокапы → реальные мобы
	"beads":  "orb",
	"tear":   "tear",
	"comet":  "blade",
	"beam":   "spark",
	"spike":  "spike",
	"cross":  "bulwark",
	"star":   "spark",
	"aeon":   "hex",
	"rune":   "rune",
	"cog":    "cog",
	"sun":    "compass",
	"mote":   "fractalite",
	"saturn": "saturn",
	"eye":    "void",
	"galaxy": "void",
	"blackhole": "singularity",
	"prism":  "rhombus", // Prism дропает петаль Rhombus (замедление)
}

// mobRareDrop — РЕДКИЙ дроп: с малым шансом вместо обычного вида (mobDropKind)
// у ДАННОГО игрока выпадает более ценный вид. Роллится per-player в
// makeDropPetal, потому оба убийцы топ-4 могут получить один и тот же редкий.
var mobRareDrop = map[string]RareDropDef{
	"beads": {Kind: "line", Chance: 0.10}, // Линия: 10% вместо обычного orb
	"eye":   {Kind: "eye", Chance: 0.01},  // Глаз: 1% вместо обычного void
	"rune":  {Kind: "twirl", Chance: 0.05}, // Вертушка: 5% вместо обычного rune
}

// RareDropDef описывает редкий дроп типа моба.
type RareDropDef struct {
	Kind   string
	Chance float64 // 0..1 — вероятность ПОДМЕНЫ обычного вида
}

// blackHoleDropChance — шанс дропа Singularity с Black Hole (per-mob roll,
// один бросок на убийство, затем все топ-4 дамагеры получают петаль).
const blackHoleDropChance = 0.2

// Дроп-таблица редкости петеля после убийства моба.
// Запись [редкость моба] = [вес Point..Chaos] (8 редкостей, сумма 1000 = 100%).
// Кривая «колокол с центром на -1 тир»: большинство дропа — на 1 тир ниже моба.
// Правило «максимум — редкость самого моба»: Point-моб кидает только Point,
// всё до Fractal включительно — максимум свой тир, Chaos-моб — максимум Fractal
// (столбец Chaos везде 0).
// Явные качели по редкостям (настройка пользователя, «чтобы шестиугольник был вкусным»):
//   - Hexagon-моб: собственный Hexagon УЛЬТРА-редкий (0.4%), центр — Square (72.6%),
//     Polygon — 15%;
//   - Fractal-моб: Hexagon РЕДКИЙ (10%), собственный Fractal — 0.2%, центр —
//     Polygon (80%);
//   - Chaos-моб: НИЧЕГО ниже Pentagon (реркость 4+), Hexagon — основной (73%),
//     Fractal — 2%.
var mobRarityDrop = [][]int{
	{1000, 0, 0, 0, 0, 0, 0, 0},     // Point моб: только свой тир (100%)
	{780, 220, 0, 0, 0, 0, 0, 0},    // Line моб: 78 Point / 22 Line
	{90, 580, 330, 0, 0, 0, 0, 0},   // Triangle моб: 9/58/33
	{20, 150, 550, 280, 0, 0, 0, 0}, // Square моб: 2/15/55/28
	// Polygon моб (ребаланс): свой Polygon дроп порезан до ~1.2%;
	// освободившаяся масса ушла в Square/Triangle/Line.
	{0, 16, 148, 824, 12, 0, 0, 0},
	// Hexagon моб (ребаланс): центр Square (~80%), Polygon ~16%, СВОЙ
	// Hexagon — 1.67%.
	{0, 0, 13, 800, 160, 17, 0, 0},
	// Fractal моб (ребаланс): Hexagon = 16.7% (1/6), Fractal 0.2%, центр Polygon 74%.
	{0, 0, 8, 83, 740, 167, 2, 0},
	{0, 0, 0, 0, 250, 730, 20, 0}, // Chaos моб: 25 Polygon/73 Hexagon/2% Fractal, никогда Chaos
}

// ---- Моб ----

type Mob struct {
	ID     string
	Type   string
	Rarity int
	X, Y   float64
	HP     float64
	MaxHP  float64
	DMG    float64
	Speed  float64
	Radius float64
	Mass   float64 // масса: чем выше редкость/больше, тем тяжелее пушить; 0 = использовать Radius
	Zone   int

	// Награда за убийство (опыт) — игрок, чья петаль нанесла последний удар.
	Damage   map[string]float64 // накопленный урон по каждому игроку (по ID)
	KillerID string             // игрок, нанёсший смертельный удар

	moveTimer float64
	dirX      float64
	dirY      float64
	state     string
	hitCD     float64

	// Целенаправленная агрессия: моб преследует конкретного игрока,
	// даже если тот вне радиуса агро (например, всей группой Pentagons
	// после того, как одного из отряда ударили).
	aggroTo string
	aggroT  float64

	// Hive: следующий порог HP для спавна моба (15% интервалы).
	hiveNextSpawn float64
	// Shiny/Golden: золотой моб (спавн 0.1% или админ).
	Golden bool
	// Hive: игнорирует столкновения с другими мобами.
	noMobCollide bool
	// Hive: при смерти преследующий Hexagon заспавнен.
	hiveDeathSpawned bool

	// Actal: мини-копии не размножаются при смерти (иначе экспоненциальный рост).
	noClone bool

	// Орбитальные петали боссов (Fractal=4, Chaos=6): вращаются вокруг моба
	// (1 круг за 5 с), живут своей жизнью — имеют HP и урон, бьют игроков и их
	// петали с кулдауном CD, а петали игроков в ответ могут убить их.
	petals []BossPetal

	// Яд (петаль-«гадюка»): периодический урон пока PoisonT > 0. Damage-карта и
	// убийство при смерти от яда начисляются PoisonBy (кто отравил).
	PoisonDps float64
	PoisonT   float64
	PoisonBy  string

	// Замедление (петаль Rhombus): моб с SlowT > 0 ходит со скоростью
	// Speed×(1-SlowFrac). Накладывается только если редкость моба <= редкости
	// петеля (Fractal Rhombus не замедлит Chaos-моба, но замедлит Fractal).
	SlowFrac float64
	SlowT    float64

	// Руны (петаль rune): накопленные стаки на мобе и таймер до их сброса.
	RuneStacks int
	RuneT      float64
	// «Шиповое тело» игрока (петаль spike): тик колющего урона мобу при
	// контакте (раз в spikeGrindCD).
	SpikeT float64

	// Comet (моб Line): фазовый цикл «полёт → зарядка → рывок → отдых».
	// Фазы и их длительности слегка случайны (см. cometPhaseSpeed).
	cometPhase int
	cometT     float64
	cometDur   float64
	cometDX    float64 // направление рывка (фиксируется в момент перехода к рывку)
	cometDY    float64
	// Arrow (моб Triangle): цикл «разгон → крейсер → торможение → отдых».
	arrowPhase int
	arrowT     float64
	arrowDur   float64
	// Beam (моб Line): таймер до следующего энергетического шара.
	BeamT float64
	// Признак, что смерть моба уже награждена (защита от повторного
	// awardMobKill при уроне из нескольких источников в один тик).
	DeadAwarded bool

	// Витринный моб TestRoom: статичный спрайт-мокап без ИИ и урона (никакие
	// петали/снаряды/яд его не трогают). Label — крупный номер над мобом.
	Showcase bool
	Label    string
}

type MobType struct {
	Name       string
	Color      uint32
	BaseHP     float64
	BaseDMG    float64
	BaseSpeed  float64
	BaseRadius float64
	AggroRange float64
	Shape      string
	Peaceful   bool // нейтральный моб: агрится только если игрок сам его ударил
}

type MobState struct {
	ID         string           `json:"id"`
	Type       string           `json:"type"`
	Shape      string           `json:"shape"`
	Name       string           `json:"name"`
	Rarity     string           `json:"rarity"`
	RarityI    int              `json:"rarityI"`
	BossPetals []BossPetalState `json:"bosspetals,omitempty"` // орбитальные петали босса
	X          float64          `json:"x"`
	Y          float64          `json:"y"`
	Size       float64          `json:"size"`
	HP         float64          `json:"hp"`
	MaxHP      float64          `json:"maxHp"`
	DMG        float64          `json:"dmg"`
	Color      uint32           `json:"color"`
	Zone       int              `json:"zone"`
	DirX       float64          `json:"dirX"`
	DirY       float64          `json:"dirY"`
	Speed      float64          `json:"speed"`
	Ally       bool             `json:"ally,omitempty"`     // союзный моб (миньон петеля)
	Owner      string           `json:"owner,omitempty"`    // чей миньон (ID игрока) — для миникарты
	Poison     bool             `json:"poison,omitempty"`   // моб отравлен (для зелёной подсветки)
	Slow       bool             `json:"slow,omitempty"`     // моб замедлен (Rhombus — для синей подсветки)
	Showcase   bool             `json:"showcase,omitempty"` // витринный мокап TestRoom
	Label      string           `json:"label,omitempty"`    // крупный номер над мокапом ("1".."20")
	Golden     bool             `json:"golden,omitempty"`   // Shiny/Golden моб (клиент: золотая обводка бара)
}

// ---- Петаль-предмет (лежит на земле) ----

type GroundPetal struct {
	ID      string
	Rarity  int
	X, Y    float64
	Size    float64
	HP      float64
	MaxHP   float64
	DMG     float64
	Mass    float64 // масса предмета (0 = вычислить от редкости)
	Color   uint32
	Zone    int
	Kind    string
	OwnerID string  // кто может видеть/подобрать; "" — видят все
	Expire  float64 // через сколько сек исчезает, если не подобрали (0 = вечно)
}

type GroundPetalState struct {
	ID      string  `json:"id"`
	Rarity  string  `json:"rarity"`
	RarityI int     `json:"rarityI"`
	X       float64 `json:"x"`
	Y       float64 `json:"y"`
	Size    float64 `json:"size"`
	HP      float64 `json:"hp"`
	MaxHP   float64 `json:"maxHp"`
	Color   uint32  `json:"color"`
	Zone    int     `json:"zone"`
	Kind    string  `json:"kind,omitempty"`
	Mine    bool    `json:"mine,omitempty"` // true, если дроп предназначен этому клиенту
}

// ---- Орбитальная петаль игрока ----

// PetalDef — «предмет» петали (инвентарь/хотбар). Определяет, какая петаль
// есть у игрока; боевую копию на орбите даёт Petal. Комбинация (Kind, Rarity)
// определяет статы и внешний вид.
type PetalDef struct {
	ID     string  `json:"id"`
	Kind   string  `json:"kind"`
	Rarity int     `json:"rarity"`
	HP     float64 `json:"hp"`
	MaxHP  float64 `json:"maxHp"`
	DMG    float64 `json:"dmg"`
	Size   float64 `json:"size"`
	Color  uint32  `json:"color"`
	Mass   float64 `json:"mass,omitempty"` // масса петеля (0 = вычислить от редкости)
	Name   string  `json:"name,omitempty"` // имя вида (для отображения)
}

// InvStack — стак одинаковых петалей в инвентаре (вид + редкость совпадают).
// Count — сколько таких петалей в запасе (не на хотбаре). Встраивает PetalDef,
// поэтому сериализуется «плоско»: все поля PetalDef + count.
type InvStack struct {
	PetalDef
	Count int `json:"count"`
}

type Petal struct {
	ID       string
	OwnerKey string // устойчивый ID предмета-петеля (PetalDef.ID) для миньонов
	Angle    float64
	Dist     float64
	HP       float64
	MaxHP    float64
	DMG      float64
	Mass     float64 // «вес удара» петеля (0 = вычислить от редкости)
	// Дробная атака: AtkT — накопленный заряд (в долях удара, ≥1 = готов),
	// Charging — идёт ли сейчас зарядка после удара. Позволяет per-kind КД
	// (в т.ч. с cog ×0.10) жить ниже тика сервера: низкий КД даёт НЕСКОЛЬКО
	// ударов за тик (burst), так что «10 Chaos cog» честно повышает DPS.
	AtkT     float64
	Charging bool
	// Recharge — время до полного восстановления (постоянность).
	Recharge float64
	CDMax    float64 // полная длительность текущей перезарядки
	BaseDist float64
	Rarity   int
	Color    uint32
	Size     float64
	Kind     string
	// Яд (ПВП): петаль противника-«гадюки» вешает на эту петаль единый DoT-пул
	// (PoisonDps/PoisonT), тикает в applyPoison; смерть от яда — обычная
	// перезарядка петеля (Recharge).
	PoisonDps float64
	PoisonT   float64
	// OrbT — кулдаун УДАРА ЭТОЙ петали по энергетическим шарам Beam (beamorb).
	// Своя каденция peбyша (petalAtkCD вида), чтобы НЕ трогать боевой
	// AtkT/Charging (шар не ворует у петалУ удар у мобов и не ускоряет её).
	OrbT float64
	// Заряд щита (петаль shield): таймер до следующего пополнения Shield игрока.
	ShieldChargeT float64
}

// cdOf возвращает время (сек) до готовности петаля: полная перезарядка
// (Recharge). Короткий ударный КД (AtkT) — клиенту не показываем: при новых
// per-kind/заводк-ных темпах он ниже тика сервера и «вездесущая» вуаль
// только мешает. 0 = петаль активна.
func cdOf(p Petal) float64 {
	return p.Recharge
}

// isUniquePetal — виды-аксессуары с максимумом 1 на активной линейке
// (глаз, линия, кольца сатурна). Повторное надевание на ту же линейку
// блокируется в EquipPetal/MovePetal.
func isUniquePetal(kind string) bool {
	return kind == "eye" || kind == "line" || kind == "saturn"
}

// overdriveMult возвращает множитель ВРЕМЕНИ ПЕРЕЗАРЯДКИ всех петалей
// игрока от живых cog-Петалей («заводка»): произведение
// (1 - overdriveBase - overdrivePerRar×редкость) по каждому живому cog,
// не ниже 1-overdriveCap (минимум 10% базового времени). Влияет на полную
// перезарядку петалей (восстановление сломанных / смену хотбара /
// способности), НЕ на короткий кулдаун удара.
func overdriveMult(p *Player) float64 {
	m := 1.0
	for i := range p.petals {
		pet := &p.petals[i]
		if petalKinds[pet.Kind].Overdrive && pet.HP > 0 && pet.Recharge <= 0 {
			m *= 1 - overdriveBase - overdrivePerRar*float64(pet.Rarity)
		}
	}
	if m < 1-overdriveCap {
		m = 1 - overdriveCap
	}
	return m
}

// petalRechargeCD — время ПОЛНОЙ перезарядки петели (после поломки / смены
// хотбара / способности) с учётом «заводки» (cog): каждый cog умножает время
// перезарядки на (1 - overdriveBase - overdrivePerRar×редкость), стакается,
// кап — минимум 10% базового времени. Восстановление всех петалей скорее.
func petalRechargeCD(p *Player, base float64) float64 {
	return base * overdriveMult(p)
}

// petalAtkCDs — БАЗОВЫЙ кулдаун удара (сек) для каждого вида петали.
// Тяжели/танки/сильные DPS-петаля — медленнее, лёгкие контроллеры — быстрее.
// Отсутствие вида = petalDamageCD (0.1с). Аксессуары (eye/line/spike) и
// entropy кулдауна НЕ имеют (DMG 0, чистые пассивы) — 0 записан явно.
var petalAtkCDs = map[string]float64{
	"oval":       0.10, // лечебный сустейн — стандартный темп
	"thorns":     0.12, // размен: чуть медленнее
	"rock":       0.14, // танк
	"ball":       0.16, // бомба
	"orb":        0.12, // регенер
	"pebble":     0.05, // праща — почти без КД (быстрый контроль)
	"shard":      0.12, // рана
	"cog":        0.12, // заводка
	"whirl":      0.12, // спиннер
	"blade":      0.10,
	"viper":      0.13, // яд
	"cube":       0.28, // тяжёлый пуш
	"bulwark":    0.25, // танк-стат
	"penta":      0.12,
	"shield":     0.15,
	"hex":        0.14,
	"hiveling":   0.20, // миссил-щелчок
	"fractalite": 0.30, // RNG-двойка, очень высокий урон
	"spark":      0.32, // цепь бьёт 3 сразу — КД высокий
	"fragwall":   0.25, // деф
	"void":       0.40, // dash-петаль с большим уроном
	"tear":       0.15,
	"compass":    0.20,
	"saturn":     0.15,
	"rune":       0.20,
	"twirl":      0.14, // вертушка-разворот: средний темп
	"eye":        0, // пассив-аксессуар
	"line":       0, // пассив-аксессуар
	"spike":      0, // пассив-аксессуар (шиповое тело)
	"entropy":    0, // pet-спавнер, сам не бьёт
	"singularity": 0, // pet-спавнер (Black Hole миньоны), сам не бьёт
}

// petalAtkCD — базовый кулдаун удара вида (fallback petalDamageCD).
func petalAtkCD(kind string) float64 {
	if cd, ok := petalAtkCDs[kind]; ok {
		return cd
	}
	return petalDamageCD
}

// eyeOrbitMult возвращает множитель радиуса орбиты от живых eye-петалей.
func eyeOrbitMult(p *Player) float64 {
	for i := range p.petals {
		pet := &p.petals[i]
		if petalKinds[pet.Kind].Eye && pet.HP > 0 && pet.Recharge <= 0 {
			return 1 + eyeOrbitPerRar*float64(pet.Rarity)
		}
	}
	return 1.0
}

// tearHealFrac возвращает долю MaxHP, которую лечит tear при разрушении.
func tearHealFrac(rar int) float64 {
	return tearHealBasePct * (1 + 0.5*float64(rar))
}

// isOrbitAccessory — «навесные» аксессуары (eye/line/spike): висят прямо на
// игроке, НЕ раскладывают остальные петали по орбите и не занимают места.
func isOrbitAccessory(kind string) bool {
	return kind == "eye" || kind == "line" || kind == "spike"
}

// bodySpikeDmg — колющий урон «тела» игрока при контакте с мобом (петаль
// spike). СКЛАДЫВАЕТСЯ из ВСЕХ живых spike на активной линейке ( каждый
// добавляет spikeBodyBase × rarityDanger[редкость]).
func bodySpikeDmg(p *Player) float64 {
	dmg := 0.0
	for i := range p.petals {
		pet := &p.petals[i]
		if pet.Kind != "spike" || pet.HP <= 0 || pet.Recharge > 0 {
			continue
		}
		mult := petalPower[minInt(pet.Rarity, len(petalPower)-1)]
		dmg += spikeBodyBase * mult
	}
	return dmg
}

type PetalState struct {
	ID           string  `json:"id"`
	X            float64 `json:"x"`
	Y            float64 `json:"y"`
	Angle        float64 `json:"angle"`
	Size         float64 `json:"size"`
	Color        uint32  `json:"color"`
	HP           float64 `json:"hp"`
	MaxHP        float64 `json:"maxHp"`
	Rarity       int     `json:"rarity"`
	CD           float64 `json:"cd"`    // 0 = активна; >0 = полная перезарядка (Recharge)
	CDMax        float64 `json:"cdMax"` // полная длительность перезарядки (для отображения)
	Kind         string  `json:"kind,omitempty"`
	Poison       bool    `json:"poison,omitempty"`
	ShieldCharge float64 `json:"shieldCharge"` // 0..1 прогресс заряда щита (для анимации подлёта к игроку)
}

// ---- Понятие «массы» ----
// Масса определяет физику отталкивания: чем тяжелее объект (выше редкость /
// больше размер), тем меньше его сдвигают чужие пуши и тем сильнее он сам
// пушит более лёгких соседей. Поле Mass может быть задано вручную; 0 означает
// «вычислить по умолчанию» — тогда масса берётся из радиуса (для мобов) или
// растёт от редкости (для петалей).

// mobMass — масса моба: по умолчанию равна радиусу (редкость влияет через
// raritySize). Заданное вручную поле Mass выигрывает.
func mobMass(m *Mob) float64 {
	if m.Mass > 0 {
		return m.Mass
	}
	return m.Radius
}

// orbitPetalMass — масса орбитального петеля игрока: более редкий петаль бьёт
// тяжелее и сильнее пушит мобов при касании.
func orbitPetalMass(p Petal) float64 {
	if p.Mass > 0 {
		return p.Mass
	}
	return 1 + float64(p.Rarity)*1.2
}

// groundPetalMass — масса петеля-предмета на земле (для расталкивания мобами).
func groundPetalMass(g *GroundPetal) float64 {
	if g.Mass > 0 {
		return g.Mass
	}
	return 1 + float64(g.Rarity)*0.5
}

// Масса петеля по редкости (для PetalDef, чтобы деф сразу носил свой вес).
func petalDefMass(rarity int) float64 {
	return 1 + float64(rarity)*1.2
}

// normalizePetalDef приводит статы петали к ФИКСИРОВАННЫМ значениям её пары
// (вид, редкость): HP/DMG = Base × множитель редкости, размер = 8×1.2^редкость,
// масса/цвет/имя — от редкости. Вызывается на всех путях создания и при чтении
// профиля из БД, поэтому любые старые/битые значения в БД не влияют на игру:
// одинаковые петали ВСЕГДА имеют одинаковые размер, HP и урон.
func normalizePetalDef(d *PetalDef) {
	if d.Kind == "" {
		d.Kind = "oval"
	}
	kn, ok := petalKinds[d.Kind]
	if !ok {
		d.Kind = "oval"
		kn = petalKinds["oval"]
	}
	rar := d.Rarity
	if rar < 0 {
		rar = 0
	}
	if rar > len(petalPower)-1 {
		rar = len(petalPower) - 1
	}
	d.Rarity = rar
	mult := petalPower[rar]
	d.HP = kn.BaseHP * mult
	d.MaxHP = d.HP
	d.DMG = kn.BaseDMG * mult
	d.Size = petalSizeBase * math.Pow(1.2, float64(rar))
	d.Mass = petalDefMass(rar)
	d.Color = groundPetalColor(rar)
	d.Name = kn.Name
}

// ---- Константы орбиты петалей ----

const (
	petalOrbitSpeed = 1.5
	petalDamageCD   = 0.1 // кулдаун удара петеля (в 3 раза быстрее: больше ударов при контакте)
	petalRechargeT  = 5.0 // сколько секунд петаль перезаряжается после "поломки" (не исчезает)
	switchRechargeT = 5.0 // короткая перезарядка петалей после смены хотбара (клавиша X)
	respawnInvuln   = 3.0 // иммунитет к урону (сек) после респавна / выхода из портала
	petalBaseDist   = 55.0
	petalMinDist    = 25.0
	petalMaxDist    = 100.0
	// Eye (петаль eye, макс. 1): множитель радиуса орбиты ВСЕХ петалей на
	// редкость: 1 + eyeOrbitPerRar×редкость (r7 ≈ ×1.7). Действует, пока глаз
	// жив. Переход РАВНОМЕРНЫЙ, не резкий (см. p.eyeRamp в updatePlayer).
	eyeOrbitPerRar = 0.1
	// Tear (петаль tear): лечение при РАЗРУШЕНИИ петеля мобом или ядом:
	// tearHealBasePct × (1 + 0.5×редкость) от MaxHP. Стакается.
	tearHealBasePct = 0.15
	// Cog (петаль cog, «заводка»): каждый cog с редкостью r умножает ВРЕМЯ
	// ПЕРЕЗАРЯДКИ всех петалей (восстановление сломанных, смена хотбара,
	// кулдауны способностей) на (1 - overdriveBase - overdrivePerRar×r),
	// стакается; кап — минимум 10% базового времени (overdriveCap).
	// «10 Chaos cog» = сущестскорый супер-DPS за счёт скорости поломок.
	overdriveBase   = 0.08
	overdrivePerRar = 0.04
	overdriveCap    = 0.90
	// Saturn (петаль saturn, макс. 1): круг-аура вокруг игрока (ЗАПОЛНЕННЫЙ диск —
// бьёт ВСЕХ врагов внутри, не только край). Радиус =
// saturnRingBase + saturnRingPerRar×редкость (бафф: после нерфа 100+20 снова
// 140+30 — радиус заметно больше); враги в круге (мобы И орбитальные петали
// боссов) тикаются saturnDpsBase×petalPower урона в секунду (бафф 5→8 — аура
// реально держит свой радиус). Аура БЕЗ коллизии: никого не толкает (отброс
// убран), это чистая зона урона. Визуал — на клиенте (светящийся круг).
	saturnRingBase   = 140.0
	saturnRingPerRar = 30.0
	saturnRingThick  = 26.0
	saturnDpsBase    = 8.0
	// Rune (петаль rune): удары вешают руну на моба; runeStackNeed стаков →
	// взрыв AoE runeBoomMult×DMG петеля в runeBoomRadius вокруг моба.
	runeStackNeed  = 3
	runeStackDur   = 6.0
	runeBoomMult   = 2.0
	runeBoomRadius = 130.0
	// Line (петаль line): потолок радиуса обзора на клиенте (защита от лагов
	// при геометрическом 1.1^редкость на высоких редкостях).
	lineViewCap = 20000.0
	// Rhombus (петаль): каждый удар замедляет моба на rhombusSlowPct на
	// rhombusDur секунд (перезаписывается свежим, макс. SlowT = rhombusMaxT).
	// Действует ТОЛЬКО если редкость моба <= редкости петеля
	// (см. fightPetalsVsMobs): Fractal Rhombus не замедлит Chaos-моба.
	rhombusSlowPct = 0.45
	rhombusDur     = 3.0
	rhombusMaxT    = 5.0
	// Comet (моб Line, «рывок»): каждые несколько секунд плавно останавливается
	// (charge 1.4с), затем РЫВОК по направлению движения (dashMult от базовой
	// скорости — «как бык»; растёт с редкостью +10%/rarity, r7 ≈ ×1.7) и
	// короткий отдых. Длительности фаз слегка случайны.
	cometChargeDur  = 1.4
	cometDashDur    = 0.35
	cometDashMult   = 7.35 // 4.2 × 1.75 (ребаланс «быка»)
	cometRestDur    = 0.8
	cometWaitDur    = 3.0
	cometDashPerRar = 0.1 // +10% скорости рывка за редкость
	// Arrow (моб Triangle, «ускорение»): цикл разгон→крейсер→торможение→отдых
	// со скоростью ×1.25 (после нерфа; раньше было ×3 — слишком неубиваемые).
	// Случайные тайминги фаз, не сильно различающиеся.
	arrowAccelDur   = 2.0
	arrowCruiseDur  = 2.0
	arrowBrakeDur   = 1.6
	arrowRestDur    = 1.2
	arrowSprintMult = 1.25 // множитель скорости в спринте (разгон/крейсер)
	// +- Moб Shiny/Golden (спавн с шансом 0.1%): сам моб золотого цвета,
	// к имени добавляется "Golden", HP ×20 (обычные) или ×50 (Fractal/Chaos),
	// дроп ×10 (топ-дамагерам). В админ-панели выбирается вручную (nothing/gold).
	goldSpawnChance = 0.001
	goldHPMult      = 20.0
	goldBossHPMult  = 50.0 // ×50 HP от базового для Fractal/Chaos золотых
	goldLootRolls   = 10   // ровно в 10 раз больше попыток дропа на топ-дамагера
	goldColor       = 0xffd700

	// spikeBodyBase (петаль spike, «шиповое тело»): контактный урон тела игрока
	// за ОДИН надетый spike r0 (× petalPower; СКЛАДЫВАЕТСЯ от всех живых
	// spike на орбите — «прибавляет урон к телу игрока»). Бафф для играбельности:
	// 5→20 и тик вдвое чаще (0.5с) — Point-шип даёт 40 DPS в контакте телом.
	spikeBodyBase = 20.0
	spikeGrindCD  = 0.5 // пауза «царапания» моба шипами (тик урона раз в 0.5с)
	// petalKnockbackK — коэффициент отброса моба от удара петеля: реальный сдвиг
	// = K * масса/редкость петеля / масса моба. При K=26 средний моб (радиус 20)
	// от летящей Polygon-петали улетает как раньше (~6px), а Chaos-босс (радиус
	// 300+) почти не сдвигается.
	petalKnockbackK = 26.0
	// Яд (петали с PetalKind.Poison): за удар вешается DoT длительностью
	// poisonDuration, тикающий poisonDpsMult * DMG петаля в секунду.
	// Пул ЕДИНЫЙ и НЕ стакается: свежий удар освежает таймер и удерживает
	// МАКСИМАЛЬНУЮ силу среди наложений (take-max в poison.go) — несколько
	// гадюк НЕ складывают ДПС (одна орбита = один яд). Ребаланс (нерф):
	// было 10 DMG / 18× / 4 с — square-wiper «снёс» Polygon-моба; стало
	// 8 DMG / 10× / 3.5 с (по тиру ≈ Spark по ДПС). Плюс tier-гейт
	// poisonCrossTierMult: на мобе редкости ВЫШЕ петаля яд работает в
	// 0.35× — petal тира ниже уже не выносит целый тир вверх.
	// У shard своя короткая «рана» woundDpsMult (не яд).
	poisonDuration = 3.5
	poisonDpsMult  = 10.0
	poisonCrossTierMult = 0.35
	// Шипы (петали с PetalKind.Thorns): моб, изнашивающий шип ударом, сам
	// получает часть этого износа обратно — «размен». Урон считается уроном
	// игрока (идёт в Damage-карту и топ-дамагерам). Выгодно против тяжело
	// бьющих боссов, слабо — против быстрых слабых мобов.
	thornsReflectRatio = 0.4
	// Реген (петаль orb): доля от макс. HP игрока в секунду × rarityDanger,
	// НЕ стакается (только максимум-редкость orb). Point-orb лечит ~3%/с,
	// Chaos-orb — 3%×17 ≈ 51%/с.
	orbRegenFrac = 0.03
	// MaxHP (петаль bulwark): доля от БАЗОВОГО MaxHP игрока × rarityDanger.
	// Ещё бафф (было 0.2 → 0.6): Point-бастион +80% (16 HP),
	// Chaos-бастион +80%×17 ≈ +1360%.
	bulwarkMaxHPFrac = 0.8
	// Магнит (петаль hex): радиус притяжения дропа растёт с редкостью, скорость
	// подтягивания — константа (быстрое притягивание к игроку).
	hexMagnetBaseRadius   = 250.0
	hexMagnetRadiusPerRar = 45.0
	hexMagnetSpeed        = 400.0
	// Разрыв (петаль hiveling): число шариков, доля урона петеля, скорость и
	// время полёта каждого. Шарики разлетаются во все стороны (равномерно по
	// кругу) и пробивают мобов (pierce), поэтому разрыв виден даже в бою.
	hivelingShardCount   = 5
	hivelingShardDmgFrac = 0.6
	hivelingShardSpeed   = 320.0
	hivelingShardLife    = 1.6
	// Миссил (петаль blade): скорость, время полёта и кулдаун после выстрела.
	// Петаль САМ вылетает и уходит в полную перезарядку (Recharge) на bladeCooldown;
	// кулдаун = время полёта, чтобы петаль возвращалась ровно к концу вылета
	// (без «дыры») и спамить пробелом было нельзя.
	bladeFlySpeed = 380.0
	bladeFlyLife  = 1.2
	bladeCooldown = 1.2
	// Износ снаряда от попадания по телу игрока в ПВП (HP снаряда убывает).
	pvpMissileBodyWear = 4.0
	// Гравитационный толчок (петаль cube, Push): коэффициент отброса сильнее
	// обычного petalKnockbackK — куб тяжёлый и крупный, отбрасывает мобов назад.
	cubePushK = 90.0
	// Цепь (петаль spark, Chain): сколько звеньев, радиус, убывание урона за
	// звено и время жизни визуальной линии-цепи.
	sparkChainCount   = 6
	sparkChainRange   = 320.0
	sparkChainFalloff = 0.25
	sparkChainLife    = 0.3
	// Рывок (петаль void, Dash): дистанция рывка и перезарядка петеля после.
	voidDashDist = 180.0
	voidCooldown = 3.0
	// Шаг рывка: всё расстояние voidDashDist проходится мелкими шагами с клампом
	// на каждом — рывок не может «перепрыгнуть» разделительную стену между Mob
	// Zone и Safe Zone (стены тоньше дистанции; без шагов кламп после прыжка
	// не видит пересечения и игрок проскакивает насквозь).
	voidDashStep = 6.0
	// Лайфстил (oval): доля от нанесённого урона, лечащая игрока.
	lifestealPct = 0.3
	// Броня (rock): множитель износа от ударов мобов (0.5 = вдвое меньше).
	armorWearMult = 0.5
	// Бомба (ball): радиус АОЕ и доля урона по соседям.
	bombRadius  = 80.0
	bombDmgFrac = 0.5
	// Праща (pebble): сильный толчок и почти мгновенный кулдаун удара
	// (pebbleHitCD — теперь в petalAtkCDs, тут оставлена константа-значение).
	pebblePushK = 80.0
	pebbleHitCD = 0.05
	// Рана (shard): длительность и сила короткого DoT.
	woundDuration = 1.0
	woundDpsMult  = 0.5
	// Ускоритель (cog): прибавка к вращению орбиты за петаль (базовая и за редкость).
	cogSpinBase   = 0.3
	cogSpinPerRar = 0.1
	// Ускорение (penta): +% к скорости передвижения за петаль (базовый и за редкость).
	swiftBasePct   = 0.10
	swiftPerRarity = 0.02
	// Самоподобие (fractalite): шанс двойного удара.
	twinChance = 0.4
	// Сглаживание движения игрока: коэффициент ускорения/децелерации к целевой
	// скорости (больше — резче разворот, меньше — плавнее).
	playerMoveAccel = 12.0
	// Барьер-щит (fragwall): снижение входящего урона за петаль (базовое и за
	// редкость), стакается, но не ниже 20% от полного урона.
	guardBasePct   = 0.05
	guardPerRarity = 0.02
	// Щит (петаль shield): ёмкость щита за петаль (×rarityDanger), заряд за одно
	// пополнение, интервал пополнения и кулдаун после пополнения (петаль исчезает
	// на время и НЕ респавнится моментально). Нерф ×2 (было 20/10): щит теперь
	// вдвое слабее — класс защиты переехал на Bulwark.
	shieldCapBase        = 10.0
	shieldChargeBase     = 5.0
	shieldChargeInterval = 1.0
	shieldRechargeT      = 2.5
	// Реген вне боя: через outOfCombatDelay секунд БЕЗ ПОЛУЧЕНИЯ урона игрок
	// медленно восстанавливает HP — outOfCombatRegenFrac от макс. HP в секунду
	// (≈1.5%/с), до полного MaxHP. Любой входящий урон обнуляет таймер.
	outOfCombatDelay     = 20.0
	outOfCombatRegenFrac = 0.015
	// Базовые статы орбитальной петали Point:
	petalBaseHP   = 5.0
	petalBaseDMG  = 3.0
	petalSizeBase = 8.0
	petalColor    = 0xecf0f1
)

// petalWorldPos возвращает мировые координаты орбитальной петали игрока вокруг
// него. Радиус орбиты масштабируется от размера игрока (p.State.Size): чем больше
// игрок (напр. админ увеличил себя кнопкой «Размер +»), тем дальше петали
// раздвигаются, оставаясь на орбите. При стандартном размере (playerSize) орбита
// остаётся прежней (dist).
func petalWorldPos(p *Player, angle, dist float64) (float64, float64) {
	// Аксессуары (eye/line/spike/saturn) висят РОВНО в центре игрока (dist=0):
	// НЕ поднимаем радиус до минимума, иначе чужие «line» уезжают вправо на 8px
	// (у себя баг маскировался клиентским перецепом в центр).
	if dist <= 0 {
		return p.State.X, p.State.Y
	}
	r := dist * (p.State.Size / playerSize)
	if r < 8 {
		r = 8
	}
	return p.State.X + math.Cos(angle)*r, p.State.Y + math.Sin(angle)*r
}

// pickRarityFromWeights выбирает индекс по весам.
func pickRarityFromWeights(w []int) int {
	total := 0
	for _, v := range w {
		total += v
	}
	if total <= 0 {
		return 0
	}
	r := rand.Intn(total)
	for i, v := range w {
		r -= v
		if r < 0 {
			return i
		}
	}
	return len(w) - 1
}

// applyRarityClamp ограничивает редкость зоны [floor, ceil] с сохранением
// градиента: веса ниже пола сливаются в пол (нижние полосы второго уровня
// стартуют с Square), веса выше потолка отрезаются (первый уровень не выше
// Square). Пустым результат не бывает — пол всегда непустой.
func applyRarityClamp(w []int, floor, ceil int) []int {
	n := len(w)
	if floor <= 0 && ceil >= n-1 {
		return w
	}
	if floor < 0 {
		floor = 0
	}
	if ceil >= n {
		ceil = n - 1
	}
	out := make([]int, n)
	below := 0
	for i := 0; i < floor; i++ {
		below += w[i]
	}
	if floor < n {
		out[floor] += below
	}
	for i := floor; i <= ceil; i++ {
		out[i] += w[i]
	}
	return out
}

// zoneRarityAllowed — разрешена ли редкость rar в зоне (по капу/полу).
func zoneRarityAllowed(zone, rar int) bool {
	if zone < 0 || zone >= len(zoneRarityFloor) {
		return true
	}
	return rar >= zoneRarityFloor[zone] && rar <= zoneRarityCeil[zone]
}

// pickRarityAtY выбирает редкость по позиции Y в Mob-зоне: выше = реже.
// Зоны второго уровня — отдельная лестница «большими прямоугольниками»
// (l2RarityBands); первый уровень — общая 8-полосная лестница (rarityBands)
// с капом зоны (первый уровень — до Square, второй — Square..Chaos).
func pickRarityAtY(z *Zone, y float64) int {
	if !hasSafeZone(z) {
		return pickL2RarityAtY(z, y)
	}
	w := applyRarityClamp(rarityBands[bandForY(z, y)], zoneRarityFloor[z.Index], zoneRarityCeil[z.Index])
	return pickRarityFromWeights(w)
}

// pickRarity случайно выбирает редкость для зоны (для совместимости —
// по случайной позиции в Mob-зоне).
func pickRarity(zone int) int {
	z := zones[zone]
	return pickRarityAtY(&z, z.Y+rand.Float64()*(mobZoneBottom(&z)-z.Y))
}

// randMobType выбирает случайный тип моба в зоне с учётом весов
// (zoneSpawnWeights; при отсутствии — равномерно).
func randMobType(zone int) string {
	opts := mobTypesInZone[zone]
	if len(opts) == 0 {
		return "dot"
	}
	ws := zoneSpawnWeights[zone]
	if len(ws) != len(opts) {
		return opts[rand.Intn(len(opts))]
	}
	total := 0
	for _, w := range ws {
		total += w
	}
	if total <= 0 {
		return opts[rand.Intn(len(opts))]
	}
	r := rand.Intn(total)
	for i, w := range ws {
		r -= w
		if r < 0 {
			return opts[i]
		}
	}
	return opts[len(opts)-1]
}

// pickDropRarity выбирает редкость петеля-дропа по редкости моба
// (см. mobRarityDrop). Chaos-моб не бросает Chaos (максимум Fractal).
func pickDropRarity(mobRarity int) int {
	w := mobRarityDrop[mobRarity]
	total := 0
	for _, v := range w {
		total += v
	}
	if total <= 0 {
		return 0
	}
	r := rand.Intn(total)
	for i, v := range w {
		r -= v
		if r < 0 {
			return i
		}
	}
	return 0
}

// mobDmgMult — ОБЩИЙ множитель урона ВСЕХ мобов (применяется в scaleMobForRarity).
const mobDmgMult = 2.0

// scaleMobForRarity пересчитывает боевые статы моба под редкость rar
// (HP, урон, радиус и — для улья — порог спавна). Используется при создании
// моба и при понижении редкости под капами (максимум 1 Chaos на игру,
// максимум 1 Fractal на зону). HP растёт по mobHPDanger (более крутая кривая,
// чем урон): высокие тиры заведомо мясистее. Chaos — финальный босс: его HP
// дополнительно усилено в 80 раз (намного выше любого Fractal).
// mobStatsForRarity — точные HP и DMG реального враждебного моба типа mtype
// редкости rar. Единая формула со scaleMobForRarity (без Golden-множителей):
// используется и для призванных миньонов (Entropy/Singularity), чтобы их HP и
// урон совпадали с враждебными мобами того же имени и той же редкости.
func mobStatsForRarity(mtype string, rar int) (hp, dmg float64) {
	mt := mobTypes[mtype]
	hp = mt.BaseHP * mobHPDanger[rar] * mobHPScale * 2.0
	// Polygon-редкость (4): HP ×2 (ребаланс).
	if rar == 4 {
		hp *= polyMobHPMult
	}
	// Hexagon-редкость (5) — «мини-босс».
	if rar == 5 {
		hp *= hexMobHPMult
	}
	if rar >= len(rarityNames)-1 {
		hp *= chaosHPMult
	}
	// Fractal — жирный босс: должен убиваться ~50-60 секунд.
	if rar == len(rarityNames)-2 {
		hp *= fractalHPMult
	}
	dmg = mt.BaseDMG * rarityDanger[rar] * mobDmgMult
	return
}

// Дебаф призывных мобов глубоких редкостей (петали Hexagon/Fractal/Chaos):
// без боссовых множителей они всё ещё сильнее обычных призывов — вдвое ослабляем
// HP и урон, чтобы призывы Black Hole не были неубиваемыми.
const summonDeepDebuff = 0.5

// summonMobStats — статы для ПРИЗЫВНЫХ миньонов (Entropy/Singularity): кривые
// как у реальных мобов, но БЕЗ боссовых множителей HP (Hexagon-редкость ×16,
// Fractal ×200, Chaos ×640) — с ними призывы глубоких редкостей были бы сильнее
// оригинальных мобов и неубиваемыми. Плюс явный дебаф ×0.5 на HP и урон.
func summonMobStats(mtype string, rar int) (hp, dmg float64) {
	hp, dmg = mobStatsForRarity(mtype, rar)
	switch {
	case rar == 5:
		hp /= hexMobHPMult
	case rar == len(rarityNames)-2:
		hp /= fractalHPMult
	case rar >= len(rarityNames)-1:
		hp /= chaosHPMult
	}
	if rar >= 5 {
		hp *= summonDeepDebuff
		dmg *= summonDeepDebuff
	}
	return
}

func scaleMobForRarity(m *Mob, rar int) {
	mt := mobTypes[m.Type]
	m.Rarity = rar
	sz := raritySize[rar]
	// Hexagon-редкость (5) — «мини-босс»: заметно крупнее и жирнее обычной
	// кривой редкости (ребаланс по просьбе игрока).
	if rar == 5 {
		sz *= hexMobSizeMult
	}
	hp, dmg := mobStatsForRarity(m.Type, rar)
	// Shiny/Golden: HP ×20 (обычные) или ×50 для золотых Fractal/Chaos.
	// Golden-«рассадники» (Hive / Actal) наследуют Golden своим призывам.
	if m.Golden {
		if rar >= len(rarityNames)-2 {
			hp *= goldBossHPMult
		} else {
			hp *= goldHPMult
		}
	}
	m.HP = hp
	m.MaxHP = hp
	m.DMG = dmg
	m.Radius = mt.BaseRadius * limitMobSizeForZone(sz, mt.BaseRadius, m.Zone)
	m.Mass = m.Radius // тяжесть = размер: редкие и крупные мобы тяжелее
	// Hive и Block — «афк»-мобы: коллизия с сородичами выкл для ВСЕХ путей
	// спавна (обычный спавн, саммоны Hive, mini-actal, посмертный Hexagon,
	// админ-спавн), чтобы другие мобы не застревали на них.
	m.noMobCollide = (m.Type == "hive" || m.Type == "block" || m.Type == "tear")
	if m.Type == "hive" {
		m.hiveNextSpawn = m.HP * 0.85
	}
	initBossPetals(m)
}

// chaosHPMult — дополнительный множитель HP финального босса Chaos.
// Ребаланс: 640 — Chaos остаётся сильно выше Fractal (×200): в 5 раз жирнее
// (как и прежде ~×4.8) — «финальный хаос» не теряет статус стены-босса.
const chaosHPMult = 640.0

// Hexagon-редкость (мини-босс) и Polygon: персональные множители ребаланса.
const hexMobSizeMult = 1.17
const hexMobHPMult = 16.0
const polyMobHPMult = 4.0

// fractalHPMult — дополнительный множитель HP босса Fractal (ребаланс ×8: 25→200).
const fractalHPMult = 200.0

// Максимум живых Hexagon-мобов (мини-боссов) в одной зоне.
const hexPerZoneCap = 2

// Полоска Hexagon-таймера: спавн каждые 5-15 минут в верхней части L2-зоны
// (макс hexPerZoneCap на зону).
const hexTimerMin = 300.0
const hexTimerMax = 900.0

// Медленное вращение орбитальных петалей боссов: 1 полный круг за 5 секунд.
const mobPetalAngSpeed = 2 * math.Pi / 5
const mobPetalOrbitMult = 1.9
const mobPetalCD = 0.2      // кулдаун урона петеля босса (в 3 раза быстрее)
const mobPetalHPMult = 0.05 // HP петеля босса = 5% от HP босса
const mobPetalDmgMult = 0.5 // урон петеля босса = 50% от урона тела босса
const bossPetalBaseSize = 30.0
const mobPetalsRespawnMin = 3.0
const mobPetalsRespawnMax = 5.0 // рандом 3-5 с до возрождения уничтоженной петали

// mobPetalCount — сколько петалей орбит у моба (Fractal 4, Chaos 6, прочие 0).
func mobPetalCount(rar int) int {
	if rar >= len(rarityNames)-1 {
		return 6
	}
	if rar == len(rarityNames)-2 {
		return 4
	}
	return 0
}

// BossPetal — орбитальная петаль босса: отдельная боевая сущность со своим HP,
// уроном и кулдауном атаки. Вращается по орбите вокруг моба (угол Angle).
// Уничтоженная петаль не исчезает насовсем: через 3-5 с респавнится на том же
// месте круга (Respawn — обратный отсчёт до возрождения, HP>0 пока жива).
type BossPetal struct {
	Idx     int // стабильный номер слота в массиве m.petals (не меняется при смерти/респавне)
	Angle   float64
	HP      float64
	MaxHP   float64
	DMG     float64
	CD      float64 // кулдаун между ударами (по игроку/его петали)
	Size    float64
	Respawn float64 // таймер до возрождения после уничтожения (0 = жива/готовa)
	// Яд от петалей-«гадюк» (ПВЕ): как у мобов — единый DoT-пул, тикает в
	// applyPoison; убитая ядом петаль уходит в Respawn (нет дропа/наград).
	PoisonDps float64
	PoisonT   float64
}

// BossPetalState — снапшот петеля босса для клиента.
type BossPetalState struct {
	Idx    int     `json:"idx"`
	Angle  float64 `json:"angle"`
	HP     float64 `json:"hp"`
	MaxHP  float64 `json:"maxHp"`
	Size   float64 `json:"size"`
	CD     float64 `json:"cd"`
	Poison bool    `json:"poison,omitempty"`
}

// initBossPetals создаёт (или пересоздаёт) орбитальные петали босса под
// текущую редкость моба. При понижении редкости (кап Chaos/Fractal) угол
// первой петали сохраняется, чтобы визуал не прыгал.
func initBossPetals(m *Mob) {
	n := mobPetalCount(m.Rarity)
	base := 0.0
	if len(m.petals) > 0 {
		base = m.petals[0].Angle
	}
	out := make([]BossPetal, 0, n)
	for i := 0; i < n; i++ {
		hp := m.MaxHP * mobPetalHPMult
		sz := bossPetalBaseSize + m.Radius*0.04
		out = append(out, BossPetal{
			Idx:   i,
			Angle: base + float64(i)*2*math.Pi/float64(n),
			MaxHP: hp,
			HP:    hp,
			DMG:   m.DMG * mobPetalDmgMult,
			Size:  sz,
		})
	}
	m.petals = out
}

// bossPetalPos возвращает мировые координаты петеля босса.
func bossPetalPos(m *Mob, bp *BossPetal) (float64, float64) {
	orbit := m.Radius * mobPetalOrbitMult
	return m.X + math.Cos(bp.Angle)*orbit, m.Y + math.Sin(bp.Angle)*orbit
}

// makeMob создаёт моба со случайным типом и редкостью для зоны.
// Спавн ТОЛЬКО в Mob Zone (выше разделительной стены); редкость растёт с
// высотой (чем ближе к верху зоны, тем выше шанс редких редкостей).
// Моб появляется В ТОЙ ЖЕ точке, по которой выбрана редкость — иначе
// (раньше) Hexagon, выпавший по верхней полосе, мог заспавниться внизу
// зоны, что выглядело бессмысленно (в Point II к порталу внизу).
func makeMob(zone int, id int) Mob {
	tname := randMobType(zone)
	mt := mobTypes[tname]

	z := zones[zone]
	// Точка спавна по высоте Mob-зоны; редкость берётся ПО ЭТОЙ ЖЕ высоте.
	y0 := z.Y + rand.Float64()*(mobZoneBottom(&z)-z.Y)
	rar := pickRarityAtY(&z, y0)

	// Скорость НЕ растёт с редкостью (мобы "весовые"): только базовый показатель,
	// с небольшой случайной вариацией.
	speed := mt.BaseSpeed * (0.8 + rand.Float64()*0.5)

	m := Mob{
		ID:        "mob_" + strconv.Itoa(id),
		Type:      tname,
		Rarity:    rar,
		Speed:     speed,
		Zone:      zone,
		Damage:    make(map[string]float64),
		state:     "patrol",
		moveTimer: rand.Float64() * 3,
		dirX:      rand.Float64()*2 - 1,
		dirY:      rand.Float64()*2 - 1,
	}
	// Shiny/Golden: шанс 0.1% при появлении (работает на ЛЮБУЮ редкость,
	// включая Fractal/Chaos боссов).
	m.Golden = rand.Float64() < goldSpawnChance
	scaleMobForRarity(&m, rar)

	// Позиция: только в Mob Zone (нижняя граница — верх стены), с учётом радиуса.
	// Высота — та же y0 (по ней выбрана редкость); радиусом лишь поджимаем,
	// если y0 слишком близко к краям зоны.
	radius := m.Radius
	top := z.Y + radius
	bot := mobZoneBottom(&z) - radius
	if bot <= top {
		bot = top + 1
	}
	m.X = z.X + radius + rand.Float64()*(z.W-2*radius)
	m.Y = y0
	if m.Y < top {
		m.Y = top
	}
	if m.Y > bot {
		m.Y = bot
	}

	// Hive и Block: коллизия с сородичами выкл — другие мобы свободно проходят
	// сквозь них и не застревают на неподвижных телах.
	if tname == "hive" || tname == "block" || tname == "tear" {
		m.noMobCollide = true
	}
	// Спецспособности мобов: случайный (но устойчиво похожий) старт фаз.
	initSpecMobPhase(&m, tname)
	return m
}

// initSpecMobPhase — инициализация фаз специальностей мобов (comet/arrow/beam).
// Используется и обычным спавном (makeMob), и админ-спавном (makeAdminMob).
func initSpecMobPhase(m *Mob, tname string) {
	switch tname {
	case "comet":
		m.cometPhase = 0
		m.cometT = cometWaitDur * (0.8 + rand.Float64()*0.4)
	case "arrow":
		m.arrowPhase = 0
		m.arrowDur = arrowAccelDur * (0.8 + rand.Float64()*0.4)
		m.arrowT = m.arrowDur
	case "beam":
		m.BeamT = 1 + rand.Float64()*1.5
	}
}

// makeShowcaseMobs создаёт 16 витринных мокапов в TestRoom (зона 16): сетка
// 4x4 статичных спрайтов с крупными номерами 1-16. Это НЕ настоящие мобы —
// без ИИ, без урона, неуязвимые (Showcase), редкость выставляется на 5
// (Hexagon), чтобы над каждым висел бирюзовый чип редкости.
func makeShowcaseMobs(zone int) []Mob {
	type def struct {
		typ   string
		label string
	}
	defs := []def{
		{"mockup1", "1"}, {"mockup2", "2"}, {"mockup3", "3"}, {"mockup4", "4"},
		{"mockup5", "5"}, {"mockup6", "6"}, {"mockup7", "7"}, {"mockup8", "8"},
		{"mockup9", "9"}, {"mockup10", "10"}, {"mockup11", "11"}, {"mockup12", "12"},
		{"mockup13", "13"}, {"mockup14", "14"}, {"mockup15", "15"}, {"mockup16", "16"},
		// Слоты 17-25 пусты: зарезервированы под будущих новых мобов.
	}
	cols := 4
	const (
		padX = 1000.0
		padY = 700.0
	)
	gapX := (zoneW - 2*padX) / float64(cols-1)
	gapY := (zoneH - 2*padY) / (float64(len(defs)/cols) - 1)
	z := zones[zone]
	mobs := make([]Mob, 0, len(defs))
	for i, d := range defs {
		mt := mobTypes[d.typ]
		col := i % cols
		row := i / cols
		hp := 1000000.0
		mobs = append(mobs, Mob{
			ID:     "show_" + strconv.Itoa(i+1),
			Type:   d.typ,
			Rarity: 5,
			X:      z.X + padX + float64(col)*gapX,
			Y:      z.Y + padY + float64(row)*gapY,
			HP:     hp, MaxHP: hp,
			DMG:      0,
			Speed:    0,
			Radius:   mt.BaseRadius,
			Zone:     zone,
			Damage:   make(map[string]float64),
			state:    "showcase",
			dirX:     1,
			Showcase: true,
			Label:    d.label,
		})
	}
	return mobs
}

// limitMobSizeForZone уменьшает множитель размера, чтобы моб влезал в зону
// и был боссом, а не "стеной". Возвращает скорректированный множитель.
func limitMobSizeForZone(sz, baseRadius float64, zone int) float64 {
	maxRadius := zones[zone].W * 0.18
	if sz*baseRadius > maxRadius {
		return maxRadius / baseRadius
	}
	return sz
}

// clampMobInZone ограничивает моба границами Mob-зоны: всегда выше
// разделительной стены (в Safe Zone не заходят ни при каких обстоятельствах)
// и не внутри конфигурируемых стен зоны (walls.go).
func clampMobInZone(m *Mob) {
	z := zones[m.Zone]
	r := m.Radius
	m.X, m.Y = clampZoneRounded(&z, m.X, m.Y, r)
	y0 := z.Y + r
	y1 := mobZoneBottom(&z) - r
	if y1 < y0 {
		y1 = y0
	}
	m.Y = clampf(m.Y, y0, y1)
	for _, w := range zoneWallRects(m.Zone) {
		m.X, m.Y = pushCircleOutOfRect(m.X, m.Y, r, w)
	}
	// Карман портала здесь НЕ выталкивает: заагренный моб может быть в кармане
	// (выталкивание делает updateMob каждый тик, но только вне погони).
}

// makeGroundPetal создаёт петаль-предмет на земле со случайным видом и редкостью.
func makeGroundPetal(zone int, id int) GroundPetal {
	kname := petalKindsPool[rand.Intn(len(petalKindsPool))]
	kind := petalKinds[kname]
	rar := pickRarity(zone)
	mult := petalPower[rar]
	// Размер петали: каждая редкость в 1.2 раза больше предыдущей.
	size := petalSizeBase * math.Pow(1.2, float64(rar))
	color := groundPetalColor(rar)

	z := zones[zone]
	return GroundPetal{
		ID:     "gp_" + strconv.Itoa(id),
		Rarity: rar,
		X:      z.X + 40 + rand.Float64()*(z.W-80),
		Y:      z.Y + 40 + rand.Float64()*(z.H-80),
		Size:   size,
		HP:     kind.BaseHP * mult,
		MaxHP:  kind.BaseHP * mult,
		DMG:    kind.BaseDMG * mult,
		Mass:   1 + float64(rar)*0.5, // лёгкая: мобы расталкивают её почти целиком
		Color:  color,
		Zone:   zone,
		Kind:   kname,
	}
}

// makeDropPetal создаёт петаль-дроп с убитого моба. Принадлежит одному игроку
// (OwnerID): видеть и подобрать его может только владелец. Вид петеля задаётся
// типом моба (см. mobDropKind), редкость — по дроп-таблице моба, позиция — возле
// места смерти.
func makeDropPetal(m *Mob, ownerID string, id int) GroundPetal {
	kname := mobDropKind[m.Type]
	if kname == "" {
		kname = "oval"
	}
	// Редкий дроп (см. mobRareDrop): шанс ПОДМЕНИТЬ обычный вид на более ценный.
	if rd, ok := mobRareDrop[m.Type]; ok && rand.Float64() < rd.Chance {
		kname = rd.Kind
	}
	kind := petalKinds[kname]
	rar := pickDropRarity(m.Rarity)
	mult := petalPower[rar]
	size := petalSizeBase * math.Pow(1.2, float64(rar))
	color := groundPetalColor(rar)
	return GroundPetal{
		ID:      "drop_" + strconv.Itoa(id),
		Rarity:  rar,
		X:       m.X + rand.Float64()*30 - 15,
		Y:       m.Y + rand.Float64()*30 - 15,
		Size:    size,
		HP:      kind.BaseHP * mult,
		MaxHP:   kind.BaseHP * mult,
		DMG:     kind.BaseDMG * mult,
		Mass:    1 + float64(rar)*0.5,
		Color:   color,
		Zone:    m.Zone,
		Kind:    kname,
		OwnerID: ownerID,
		Expire:  groundDropLifetime,
	}
}

// groundPetalColor - цвет петали в зависимости от редкости.
// Point, Line, Triangle, Square, Polygon, Hexagon, Fractal, Chaos.
func groundPetalColor(rarity int) uint32 {
	cs := []uint32{0xecf0f1, 0x2ecc71, 0xf1c40f, 0xe74c3c, 0x3498db, 0x1abc9c, 0x9b59b6, 0xff2d95}
	if rarity < len(cs) {
		return cs[rarity]
	}
	return 0xffffff
}

// updateMob обновляет ИИ моба: патруль или погоня за игроком.
func (r *Room) updateMob(m *Mob, players map[string]*Player, dt float64) {
	// Витринные мокапы TestRoom полностью статичны: никакого ИИ, движения,
	// агрессии. Это декорации, а не мобы.
	if m.Showcase {
		return
	}
	z := zones[m.Zone]
	zoneBottom := mobZoneBottom(&z) // ниже — Safe Zone: агро на игроков там отключено
	mt := mobTypes[m.Type]
	// Нейтральная зона у портала (карман): игрок внутри НЕ агрится на себя,
	// пока сам не ударит моба (агро через урон/aggroTo) или не выйдет из кармана.
	pocket, hasPocket := portalPocketRect(m.Zone)
	inPocket := func(x, y float64) bool {
		return hasPocket && x > pocket.X && x < pocket.X+pocket.W && y > pocket.Y && y < pocket.Y+pocket.H
	}

	// Ищем ближайшего живого игрока в той же зоне.
	var closest *Player
	closestDist := math.MaxFloat64
	for _, p := range players {
		if p.State.Zone != m.Zone || p.Dead || p.transferT > 0 {
			continue
		}
		if p.Passive {
			continue // пассивный админ — мобы его не агрятся
		}
		if !mobAggrosPlayer(p.Team) {
			continue // команда 2 (на стороне мобов) — мобы её не агрятся
		}
		if p.State.Y > zoneBottom {
			continue // в Safe Zone мобы не видят/не агрятся
		}
		// Нейтральная зона: игрока в кармане не видим, ЕСЛИ он не бил этого моба.
		// (Если бил — m.Damage поднят, агро честное: «тронул моба».)
		if inPocket(p.State.X, p.State.Y) && m.Damage[p.State.ID] <= 0 {
			continue
		}
		// Нейтральный моб (Peaceful, напр. Prism): игнорирует игроков, пока
		// те сами его не ударили (тогда догоняет обидчика по Damage-карте).
		if mt.Peaceful && m.Damage[p.State.ID] <= 0 {
			continue
		}
		dx := p.State.X - m.X
		dy := p.State.Y - m.Y
		d := math.Sqrt(dx*dx + dy*dy)
		if d < closestDist {
			closestDist = d
			closest = p
		}
	}

	// Цель групповой агрессии: если кого-то из отряда ударили, все ближайшие
	// мобы группы преследуют обидчика, игнорируя радиус агро.
	target := closest
	targetDist := closestDist
	if m.aggroTo != "" {
		a := players[m.aggroTo]
		if a != nil && a.State.Zone == m.Zone && !a.Dead && a.transferT <= 0 && !a.Passive &&
			a.State.Y <= zoneBottom {
			dx := a.State.X - m.X
			dy := a.State.Y - m.Y
			d := math.Sqrt(dx*dx + dy*dy)
			if d < targetDist {
				target = a
				targetDist = d
			}
		} else {
			m.aggroTo = ""
			m.aggroT = 0
		}
	}

	// Радиус агрессии растёт с редкостью: Chaos видит в rarityAggro[7]=6 раз
	// дальше базового диапазона (иначе огромного Chaos просто нельзя заметить).
	aggRange := mt.AggroRange
	if m.Rarity >= 0 && m.Rarity < len(rarityAggro) {
		aggRange *= rarityAggro[m.Rarity]
	}

	if (aggRange > 0 && target != nil && targetDist < aggRange) || (m.aggroTo != "" && target != nil) {
		m.state = "chase"
		dx := target.State.X - m.X
		dy := target.State.Y - m.Y
		ln := math.Sqrt(dx*dx + dy*dy)
		if ln > 0 {
			m.dirX = dx / ln
			m.dirY = dy / ln
		}
		m.moveTimer = 9999
	} else {
		// Счётчик групповой агрессии убывает только вне погони.
		if m.aggroT > 0 {
			m.aggroT -= dt
			if m.aggroT <= 0 {
				m.aggroTo = ""
			}
		}
		if m.state != "patrol" {
			m.state = "patrol"
			m.moveTimer = 1 + rand.Float64()*3
			m.dirX = rand.Float64()*2 - 1
			m.dirY = rand.Float64()*2 - 1
		}
		m.moveTimer -= dt
		if m.moveTimer <= 0 {
			if rand.Float64() < 0.3 {
				m.dirX = 0
				m.dirY = 0
				m.state = "idle"
				m.moveTimer = 0.5 + rand.Float64()*2
			} else {
				m.dirX = rand.Float64()*2 - 1
				m.dirY = rand.Float64()*2 - 1
				m.moveTimer = 1 + rand.Float64()*3
			}
		}
	}

	// Beam: периодически стреляет энергетическим шаром в текущую цель (только
	// когда моб агрится — шар сам «доводится» до ближайшего игрока).
	if m.Type == "beam" {
		if m.BeamT > 0 {
			m.BeamT -= dt
		}
		if m.BeamT <= 0 && m.state == "chase" && target != nil {
			r.spawnBeamOrb(m, target)
			m.BeamT = beamShootBase * (0.8 + rand.Float64()*0.4)
		}
	}

	// Компания Shield+Pentagon: во время патруля каждый моб группы (shield и
	// pentagon) следует за «вожаком» — ближайшим соседом той же группы с
	// наименьшим индексом. Ведомый держит строй: идёт в точку чуть позади
	// вожака, чтобы отряд двигался одной компанией, а не складывался в кучу.
	if (m.Type == "shield" || m.Type == "pentagon") && m.state != "chase" {
		lead := -1
		for i := range r.mobs {
			o := &r.mobs[i]
			if o == m || o.HP <= 0 || o.Zone != m.Zone {
				continue
			}
			if o.Type != "shield" && o.Type != "pentagon" {
				continue
			}
			dx := o.X - m.X
			dy := o.Y - m.Y
			if dx*dx+dy*dy < 900*900 && (lead < 0 || i < lead) {
				lead = i
			}
		}
		if lead >= 0 {
			l := &r.mobs[lead]
			// Точка позади вожака (по его направлению движения).
			tx := l.X - l.dirX*55
			ty := l.Y - l.dirY*55
			dx := tx - m.X
			dy := ty - m.Y
			dd := math.Sqrt(dx*dx + dy*dy)
			if dd > 0.5 {
				m.dirX = dx / dd
				m.dirY = dy / dd
			}
			m.moveTimer = 1.0 + rand.Float64()*0.8
		}
	}

	// FragWall — «стена»: движется строго по осям (горизонталь/вертикаль),
	// чтобы визуально не крутилась в разные стороны как стрелка.
	if m.Type == "fragwall" && m.Speed > 0 && (m.dirX != 0 || m.dirY != 0) {
		if math.Abs(m.dirX) >= math.Abs(m.dirY) {
			m.dirX = math.Copysign(1, m.dirX)
			m.dirY = 0
		} else {
			m.dirX = 0
			m.dirY = math.Copysign(1, m.dirY)
		}
	}

	// Эффективная скорость: у comet/arrow своя фазовая скорость (рывок/цикл разгона).
	eff := m.Speed
	if m.Type == "comet" {
		eff = cometPhaseSpeed(m, dt)
	} else if m.Type == "arrow" {
		eff = arrowPhaseSpeed(m, dt)
	}
	// Замедление (Rhombus): пока SlowT > 0 моб движется на SlowFrac медленнее.
	if m.SlowT > 0 {
		eff *= 1 - m.SlowFrac
	}
	if m.dirX != 0 || m.dirY != 0 {
		ln := math.Sqrt(m.dirX*m.dirX + m.dirY*m.dirY)
		m.X += (m.dirX / ln) * eff * dt
		m.Y += (m.dirY / ln) * eff * dt
	}

	// Клампы зоны + скольжение: составляющую скорости, давящую В стену, гасим,
	// тангенциальную оставляем — моб скользит вдоль стены к цели, не вибрирует.
	bx, by := m.dirX, m.dirY
	if m.X < z.X+m.Radius {
		m.X = z.X + m.Radius
		if bx < 0 {
			bx = 0
		}
	}
	if m.X > z.X+z.W-m.Radius {
		m.X = z.X + z.W - m.Radius
		if bx > 0 {
			bx = 0
		}
	}
	if m.Y < z.Y+m.Radius {
		m.Y = z.Y + m.Radius
		if by < 0 {
			by = 0
		}
	}
	// Мобы не заходят в стены и Safe Zone: нижняя граница — верх стены.
	bot := mobZoneBottom(&z) - m.Radius
	if m.Y > bot {
		m.Y = bot
		if by > 0 {
			by = 0
		}
	}
	// Скруглённые углы (зоны II и верх L1): центр моба остаётся на дуге касания
	// (crs[i] - радиус, т.е. ВНУТРИ зоны у стены), а радиальную составляющую
	// скорости гасим — моб плавно скользит вдоль дуги.
	for i := 0; i < 4; i++ {
		if z.crs[i] <= m.Radius {
			continue
		}
		cr := z.crs[i] - m.Radius
		if 2*z.crs[i] >= z.W || 2*z.crs[i] >= z.H {
			continue
		}
		cx := z.cxs[i]
		cy := z.cys[i]
		dx := m.X - cx
		dy := m.Y - cy
		if z.sxs[i]*dx <= 0 || z.sys[i]*dy <= 0 {
			continue
		}
		d := math.Sqrt(dx*dx + dy*dy)
		if d <= cr {
			continue
		}
		if d < 1e-9 {
			d = 1e-9
		}
		m.X = cx + dx/d*cr
		m.Y = cy + dy/d*cr
		// Нормаль дуги = радиальное направление от центра угла наружу.
		nx := (m.X - cx) / cr
		ny := (m.Y - cy) / cr
		rad := bx*nx + by*ny
		if rad > 0 {
			bx -= nx * rad
			by -= ny * rad
		}
	}
	// Кастомные стены (walls.json): выталкиваем КАЖДЫЙ тик и гасим нормальную
	// составляющую скорости — иначе моб дёргался на выталкиваниях и застревал.
	for _, w := range zoneWallRects(m.Zone) {
		px, py := m.X, m.Y
		m.X, m.Y = pushCircleOutOfRect(m.X, m.Y, m.Radius, w)
		if m.X != px || m.Y != py {
			nx, ny := m.X-px, m.Y-py
			ln := math.Sqrt(nx*nx + ny*ny)
			if ln > 0 {
				nx, ny = nx/ln, ny/ln
				dot := bx*nx + by*ny
				if dot < 0 {
					bx -= nx * dot
					by -= ny * dot
				}
			}
		}
	}
	// Карман безопасности вокруг портала: мобы не могут входить телом, НО
	// заагренный (в погоне за игроком) моб МОЖЕТ заезжать в карман и на портал.
	// Когда погоня кончается (игрок умер/ушёл/вышел из радиуса) — моб снова
	// выталкивается и выбегает из кармана.
	if m.state != "chase" {
		if pr, ok := portalPocketRect(m.Zone); ok {
			px, py := m.X, m.Y
			m.X, m.Y = pushCircleOutOfRect(m.X, m.Y, m.Radius, pr)
			if m.X != px || m.Y != py {
				nx, ny := m.X-px, m.Y-py
				ln := math.Sqrt(nx*nx + ny*ny)
				if ln > 0 {
					nx, ny = nx/ln, ny/ln
					dot := bx*nx + by*ny
					if dot < 0 {
						bx -= nx * dot
						by -= ny * dot
					}
				}
			}
		}
	}
	// ФИНАЛЬНЫЙ кламп зоны: никакое выталкивание (стены, карман портала) не
	// может выбросить моба за границы зоны (со скруглёнными углами — тоже).
	// Раньше после смерти игрока моба, заехавшего в карман в погоне, выталкивало
	// за нижний край карты — и это повторялось каждый тик (кламп втягивал,
	// карман выталкивал).
	m.X, m.Y = clampZoneRounded(&z, m.X, m.Y, m.Radius)
	m.dirX, m.dirY = bx, by

	if m.hitCD > 0 {
		m.hitCD -= dt
	}
	// Орбитальные петали боссов: вращение (1 круг за 5 с) и кулдауны атак.
	for i := range m.petals {
		bp := &m.petals[i]
		bp.Angle += mobPetalAngSpeed * dt
		if bp.Respawn > 0 {
			bp.Respawn -= dt
			if bp.Respawn <= 0 {
				bp.Respawn = 0
				bp.HP = bp.MaxHP
				bp.CD = 0
			}
		} else if bp.CD > 0 {
			bp.CD -= dt
		}
	}
}

// ---- Спецспособности мобов: Comet (рывок) и Arrow (ускорение) ----

// phaseVariance — случайная (но устойчиво похожая) вариация длительностей фаз:
// «случайные по времени, не сильно отличающиеся» — ±20%.
func phaseVariance() float64 {
	return 0.8 + rand.Float64()*0.4
}

// cometPhaseSpeed — эффективная скорость кометы в этом тике. Фазы:
// 0 полёт (обычная скорость) → 1 заряд (плавное торможение Speed→0) →
// 2 рывок (Speed×cometDashMult по зафиксированному направлению) →
// 3 отдых (стоит) → снова полёт. Платит за прыжки рывков случайностью фаз.
func cometPhaseSpeed(m *Mob, dt float64) float64 {
	m.cometT -= dt
	switch m.cometPhase {
	case 1: // заряд: плавно до полной остановки
		if m.cometT <= 0 {
			m.cometPhase = 2
			m.cometDur = cometDashDur * phaseVariance()
			m.cometT = m.cometDur
			// Рывок по направлению движения в момент остановки (в погоне — на игрока).
			ln := math.Sqrt(m.dirX*m.dirX + m.dirY*m.dirY)
			if ln > 0 {
				m.cometDX, m.cometDY = m.dirX/ln, m.dirY/ln
			} else {
				m.cometDX, m.cometDY = 1, 0
			}
			return m.Speed * cometDashMult // рывок начинается тут же
		}
		return m.Speed * clampf(m.cometT/m.cometDur, 0, 1)
	case 2: // рывок: направление зафиксировано, скорость ×dashMult (+ редкость)
		m.dirX, m.dirY = m.cometDX, m.cometDY
		if m.cometT <= 0 {
			m.cometPhase = 3
			m.cometDur = cometRestDur * phaseVariance()
			m.cometT = m.cometDur
			return 0
		}
		return m.Speed * cometDashSpeedMult(m)
	case 3: // отдых после рывка
		if m.cometT <= 0 {
			m.cometPhase = 0
			m.cometT = cometWaitDur * phaseVariance()
		}
		return 0
	default: // обычный полёт до следующей зарядки
		if m.cometT <= 0 {
			m.cometPhase = 1
			m.cometDur = cometChargeDur * phaseVariance()
			m.cometT = m.cometDur
		}
		return m.Speed
	}
}

// cometDashSpeedMult — множитель скорости рывка кометы с учётом редкости:
// cometDashMult × (1 + cometDashPerRar×редкость) — редкие бьют интереснее.
func cometDashSpeedMult(m *Mob) float64 {
	rar := clampf(float64(m.Rarity), 0, 7)
	return cometDashMult * (1 + cometDashPerRar*rar)
}

// arrowPhaseSpeed — скорость стрелы в цикле: разгон (0→Speed) → крейсер →
// торможение (Speed→0) → полная остановка на короткое время. Фазы слегка
// случайны, но не сильно отличаются (±20%).
func arrowPhaseSpeed(m *Mob, dt float64) float64 {
	m.arrowT -= dt
	next := func(newPhase int, baseDur float64) {
		m.arrowPhase = newPhase
		m.arrowDur = baseDur * phaseVariance()
		m.arrowT = m.arrowDur
	}
	switch m.arrowPhase {
	case 1: // крейсер: спринт на тройной скорости
		if m.arrowT <= 0 {
			next(2, arrowBrakeDur)
		}
		return m.Speed * arrowSprintMult
	case 2: // торможение: плавно до нуля от спринт-скорости
		if m.arrowT <= 0 {
			next(3, arrowRestDur)
			return 0
		}
		return m.Speed * arrowSprintMult * clampf(m.arrowT/m.arrowDur, 0, 1)
	case 3: // отдых: стоит на месте
		if m.arrowT <= 0 {
			next(0, arrowAccelDur)
		}
		return 0
	default: // разгон: по чуть-чуть КАК БЫК к игроку (×3)
		if m.arrowT <= 0 {
			next(1, arrowCruiseDur)
			return m.Speed * arrowSprintMult // старт крейсера сразу на максимуме
		}
		return m.Speed * arrowSprintMult * clampf(1-m.arrowT/m.arrowDur, 0, 1)
	}
}

// packRadius — радиус, в котором мобы группы считаются «своей стаей».
const packRadius = 900.0

// groupMobType — моб входит в стаю (ходит компанией) и агрится всей стаей.
func groupMobType(t string) bool {
	return t == "pentagon" || t == "shield"
}

// aggroPack берёт моба стаи (Pentagon/Shield), которого ударил игрок, и
// натравливает всю ближайшую стаю (всех живых Pentagon/Shield в зоне, в
// радиусе packRadius) на этого игрока на некоторое время.
func (r *Room) aggroPack(m *Mob, attackerID string) {
	if !groupMobType(m.Type) || attackerID == "" {
		return
	}
	for i := range r.mobs {
		o := &r.mobs[i]
		if o == m || o.HP <= 0 || o.Zone != m.Zone || !groupMobType(o.Type) {
			continue
		}
		dx := o.X - m.X
		dy := o.Y - m.Y
		if dx*dx+dy*dy > packRadius*packRadius {
			continue
		}
		o.aggroTo = attackerID
		o.aggroT = 15.0
	}
}
