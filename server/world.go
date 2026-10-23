package main

import (
	"math"
	"sync"
)

// world.go - геометрия мира: зоны и порталы.
// Мир устроен ЛИНЕЙНО: 8 зон выстроены в ряд по оси X.
// Между зонами пустое "расстояние" (заблокировано стенами), чтобы игрок
// из своей зоны не видел соседнюю.

const (
	zoneW        = 8000.0 // ширина зоны
	zoneH        = 5000.0 // высота зоны
	zoneGap      = 2000.0 // пустое расстояние между зонами одного ряда (за стенами)
	zoneRowGap   = 300.0  // пустая полоса между рядом первого уровня и рядом второго уровня
	portalInset  = 130.0  // расстояние от боковой стены, на котором стоит портал

	// Второй уровень зон. Первые secondTierOffset зон (0-7, Point..Chaos)
	// выстроены в ряд по X; над КАЖДОЙ из них лежит зона второго уровня
	// (индекс i+secondTierOffset), куда ведёт портал сверху посередине, обратно —
	// портал снизу. Первый уровень весь «безопасный» (максимум Square, включая
	// Chaos); на втором уровне 8-14 мобы от Square до Chaos, а Chaos II (15) —
	// единственная зона полного спектра.
	firstTierZones     = 7    // сколько зон «безопасного тира» Point..Fractal: у Chaos нет правого соседа в ряду
	secondTierOffset   = 8    // индекс зоны второго уровня для зоны i: i+8

	// Порталы «второй уровень» стоят вверху/внизу посередине зоны.
	vertPortalInset = 100.0 // отступ от края зоны (верх первого тира / низ второго)

	// Safe Zone / стена между Safe Zone и Mob Zone.
	// Внизу каждой зоны полоса безопасности высотой safeZoneH: там нет мобов,
	// игроку безопасно. Между Safe Zone и Mob Zone — горизонтальная стена
	// толщиной zoneWallTh на всю ширину зоны с проходом zoneWallGap посередине.
	// Мобы живут ТОЛЬКО в Mob Zone (выше стены), в стену не заходят.
	safeZoneH   = 500.0 // высота Safe Zone (низ зоны)
	zoneWallTh  = 300.0 // толщина разделительной стены
	zoneWallGap = 500.0 // ширина прохода посередине стены

	// Чекпоинт-площадка: нижний центральный «квадрат» сейф-зоны. Встав на неё,
	// игрок закрепляет спавнпоинт этой зоны (против читерского телепорта
	// через релог — версия Т).
	checkpointPadHalfW = 400.0 // половина ширины площадки (800 всего)

	// Индекс витринной зоны TestRoom: 25 статичных мокапов мобов для выбора
	// дизайнов (16 Diamond .. 25 Prism). Без порталов и обычных мобов — попасть
	// можно только админ-телепортом из панели администратора.
	testRoomZone = 16

	// Скругление углов зон ВТОРОГО уровня (8-15): радиус дуги угла. Прямые
	// углы давали «карманы», в которых застревали огромные Fractal/Chaos, а
	// игроки могли чизить гигантов из-за радиуса. Дуга режет карман.
	zoneCornerR = 700.0
)

type Zone struct {
	Index int
	Biome int
	Name  string
	X, Y  float64
	W, H  float64
	Color uint32

	// crs — радиус скругления КАЖДОГО угла зоны (дуга) в порядке:
	// [верх-лево, верх-право, низ-лево, низ-право]. У зон L2 скруглены все
	// четыре угла зоны. У зон L1 скруглены верхние углы зоны (Mob Zone),
	// а нижние углы МОБ-зоны (на линии стены) — отдельными дугами cxsM/cysM;
	// Safe Zone ниже стены остаётся прямоугольной.
	crs [4]float64

	// Предвычисленные центры углов и знаки клиньев для clampZoneRounded
	// (массивы[4], чтобы не аллоцировать слайсы на каждый вызов в тике).
	cxs, cys [4]float64
	sxs, sys [4]float64

	// Центры нижних углов МОБ-зоны (L1, на линии стены): cxsM/cysM[2] —
	// [низ-лево, низ-право]. У зон без разделительной стены — нулевые.
	cxsM, cysM [2]float64
}

// cornerR возвращает радиус скругления угла i (для совместимого кода).
func (z *Zone) cornerR(i int) float64 {
	return z.crs[i]
}

type Portal struct {
	ID       int
	From, To int
	X, Y     float64
	W, H     float64
	Dir      int
}

