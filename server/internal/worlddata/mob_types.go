package worlddata

// mob_types.go — данные типов мобов и предмета-петеля: структура MobType,
// таблица MobTypes и структура PetalDef. Разрезано из mob_types.go и
// mob_data.go в 3.3 (срез C), содержимое перенесено байт в байт.
//
// Почему данные, а не рантайм: таблицу типов мобов ЧИТАЮТ почти все игровые
// подсистемы (бой, спавн, снапшот, галерея, редактор), а вот ЗАПИСЬ в неё идёт
// из четырёх мест рантайма — туториальные комнаты (tutdummy/tutspot/blob),
// квестодатель холла Origin (halldot), витрина TestRoom (show1..showN) и правки
// редактора из config.json. Поэтому запись здесь одна — RegisterMob, а не
// шесть прямых присваиваний по ключу в разных файлах.
//
// Типы переехали АЛИАСАМИ (type MobType = worlddata.MobType и так же PetalDef):
// это не правит ни одного места вызова, но убирает определения из рантайма.
//
// Что осталось в рантайме намеренно: сам Mob/Petal/MobState (у них unexported
// поля и методы боя) и всё, что требует *Mob.

// MobType — базовые характеристики вида моба (до умножителей редкости).
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

	// --- Потенциал (Pot.md §17) ---
	// Pot — множитель HP/DMG, 100 = база; Eff — множитель пассивок вида.
	// Оба только из крафта и только серверные: клиент их отображает, но не
	// присылает. 0 в старых сохранениях читается как 100 (normPot/normEff).
	Pot       int    `json:"pot,omitempty"`
	Eff       int    `json:"eff,omitempty"`
	UID       string `json:"uid,omitempty"`       // стабильный ID уникального предмета
	CraftedBy string `json:"craftedBy,omitempty"` // ник автора (провенанс)
	CraftedAt int64  `json:"craftedAt,omitempty"` // unix-время крафта
}

