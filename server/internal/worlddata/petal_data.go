package worlddata

import (
	"sort"
)

// petal_data.go — данные видов петалей: структура PetalKind, каталог PetalKinds,
// список настоящих видов (без админских passive) и зонные лимиты количества.
// Разрезано из mob_data.go в 3.3; HTTP-хендлер каталога — http_petals.go.

// mobTypesInZone / zoneSpawnWeights / zoneMobLimits / zoneRarityFloor /
// zoneRarityCeil УДАЛЕНЫ: базовый (band) спавн больше не существует. Типы,
// веса и лимиты мобов задаются ТОЛЬКО территориями спавна в walls.json
// (client/editor). См. NOTES.md (Ф2).

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
	Trap      bool // ловушка: захват врага с HP ≤ trapCaptureFrac — моб уходит в Коллекцию владельца (trap)
	Token     bool // жетон: сюжетный предмет (Ying/Yang/Ying-Yang). Soulbound, не участвует
	// в бою и не изнашивается, НО надевается на орбиту как трофей. Размер
	// фиксированный (tokenPetalSize) — ключ не должен расти с редкостью.

	MinionType string // тип моба-миньона ("entropy" / "blackhole"); если "" — не спавнит миньонов
	Passive    bool   // пассивный служебный петаль: неуничтожим, невидим, не участвует в бою и не занимает слот орбиты (админ-петаль «nexus»)
}