// 17 зон: первый уровень (Point..Chaos) в ряд по оси X + 8 зон второго уровня
// (индексы 8-15), каждая прямо над своей «родительской» + зона 16 TestRoom —
// витрина с 25 статичными мокапами мобов (только админ-телепорт, без порталов).
var zoneNames = []string{
	"Point", "Line", "Triangle", "Square", "Pentagon", "Hexagon", "Fractal", "Chaos",
	"Point II", "Line II", "Triangle II", "Square II", "Pentagon II", "Hexagon II", "Fractal II", "Chaos II",
	"TestRoom",
}
var zoneBiomes = []int{0, 1, 2, 3, 4, 5, 6, 7, 0, 1, 2, 3, 4, 5, 6, 7, 5}
var zoneColors = []uint32{
	0x2ecc71, // Point — зелёный
	0xf1c40f, // Line — жёлтый
	0xe67e22, // Triangle — оранжевый
	0xe74c3c, // Square — красный
	0x3498db, // Pentagon — синий
	0x1abc9c, // Hexagon — бирюзовый
	0x9b59b6, // Fractal — фиолетовый
	0x2c3e50, // Chaos — тёмный
	0x1e9957, // Point II — тёмно-зелёный
	0xa8860a, // Line II — тёмно-жёлтый
	0xa04d16, // Triangle II — тёмно-оранжевый
	0xa92e2b, // Square II — тёмно-красный
	0x256ba5, // Pentagon II — тёмно-синий
	0x12806b, // Hexagon II — тёмно-бирюзовый
	0x6f3f85, // Fractal II — тёмно-фиолетовый
	0x1b2631, // Chaos II — ещё темнее Chaos
	0x0e5748, // TestRoom — тёмно-бирюзовый (витрина Hexagon-мокапов)
}

// Редкости (отдельно от зон!) — единый реестр server/rarity (rarity_registry.go):
// 16 ступеней, Point..Chaos (0..7) + истинная Dot..Origin (8..15).
// Индекс 4 — Pentagon (раньше ошибочно назывался «Polygon»).

const portalSize   = 140.0
const portalHeight = 140.0
const vertPortalW  = 200.0
const vertPortalH  = 260.0

var zones   []Zone
var portals []Portal

var worldInitOnce sync.Once

// initWorld инициализирует глобальный мир ровно ОДИН раз за процесс. Раньше
// каждый createLobby (newWaveRoom → initWorld) перезаписывал zones/portals/
// стены заново: параллельное создание лобби давало data race на глобалах
// (нашлось тестом TestLobbyCodeRace с -race).
func initWorld() {
	worldInitOnce.Do(initWorldOnce)
}