var MobTypes = map[string]MobType{
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
	"fragtank": {
		Name: "FragTank", Color: 0x6c3483,
		BaseHP: 55, BaseDMG: 5, BaseSpeed: 70, BaseRadius: 30,
		AggroRange: 700, Shape: "tank", // танк видит дальше обычных мобов
	},
	// Wave-уникальный моб (в родителе заменён на fragtank): стена-фрактал с
	// осевым движением. Оставлена при порте worlddata — часть идентичности Wave.
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
	// Новая синяя звезда (Pentagon): единый «звёздный» дух, но с СВОИМ именем
	// (Star уже занят). Спавнится в 3 раза реже обычной Star (zoneSpawnWeights).
	// Дроп — Spark (искры-звёздочки, как у Star).
	"nova": {
		Name: "Nova", Color: 0x2e86c1,
		BaseHP: 24, BaseDMG: 7, BaseSpeed: 60, BaseRadius: 24,
		AggroRange: 360, Shape: "mockup_68",
	},
	// Mirror (Pentagon): пятиугольное зеркало с бегущим бликом по ободу.
	// Очень редкий спавн. Дроп — Twirl (зеркальная петаль: разворот орбиты).
	"mirror": {
		Name: "Mirror", Color: 0x76d7ea,
		BaseHP: 26, BaseDMG: 7, BaseSpeed: 55, BaseRadius: 24,
		AggroRange: 360, Shape: "mockup_70",
	},
	// Trinity (Point): тройной моб — ЕДИНЫЙ организм из 3 скреплённых мобов.
	// Каждый из трёх кружков — отдельный Mob со своим HP/дропом; пока жив хоть
	// один — группа продолжает ехать на игрока (см. updateMob, spawnMob).
	// Каждая единица отрисовывается одиночным кружком (mockup_111), дроп — Tear.
	"trinity": {
		Name: "Trinity", Color: 0x2ecc71,
		BaseHP: 14, BaseDMG: 4, BaseSpeed: 72, BaseRadius: 16,
		AggroRange: 300, Shape: "mockup_111",
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
	// Nexus (17, Shape Player): моб-«игрок» зоны Nexus — андроид с орбитальными
	// петалыми, уровнем по редкости (Lv.1..100), бегством/регеном. HP и DMG
	// берутся не из этой записи, а по Block/Black Hole той же редкости
	// (см. makeNexusMob / mobStatsForRarity).
	"nexus": {
		Name: "Nexus", Color: 0xffffff,
		BaseHP: 100, BaseDMG: 8, BaseSpeed: 75, BaseRadius: 22,
		AggroRange: 380, Shape: "nexus",
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
	// === Витрина: 15 концептов зоны Triangle (17-31, оранжево-огненные) ===
	// mockup17-20: финалисты витрины — РЕАЛЬНЫЕ мобы (Arrowhead/Shardwing в
	// Triangle, Frame/Kite в Square); их формы — дизайн-мокапы 20/39/26/21.
	"mockup17": {Name: "Arrowhead", Color: 0xf4511e, BaseHP: 1, BaseDMG: 0, BaseSpeed: 0, BaseRadius: 42, AggroRange: 0, Shape: "mockup_20"},
	"mockup18": {Name: "Frame", Color: 0xb93232, BaseHP: 1, BaseDMG: 0, BaseSpeed: 0, BaseRadius: 42, AggroRange: 0, Shape: "mockup_39"},
	"mockup19": {Name: "Kite", Color: 0xffb74d, BaseHP: 1, BaseDMG: 0, BaseSpeed: 0, BaseRadius: 42, AggroRange: 0, Shape: "mockup_26"},
	"mockup20": {Name: "Shardwing", Color: 0xfb8c00, BaseHP: 1, BaseDMG: 0, BaseSpeed: 0, BaseRadius: 42, AggroRange: 0, Shape: "mockup_21"},
	// exp1-3: витрина TestRoom (слоты №21-23) — отобранные концепты:
	// Trinity (тройной моб-Point, mockup_63), Nova (новая синяя звезда
	// Pentagon, mockup_68) и Mirror (зеркало Pentagon, mockup_70).
	// Слоты 24-77 — витринные дизайны: типы show24..show77
	// создаются автоматически в showcase_slots.go (имена берутся из реестра),
	// поэтому ниже их НЕТ — сюда вписывать нечего.
	"exp1": {Name: "Trinity", Color: 0x2ecc71, BaseHP: 1, BaseDMG: 0, BaseSpeed: 0, BaseRadius: 42, AggroRange: 0, Shape: "mockup_63"},
	"exp2": {Name: "Nova", Color: 0x2e86c1, BaseHP: 1, BaseDMG: 0, BaseSpeed: 0, BaseRadius: 42, AggroRange: 0, Shape: "mockup_68"},
	"exp3": {Name: "Mirror", Color: 0x76d7ea, BaseHP: 1, BaseDMG: 0, BaseSpeed: 0, BaseRadius: 42, AggroRange: 0, Shape: "mockup_70"},
	// === Новые мобы зон ===
	"arrowhead": {
		Name: "Arrowhead", Color: 0xf4511e,
		// Наконечник (Triangle): клин-копьё, бьёт на проходе. Дроп — Piercer.
		BaseHP: 12, BaseDMG: 7, BaseSpeed: 60, BaseRadius: 20,
		AggroRange: 300, Shape: "mockup_20",
	},
	"shardwing": {
		Name: "Shardwing", Color: 0xfb8c00,
		// Осколокрыл (Triangle): машущий крылатый осколок. Дроп — Wingblade.
		BaseHP: 9, BaseDMG: 8, BaseSpeed: 75, BaseRadius: 17,
		AggroRange: 320, Shape: "mockup_21",
	},
	"frame": {
		Name: "Frame", Color: 0xb93232,
		// Рама (Square): бронированная квадратная рама-обруч. Дроп — Rivet.
		BaseHP: 30, BaseDMG: 5, BaseSpeed: 35, BaseRadius: 26,
		AggroRange: 260, Shape: "mockup_39",
	},
	"kite": {
		Name: "Kite", Color: 0xffb74d,
		// Змей (Square): воздушный змей с толкающим ударом. Дроп — Tail.
		BaseHP: 14, BaseDMG: 6, BaseSpeed: 70, BaseRadius: 19,
		AggroRange: 300, Shape: "mockup_26",
	},
	"zigzag": {
		Name: "Zigzag", Color: 0xf4511e,
		// Зигзаг (Triangle): быстрый дротик-молния, рваные рывки по зигзагу.
		// Хрупкий, но бьёт больно; дропает петаль Jag.
		BaseHP: 10, BaseDMG: 8, BaseSpeed: 115, BaseRadius: 15,
		AggroRange: 330, Shape: "zigzag",
	},
	"quarry": {
		Name: "Quarry", Color: 0xb03a2e,
		// Карьерник (Square): медленная каменная глыба-добытчик, очень живучая.
		// Дропает петаль Anvil.
		BaseHP: 45, BaseDMG: 6, BaseSpeed: 22, BaseRadius: 30,
		AggroRange: 280, Shape: "quarry",
	},
	// === Древние странники «Origin of Point» (зона 44) ===
	// Редкие гости среди обычных Point-мобов, спавнятся с редкостью Dot (8).
	// Формы — витринные дизайны mockup_24 (Bud) и mockup_25 (Marble).
	"bud": {
		Name: "Bud", Color: 0x2ecc71,
		// Бутон: живой цветок-стражник. Раз в ~2.2с (в агре) выбрасывает
		// 8 лепестков-снарядов строго по 360° (см. mob_growth.go, bud).
		BaseHP: 5, BaseDMG: 2, BaseSpeed: 70, BaseRadius: 26,
		AggroRange: 300, Shape: "mockup_24",
	},
	"marble": {
		Name: "Marble", Color: 0x27ae60,
		// Мрамор: древняя сфера со спиралью. Движение — серпантин
		// (Sine -> Serpentine, movement_designs #3): виляет на подлёте.
		BaseHP: 6, BaseDMG: 2, BaseSpeed: 85, BaseRadius: 24,
		AggroRange: 320, Shape: "mockup_25",
	},
	// === Зона «Origin of Line» (45): копейщики тоннеля ===
	// Две пары стражей зигзага: в средней части тоннеля — Lancer/Javelin
	// редкости Point и Line, на вершине — Darter/SpikeLance редкости Vector.
	//
	// Формы — РОВНО витринные дизайны client/Designs/mob_designs.html (слоты
	// 26, 29, 32, 35). Все четыре нарисованы остриём в +X, а клиент поворачивает
	// моба по вектору движения (render.js → a.rot = atan2(dirY, dirX)), поэтому
	// в агра копьё/дротик/рыба смотрят острым концом на игрока. До этого здесь
	// стояли чужие формы (line/zigzag/dart/arrow): моб выглядел не как свой
	// дизайн, а «чем-то похожим», и остриё у части из них было сзади.
	"lancer": {
		Name: "Lancer", Color: 0xf1c40f,
		// Копейщик: держит дистанцию и бьёт длинным выпадом.
		BaseHP: 18, BaseDMG: 7, BaseSpeed: 82, BaseRadius: 20,
		AggroRange: 320, Shape: "mockup_26",
	},
	"javelin": {
		Name: "Javelin", Color: 0xf39c12,
		// Метатель: быстрый, хрупкий, заходит с фланга.
		BaseHP: 12, BaseDMG: 9, BaseSpeed: 124, BaseRadius: 15,
		AggroRange: 340, Shape: "mockup_29",
	},
	"darter": {
		Name: "Darter", Color: 0xf1c40f,
		// Дротик вершины: самый быстрый страж тоннеля.
		BaseHP: 14, BaseDMG: 10, BaseSpeed: 138, BaseRadius: 14,
		AggroRange: 360, Shape: "mockup_32",
	},
	"spikelance": {
		Name: "SpikeLance", Color: 0xf3c623,
		// Шипастое копьё: тяжёлый, медленный, но бьёт больно.
		BaseHP: 28, BaseDMG: 12, BaseSpeed: 58, BaseRadius: 24,
		AggroRange: 340, Shape: "mockup_35",
	},
}

// --- сеттеры: эти данные меняет рантайм (комнаты туториала, витрина, конфиг) ---

// RegisterMob регистрирует или переопределяет вид моба по ключу. Единственный
// путь записи в таблицу: раньше её писали по ключу из шести мест, и опечатка в
// ключе там же создавала моба-пустышку (нулевые Name/HP), которую не видел ни
// один сторож.
func RegisterMob(kind string, def MobType) {
	MobTypes[kind] = def
}