// Каталог видов петалей. Статы далее умножаются на редкость (rarityDanger).
// Базовые «стартовые» виды (oval/thorns/rock/ball) лежат в земле и на них же
// можно учиться крафтить; тематические (orb..entropy) дропаются с конкретных
// мобов (см. MobDropKind).
var PetalKinds = map[string]PetalKind{
	// Стартовые виды
	"oval": {
		Name: "Oval", Shape: "oval",
		// Лайфстил (механика «лепесток»): при ударе лечит игрока lifestealPct
		// от нанесённого урона — састийн в бою.
		BaseHP: 8.0, BaseDMG: 4.5,
		Lifesteal: true,
	},
	"thorns": {
		Name: "Thorns", Shape: "thorns",
		// Шипы («размен»): прямой урон высокий, HP низкое. За каждый износ
		// (моб бьёт шип) моб получает обратно thornsReflectRatio × урон износа.
		// Чем больнее бьёт моб — тем больше «сам колется» о шип.
		BaseHP: 5.0, BaseDMG: 7.5,
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
		BaseHP: 5.5, BaseDMG: 10.0,
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
		// На орбите — «стеклянная» DPS-петаль с частыми ударами (atkCD 0.05).
		// HP 8 (было 5): возросшей темп ударов не стал ломать клинок
		// мгновенно об износ (m.DMG × 0.3 за удар).
		BaseHP: 8.0, BaseDMG: 11.0,
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
		// Вспомогательная петаль: контактный урон сильно дебафнут.
		BaseHP: 9.0, BaseDMG: 2.0,
		Swift: true,
	},
	"shield": {
		Name: "Shield", Shape: "shield",
		// Щит (механика «барьер»): автоматически пополняет синюю полоску щита
		// игрока (броня перед HP). Каждый активный щит даёт ёмкость MaxShield и
		// периодически (shieldChargeInterval) заряжает её. Вспомогательная
		// петаль: контактного урона НЕ наносит.
		BaseHP: 13.0, BaseDMG: 0.0,
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
		BaseHP: 1.0, BaseDMG: 10.0,
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
		BaseHP: 4.67, BaseDMG: 8.5, // баф дпс
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
		// Вспомогательная петаль: контактного урона НЕ наносит.
		BaseHP: 11.0, BaseDMG: 0.0,
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
	"nexus": {
		Name: "Nexus", Shape: "nexus",
		// Спавн-петаль (админ-петаль «Nexus»): НЕВИДИМЫЙ, НЕУНИЧТОЖИМЫЙ,
		// ПАССИВНЫЙ служебный петаль — при надевании МГНОВЕННО призывает
		// ДРУЖЕЛЮБНОГО Nexus (см. minion.go, Passive => пропуск всех боевых
		// циклов через isOrbitAccessory). Редкость миньона = редкость петеля
		// (НЕ таблица summonSpec), размер фиксированный (nexusRadius), цвет
		// 0x000000, статы — как у вражеского Nexus той же редкости (Block HP /
		// Black Hole урон, без нерфов). Ровно 1 миньон на петаль.
		BaseHP: 10.0, BaseDMG: 0.0,
		MinionType: "nexus",
		Passive:    true,
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
		// всех петалей на eyeOrbitPerRar×редкость (r7 ≈ ×2.54), петали разлетаются
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
	// === Новые ДПС-петали ===
	"volt": {
		Name: "Volt", Shape: "spark",
		// Разряд (дроп с Comet): чистый дпс-петаль высокого урона — «электро-
		// клык» Линии. HP невелико: роль — бить, не танковать.
		BaseHP: 4.0, BaseDMG: 11.0,
	},
	"jag": {
		Name: "Jag", Shape: "thorns",
		// Зазубрина (дроп с Zigzag, Triangle): средний урон + живучесть скромная,
		// но частые касания за счёт малой орбиты... чистый дпс-петаль.
		BaseHP: 3.5, BaseDMG: 9.5,
	},
	"anvil": {
		Name: "Anvil", Shape: "rock",
		// Наковальня (дроп с Quarry, Square): тяжёлый дпс-танк — бьёт больно
		// и держит удар.
		BaseHP: 12.0, BaseDMG: 10.0,
	},
	"lance": {
		Name: "Lance", Shape: "thorns",
		// Пика (без дропа): максимальный разовый урон среди «простых» петалей,
		// хрупкая — glass cannon.
		BaseHP: 4.0, BaseDMG: 13.0,
	},
	"saw": {
		Name: "Saw", Shape: "whirl",
		// Пила (без дропа): сбалансированный дпс-петаль.
		BaseHP: 6.5, BaseDMG: 9.5,
	},
	"flare": {
		Name: "Flare", Shape: "circle",
		// Вспышка (без дропа): живучий дпс-петаль для долгого боя.
		BaseHP: 9.0, BaseDMG: 8.5,
	},
	"piercer": {
		Name: "Piercer", Shape: "thorns",
		// Пронзатель (дроп с Arrowhead, Triangle): острый клин, высокий
		// разовый урон при скромной живучести.
		BaseHP: 4.0, BaseDMG: 12.0,
	},
	"wingblade": {
		Name: "Wingblade", Shape: "thorns",
		// Крыло-клинок (дроп с Shardwing, Triangle): быстрый летающий осколок,
		// сбалансированный дпс.
		BaseHP: 5.0, BaseDMG: 10.0,
	},
	"rivet": {
		Name: "Rivet", Shape: "rock",
		// Заклёпка (дроп с Frame, Square): броневая петаль — очень живучая,
		// урон символический (механика Armor: сниженный износ).
		BaseHP: 20.0, BaseDMG: 3.0,
		Armor: true,
	},
	"tail": {
		Name: "Tail", Shape: "circle",
		// Хвост (дроп с Kite, Square): хлёсткий хвост-змей отталкивает моба
		// (механика Push) и наносит урон.
		BaseHP: 6.0, BaseDMG: 7.0,
		Push: true,
	},
	// === Ловушка (trap) ===
	// === Жетоны Ying/Yang (сюжет Hall of Trials) ===
	// Не самостоятельные «редкости»: Yang носит редкость зонной лестницы
	// (Point..Chaos = 0..7), Ying — истинной (Dot..Origin = 8..15). Поэтому
	// цвет/размер берутся из обычной редкости, а Token лишь запрещает бой и
	// включает soulbound. Ying-Yang — ОДИН и тот же ключ на все ворота: он
	// сгорает при открытии, и на следующем ярусе куётся заново (см.
	// origin_quests.go / origin_trade.go).
	"yang": {
		Name: "Yang", Shape: "oval",
		BaseHP: 1.0, BaseDMG: 0.0,
		Token: true,
	},
	"ying": {
		Name: "Ying", Shape: "thorns",
		BaseHP: 1.0, BaseDMG: 0.0,
		Token: true,
	},
	"yinyang": {
		Name: "Ying Yang", Shape: "circle",
		BaseHP: 1.0, BaseDMG: 0.0,
		Token: true,
	},
	// === Ловушка (trap) ===
	"trap": {
		Name: "Trap", Shape: "trap",
		// Ловушка (механика «капкан»): касание врага с HP ≤ 10% MaxHP запускает
		// захват (~1.8 с): моб заморожен, неуязвим и затягивается к зрителю-
		// владельцу, затем исчезает в Коллекции игрока. Урона не наносит
		// (DMG 0), износа не имеет. Редкость петеля задаёт «список» мобов:
		// mobCapturableBy (Dot ловит Point, Lattice — почти всех). Не крафтится,
		// не абсорбится и не выбрасывается (soulbound из Invoke) — одноразовый
		// подарок путешественников холла: каждый тир дарит ловушку раз в жизни.
		BaseHP: 10.0, BaseDMG: 0.0,
		Trap: true,
	},
}

// RealPetalKindKeys — отсортированный список ВСЕХ настоящих видов петалей
// (без служебных Passive вроде админ-петеля «nexus»). Единственный источник
// правды для клиентского счётчика коллекции «Shapes discovered»: раньше он
// держал свой захардкоженный список, который разъезжался с этим каталогом.
func RealPetalKindKeys() []string {
	keys := make([]string, 0, len(PetalKinds))
	for k, kind := range PetalKinds {
		if kind.Passive || kind.Token {
			continue // nexus и жетоны Ying/Yang — не «формы» коллекции
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Виды, из которых случайно выбирается петаль на земле.
var PetalKindsPool = []string{"oval", "thorns", "rock", "ball"}

// Спавн петалей-предметов на земле: сколько их и в каких зонах.
var ZonePetalCount = []int{5, 5, 5, 5, 5, 4, 4, 3}

// SetPetalKindsPool задаёт пул видов петалей зоны (копия, а не срез конфига).
func SetPetalKindsPool(p []string) { PetalKindsPool = append([]string(nil), p...) }