func initWorldOnce() {
	zones = make([]Zone, len(zoneNames))
	for i := 0; i < len(zoneNames); i++ {
		x0 := float64(i) * (zoneW + zoneGap)
		y0 := 0.0
		// Зоны второго уровня лежат НАД зонами первого уровня,
		// с пустой полосой zoneRowGap между рядами.
		if i >= secondTierOffset {
			x0 = float64(i-secondTierOffset) * (zoneW + zoneGap)
			y0 = -zoneH - zoneRowGap
		}
		zones[i] = Zone{
			Index: i, Biome: zoneBiomes[i], Name: zoneNames[i],
			X: x0, Y: y0, W: zoneW, H: zoneH, Color: zoneColors[i],
		}
		if i >= secondTierOffset {
			// Зоны II: скруглены все четыре угла зоны (вся зона — моб-зона).
			zones[i].crs = [4]float64{zoneCornerR, zoneCornerR, zoneCornerR, zoneCornerR}
		} else {
			// Зоны I: скруглены верхние углы зоны; нижние углы МОБ-зоны (на
			// линии разделительной стены) — дугами cxsM/cysM (см. ниже).
			// Safe Zone ниже стены остаётся прямоугольной.
			zones[i].crs = [4]float64{zoneCornerR, zoneCornerR, 0, 0}
		}
	}
	// Предвычисляем углы скругления для всех зон (для clampZoneRounded).
	for i := range zones {
		z := &zones[i]
		z.cxs = [4]float64{z.X + z.crs[0], z.X + z.W - z.crs[1], z.X + z.crs[2], z.X + z.W - z.crs[3]}
		z.cys = [4]float64{z.Y + z.crs[0], z.Y + z.crs[1], z.Y + z.H - z.crs[2], z.Y + z.H - z.crs[3]}
		z.sxs = [4]float64{-1, 1, -1, 1}
		z.sys = [4]float64{-1, -1, 1, 1}
		// L1: нижние углы МОБ-зоны (на линии разделительной стены) — дуга
		// zoneCornerR с центрами чуть ВЫШЕ линии стены. Safe Zone не трогаем.
		if hasSafeZone(z) {
			mb := mobZoneBottom(z)
			z.cxsM = [2]float64{z.X + zoneCornerR, z.X + z.W - zoneCornerR}
			z.cysM = [2]float64{mb - zoneCornerR, mb - zoneCornerR}
		}
	}
	initWalls()

	// Порталы — на боковых стенах зоны, на средней высоте Safe Zone: назад
	// (предыдущая зона) слева, вперёд (следующая зона) справа. Только у зон
	// ПЕРВОГО уровня: зоны второго уровня (8-15) изолированы — из них есть
	// только портал вниз, обратно в родительскую зону.
	portals = make([]Portal, 0, (secondTierOffset-1)*2+secondTierOffset*2)
	id := 1
	for i := 0; i < secondTierOffset; i++ {
		z := zones[i]
		portalY := z.Y + z.H - safeZoneH/2
		if i > 0 {
			portals = append(portals, Portal{
				ID: id, From: i, To: i - 1,
				X: z.X + portalInset, Y: portalY,
				W: portalSize, H: portalHeight, Dir: -1,
			})
			id++
		}
		if i < firstTierZones {
			portals = append(portals, Portal{
				ID: id, From: i, To: i + 1,
				X: z.X + z.W - portalInset, Y: portalY,
				W: portalSize, H: portalHeight, Dir: +1,
			})
			id++
		}
	}
	// Вертикальные порталы «второй уровень»: в каждой зоне первого уровня (0-7)
	// портал СВЕРХУ посередине ведёт на второй уровень (зона i+8), оттуда
	// портал ВНИЗУ посередине возвращает обратно — как у всех остальных зон.
	for i := 0; i < secondTierOffset; i++ {
		z := zones[i]
		lz := zones[i+secondTierOffset]
		cx := z.X + z.W/2
		// Вверх: первый уровень -> второй уровень.
		portals = append(portals, Portal{
			ID: id, From: i, To: i + secondTierOffset,
			X: cx, Y: z.Y + vertPortalInset,
			W: vertPortalW, H: vertPortalH, Dir: 0,
		})
		id++
		// Вниз: второй уровень -> первый уровень.
		portals = append(portals, Portal{
			ID: id, From: i + secondTierOffset, To: i,
			X: cx, Y: lz.Y + lz.H - vertPortalInset,
			W: vertPortalW, H: vertPortalH, Dir: 0,
		})
		id++
	}
}

// portalAt проверяет, пересёк ли игрок (круг с центром x,y и радиусом r)
// какой-либо портал. Возвращает портал или nil.
func portalAt(x, y, r float64) *Portal {
	for i := range portals {
		p := &portals[i]
		if x > p.X-p.W/2 && x < p.X+p.W/2 &&
			y > p.Y-p.H/2 && y < p.Y+p.H/2 {
			return p
		}
	}
	return nil
}

// clampToZone не даёт игроку выйти за пределы его зоны (стены зоны).
// Возвращает исправленные координаты x, y. У зон второго уровня углы
// скруглены — работает общий скруглённый кламп.
func clampToZone(zoneIndex int, x, y, r float64) (float64, float64) {
	return clampZoneRounded(&zones[zoneIndex], x, y, r)
}

