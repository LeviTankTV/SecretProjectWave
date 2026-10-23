package worlddata

import (
	"math/rand"
)

// mob_drops.go — данные дропа мобов: вид основной петели, редкий дроп,
// оверрайды из админ-панели и таблица качелей по редкостям. Разрезано
// из mob_data.go в 3.3.

// Каждый тип моба при смерти "дропает" конкретный вид петеля.
var MobDropKind = map[string]string{
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
	"fragtank":  "fragwall",
	"fragwall": "fragwall", // Wave-уникальный моб (в родителе заменён на fragtank)
	"void":      "void",
	"entropy":   "entropy",
	// Витринные мокапы → реальные мобы
	"beads":     "orb",
	"tear":      "tear",
	"comet":     "volt",      // Comet теперь дропает новый дпс-петаль Volt
	"zigzag":    "jag",       // Zigzag (Triangle) дропает Jag
	"quarry":    "anvil",     // Quarry (Square) дропает Anvil
	"arrowhead": "piercer",   // Arrowhead (Triangle) дропает Piercer
	"shardwing": "wingblade", // Shardwing (Triangle) дропает Wingblade
	"frame":     "rivet",     // Frame (Square) дропает Rivet
	"kite":      "tail",      // Kite (Square) дропает Tail
	"beam":      "spark",
	"spike":     "spike",
	"cross":     "bulwark",
	"star":      "spark",
	"nova":      "spark", // Новая синяя звезда: искры-звёздочки (как у Star)
	"mirror":    "twirl", // Зеркало: Twirl — «разворот» орбиты, зеркальная петаль
	"trinity":   "tear",  // Тройной Point-моб: Tear (лечит при разрушении — «держатся до последнего»)
	"aeon":      "hex",
	"rune":      "rune",
	"cog":       "cog",
	"sun":       "compass",
	"mote":      "fractalite",
	"saturn":    "saturn",
	"eye":       "void",
	"galaxy":    "void",
	"blackhole": "singularity",
	"prism":     "rhombus", // Prism дропает петаль Rhombus (замедление)
	// Древние странники «Origin of Point»: Bud — orb (реген-петаль),
	// Marble — pebble (сшинг с отбросом). Числа тюнятся через админку.
	"bud":    "orb",
	"marble": "pebble",
	// Стражи «Origin of Line»: копьё → Piercer, метатель → Volt,
	// дротик → Shard, шипастое копьё → Spike.
	"lancer":     "piercer",
	"javelin":    "volt",
	"darter":     "shard",
	"spikelance": "spike",
}

// MobRareDrop — РЕДКИЙ дроп: с малым шансом вместо обычного вида (MobDropKind)
// у ДАННОГО игрока выпадает более ценный вид. Роллится per-player в
// makeDropPetal, потому оба убийцы топ-4 могут получить один и тот же редкий.
var MobRareDrop = map[string]RareDropDef{
	"beads": {Kind: "line", Chance: 0.10},  // Линия: 10% вместо обычного orb
	"eye":   {Kind: "eye", Chance: 0.01},   // Глаз: 1% вместо обычного void
	"rune":  {Kind: "twirl", Chance: 0.05}, // Вертушка: 5% вместо обычного rune
}

// MobDropOverride — СВОЙ дроп одного типа моба из админ-панели (config.json
// → drop.mobDrops). Пустые поля означают «брать глобальное значение»:
// Kind пустой — основной вид из MobDropKind, RareKind пустой — редкой подмены
// нет, RarityW пустой — таблица редкостей из глобальной MobRarityDrop.
// Custom — список КАСТОМНЫХ дропов с произвольным шансом (15%, 0.5% — какой
// задаст админ): вид + фиксированная редкость + шанс + количество.
type MobDropOverride struct {
	Kind     string
	RareKind string
	RarePct  float64
	RarityW  [][]int
	Custom   []CustomDropDef
}

// CustomDropDef — один кастомный дроп моба.
type CustomDropDef struct {
	Kind   string  `json:"kind"`   // вид петали (обязателен, из PetalKinds)
	Rarity int     `json:"rarity"` // фиксированная редкость дропа 0..7
	Pct    float64 `json:"pct"`    // 0..1 — шанс (0.15 = 15%)
	Count  int     `json:"count"`  // сколько петалей падает при успехе (1..10)
	PerMob bool    `json:"perMob"` // true — один ролл на моба (достаётся всем квалифицированным), false — свой ролл каждому дамагеру
}

// CustomDropsForMob — кастомные дропы моба из панели (без записи — пусто).
func CustomDropsForMob(mobType string) []CustomDropDef {
	if o, ok := MobDropOverrides[mobType]; ok {
		return o.Custom
	}
	return nil
}

// MobDropOverrides — живые оверрайды дропов мобов (пишет applyConfig).
var MobDropOverrides = map[string]MobDropOverride{}