// clampZoneRounded — общий кламп центра круга радиуса r в границах зоны.
// Для зон со скруглёнными углами (cornerR > 0) углы ограничены дугой: центр
// круга остаётся «внутри» дуги — на расстоянии cornerR-r от центра угла
// (касание дуги), т.е. круг никогда не вылезает за скруглённую стену.
// Кламп чистый (без итераций): сперва плоские стены, затем каждый угол —
// если точка в клине угла и дальше дуги касания, проекция радиально внутрь.
func clampZoneRounded(z *Zone, x, y, r float64) (float64, float64) {
	if x < z.X+r {
		x = z.X + r
	}
	if x > z.X+z.W-r {
		x = z.X + z.W - r
	}
	if y < z.Y+r {
		y = z.Y + r
	}
	if y > z.Y+z.H-r {
		y = z.Y + z.H - r
	}
	// Предвычисленные углы (z.cxs/z.cys/z.sxs/z.sys) — без аллокаций на тик.
	// Всегда обрабатываем все 4 угла: у прямых (сrs=0) cr<0, ветка скипается.
	for i := 0; i < 4; i++ {
		cr := z.crs[i] - r // радиус дуги касания для центра круга
		if cr <= 0 || 2*z.crs[i] >= z.W || 2*z.crs[i] >= z.H {
			continue
		}
		dx := x - z.cxs[i]
		dy := y - z.cys[i]
		// Точка в клине угла (в сторону вырезанного угла).
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
		// Проекция внутрь: на дугу касания (круг не вылезает за стену).
		x = z.cxs[i] + dx/d*cr
		y = z.cys[i] + dy/d*cr
	}
	// L1: нижние углы МОБ-зоны скруглены на ЛИНИИ стены (как у L2) — но только
	// ВЫШЕ стены: в самих углах моб-зоны центр не заходит за дугу касания.
	// Safe Zone ниже стены остаётся прямоугольной (там дугу не применяем).
	if z.cxsM[0] > 0 && y < mobZoneBottom(z) {
		for j := 0; j < 2; j++ {
			cr := zoneCornerR - r
			if cr <= 0 {
				continue
			}
			dx := x - z.cxsM[j]
			dy := y - z.cysM[j]
			if j == 0 {
				if dx >= 0 || dy <= 0 {
					continue // клин левого нижнего угла моб-зоны: dx<0 && dy>0
				}
			} else {
				if dx <= 0 || dy <= 0 {
					continue // клин правого нижнего угла моб-зоны: dx>0 && dy>0
				}
			}
			d := math.Sqrt(dx*dx + dy*dy)
			if d <= cr {
				continue
			}
			if d < 1e-9 {
				d = 1e-9
			}
			x = z.cxsM[j] + dx/d*cr
			y = z.cysM[j] + dy/d*cr
		}
	}
	return x, y
}

// hasSafeZone — есть ли у зоны Safe Zone и разделительная стена снизу.
// Зоны ВТОРОГО уровня (8-14) безжалостны: безопасной полосы и стены у них
// нет — вся зона целиком моб-зона, от портала внизу до верха.
func hasSafeZone(z *Zone) bool {
	return z.Index < secondTierOffset
}

// safeZoneY возвращает Y верхней границы Safe Zone зоны (ниже — безопасно).
// У зон второго уровня Safe Zone нет — верхняя граница = дно зоны.
func safeZoneY(z *Zone) float64 {
	if !hasSafeZone(z) {
		return z.Y + z.H
	}
	return z.Y + z.H - safeZoneH
}

// mobZoneBottom возвращает Y нижней границы Mob Zone (верхняя граница стены) —
// ниже стены/в Safe Zone мобы не живут. У зон второго уровня границы нет —
// мобы живут по всей высоте зоны.
func mobZoneBottom(z *Zone) float64 {
	if !hasSafeZone(z) {
		return z.Y + z.H
	}
	return z.Y + z.H - safeZoneH - zoneWallTh
}

// wallSegments возвращает два прямоугольника разделительной стены зоны
// (без прохода посередине). Стена — горизонтальная полоса толщиной zoneWallTh
// на всю ширину зоны, разорванная проходом zoneWallGap по центру. У зон
// второго уровня стены нет.
func wallSegments(z *Zone) [2]Rect {
	if !hasSafeZone(z) {
		return [2]Rect{}
	}
	top := mobZoneBottom(z)
	cy := z.X + z.W/2
	half := zoneWallGap / 2
	return [2]Rect{
		{X: z.X, Y: top, W: cy - half - z.X, H: zoneWallTh},
		{X: cy + half, Y: top, W: z.X + z.W - (cy + half), H: zoneWallTh},
	}
}

// wallRect возвращает прямоугольник разделительной стены зоны целиком
// (без прохода) — используется для мобов, которым нельзя в Safe Zone вовсе.
// У зон второго уровня стены нет.
func wallRect(z *Zone) Rect {
	if !hasSafeZone(z) {
		return Rect{}
	}
	return Rect{X: z.X, Y: mobZoneBottom(z), W: z.W, H: zoneWallTh}
}

// pushCircleOutOfRect выталкивает круг (центр x,y, радиус r) из прямоугольника
// rect, если он пересекает его. Возвращает исправленные координаты.
func pushCircleOutOfRect(x, y, r float64, rect Rect) (float64, float64) {
	nx := clampf(x, rect.X, rect.X+rect.W)
	ny := clampf(y, rect.Y, rect.Y+rect.H)
	dx := x - nx
	dy := y - ny
	d2 := dx*dx + dy*dy
	if d2 >= r*r {
		return x, y
	}
	if d2 == 0 {
		// Центр внутри прямоугольника — выталкиваем по меньшей оси.
		left := x - rect.X
		right := rect.X + rect.W - x
		top := y - rect.Y
		bottom := rect.Y + rect.H - y
		m := left
		if right < m {
			m = right
		}
		if top < m {
			m = top
		}
		if bottom < m {
			m = bottom
		}
		if m == left {
			return rect.X - r, y
		}
		if m == right {
			return rect.X + rect.W + r, y
		}
		if m == top {
			return x, rect.Y - r
		}
		return x, rect.Y + rect.H + r
	}
	d := math.Sqrt(d2)
	push := r - d
	return x + dx/d*push, y + dy/d*push
}

// clampPlayerInZone удерживает игрока в зоне с учётом разделительной стены:
// не даёт пересечь стену, кроме прохода по центру. Дополнительно выталкивает
// из всех конфигурируемых стен зоны (см. walls.go).
func clampPlayerInZone(zoneIndex int, x, y, r float64) (float64, float64) {
	x, y = clampToZone(zoneIndex, x, y, r)
	z := zones[zoneIndex]
	segs := wallSegments(&z)
	for _, s := range segs {
		x, y = pushCircleOutOfRect(x, y, r, s)
	}
	for _, w := range zoneWallRects(zoneIndex) {
		x, y = pushCircleOutOfRect(x, y, r, w)
	}
	return x, y
}

// Slimy-стены: отталкивание ИГРОКОВ от стен зоны (анти-AFK-фарм вплотную
// к стене). Push пропорционален близости и ТЕКУЩЕЙ скорости игрока: у самой
// стены — spd×slimeWallFactor, за slimeWallRange — нулевое. Чтобы обычный
// подход/касание стены НЕ блокировались «полем с метра», push включается
// плавным рампом wallHold: он копится только пока игрок ЖМЁТ движение в стену
// (кламп реально удерживает его у стены). Быстрое касание свободно, а
// длительное удержание через slimeWallRampTime секунд наращивает push до
// максимума — стена выталкивает игрока в моб-зону (зажми → афк нельзя).
const (
	slimeWallRange     = 180.0
	slimeWallFactor    = 1.6 // push у самой стены = скорость_игрока×этот_множитель
	slimeWallRampTime  = 1.2 // сек до полного push при непрерывном вдавливании
	slimeWallDecayRate = 3.0 // скорость спада накопителя (в раз/сек) при отпускании
)

// wallSlimePush возвращает смещение игрока от ближайших стен зоны за такт:
// учитывает границы зоны, разделительную стену и конфигурируемые стены.
// spd — текущая (максимальная) скорость перемещения игрока в этом такте.
func wallSlimePush(zoneIndex int, x, y, spd, dt float64) (float64, float64) {
	z := &zones[zoneIndex]
	push := func(d float64) float64 {
		if d >= slimeWallRange {
			return 0
		}
		return spd * slimeWallFactor * (1 - d/slimeWallRange) * dt
	}
	var px, py float64
	// Границы зоны (по центру игрока — упрощённо, для плавного отскока).
	if d := x - z.X; d < slimeWallRange {
		px += push(d)
	}
	if d := z.X + z.W - x; d < slimeWallRange {
		px -= push(d)
	}
	if d := y - z.Y; d < slimeWallRange {
		py += push(d)
	}
	if d := z.Y + z.H - y; d < slimeWallRange {
		py -= push(d)
	}
	// Разделительная стена и кастомные стены: центром игрока (для плавности).
	rects := make([]Rect, 0, 6)
	segs := wallSegments(z)
	rects = append(rects, segs[:]...)
	rects = append(rects, zoneWallRects(zoneIndex)...)
	for _, r := range rects {
		nx := clampf(x, r.X, r.X+r.W)
		ny := clampf(y, r.Y, r.Y+r.H)
		dx := x - nx
		dy := y - ny
		d2 := dx*dx + dy*dy
		if d2 > slimeWallRange*slimeWallRange {
			continue
		}
		if d2 < 1e-6 {
			// Центр ВНУТРИ стены (не должно случаться после клампа) — толкаем
			// по наименьшей оси наружу, как pushCircleOutOfRect.
			left := x - r.X
			right := r.X + r.W - x
			top := y - r.Y
			bottom := r.Y + r.H - y
			m := left
			if right < m {
				m = right
			}
			if top < m {
				m = top
			}
			if bottom < m {
				m = bottom
			}
			pu := spd * slimeWallFactor * dt
			if m == left {
				px -= pu
			} else if m == right {
				px += pu
			} else if m == top {
				py -= pu
			} else {
				py += pu
			}
			continue
		}
		d := math.Sqrt(d2)
		if d > slimeWallRange {
			continue
		}
		p := spd * slimeWallFactor * (1 - d/slimeWallRange) * dt
		px += (dx / d) * p
		py += (dy / d) * p
	}
	return px, py
}