// DropKindForMob — основной вид петали-дропа моба с учётом оверрайда панели.
func DropKindForMob(mobType string) string {
	if o, ok := MobDropOverrides[mobType]; ok && o.Kind != "" {
		return o.Kind
	}
	if dk := MobDropKind[mobType]; dk != "" {
		return dk
	}
	return "oval"
}

// RareDropForMob — редкая подмена дропа моба с учётом оверрайда панели
// (ok=false — подмены нет).
func RareDropForMob(mobType string) (string, float64, bool) {
	if o, ok := MobDropOverrides[mobType]; ok && o.RareKind != "" && o.RarePct > 0 {
		return o.RareKind, o.RarePct, true
	}
	rd, ok := MobRareDrop[mobType]
	if ok {
		return rd.Kind, rd.Chance, true
	}
	return "", 0, false
}

// DropRarityForMob выбирает редкость дропа моба: СВОЯ таблица из панели
// (если настроена), иначе глобальная MobRarityDrop. Возврат — редкость 0..7.
// Таблица панели — 8 строк (редкости 8..15 переиспользуют парные ступени по r%8).
func DropRarityForMob(mobType string, mobRarity int) int {
	if o, ok := MobDropOverrides[mobType]; ok && len(o.RarityW) > 0 {
		idx := max(mobRarity, 0)
		if idx >= len(o.RarityW) {
			idx %= len(o.RarityW)
		}
		row := o.RarityW[idx]
		total := 0
		for _, v := range row {
			total += v
		}
		if total > 0 {
			r := rand.Intn(total)
			for i, v := range row {
				r -= v
				if r < 0 {
					return i
				}
			}
			return 0
		}
		// Пустая строка (все нули) — падение на глобальную таблицу.
	}
	return PickDropRarity(mobRarity)
}

// RareDropDef описывает редкий дроп типа моба.
type RareDropDef struct {
	Kind   string
	Chance float64 // 0..1 — вероятность ПОДМЕНЫ обычного вида
}

// BlackHoleDropChance — шанс дропа Singularity с Black Hole (per-mob roll,
// один бросок на убийство, затем все топ-4 дамагеры получают петаль).
var BlackHoleDropChance = 0.2

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
var MobRarityDrop = [][]int{
	{1000, 0, 0, 0, 0, 0, 0, 0},    // Point моб: только свой тир (100%)
	{780, 220, 0, 0, 0, 0, 0, 0},   // Line моб: 78 Point / 22 Line
	{345, 580, 75, 0, 0, 0, 0, 0},  // Triangle моб: 9/58/33
	{20, 706, 250, 24, 0, 0, 0, 0}, // Square моб: 2/70.6%/25%/2.4%
	// Polygon моб (ребаланс): свой Polygon дроп порезан до ~1.2%;
	// освободившаяся масса ушла в Square/Triangle/Line.
	{0, 16, 148, 824, 12, 0, 0, 0},
	// Hexagon моб (ребаланс): центр Square (~80%), Polygon ~16%, СВОЙ
	// Hexagon — 1.67%.
	{0, 0, 13, 800, 160, 16, 0, 0},
	// Fractal моб (ребаланс): Hexagon = 16.7% (1/6), Fractal 0.2%, центр Polygon 74%.
	{0, 0, 8, 83, 740, 167, 2, 0},
	{0, 0, 0, 0, 250, 730, 20, 0}, // Chaos моб: 25 Polygon/73 Hexagon/2% Fractal, никогда Chaos
	// Истинная лестница (8..15): дроп-правила копируют парные ступени
	// (Dot как Point, Vector как Line, … Origin как Chaos).
	{1000, 0, 0, 0, 0, 0, 0, 0},
	{780, 220, 0, 0, 0, 0, 0, 0},
	{345, 580, 75, 0, 0, 0, 0, 0},
	{20, 706, 250, 24, 0, 0, 0, 0},
	{0, 16, 148, 824, 12, 0, 0, 0},
	{0, 0, 13, 800, 160, 16, 0, 0},
	{0, 0, 8, 83, 740, 167, 2, 0},
	{0, 0, 0, 0, 250, 730, 20, 0},
}

// --- сеттеры: эти значения меняет конфиг мира, а не рантайм ---

// SetMobDropOverrides заменяет таблицу админ-оверрайдов дропа целиком.
func SetMobDropOverrides(m map[string]MobDropOverride) { MobDropOverrides = m }

// SetBlackHoleDropChance задаёт шанс дропа из чёрной дыры (конфиг мира).
func SetBlackHoleDropChance(v float64) { BlackHoleDropChance = v }

// PickDropRarity выбирает редкость дропа по весам таблицы MobRarityDrop.
// Живёт здесь, а не в рантайме: зависит только от данных.
func PickDropRarity(mobRarity int) int {
	w := MobRarityDrop[mobRarity]
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
