package main

import (
	"math"
	"strconv"
	"testing"
)

// Первый уровень (зоны 0-7, включая Chaos): мобы максимум Square.
func TestFirstTierRarityCap(t *testing.T) {
	initWorld()
	for zone := 0; zone < secondTierOffset; zone++ {
		ceil := zoneRarityCeil[zone]
		for n := 0; n < 600; n++ {
			m := makeMob(zone, n)
			if m.Rarity > ceil {
				t.Fatalf("zone %d mob rarity %d exceeds cap %d", zone, m.Rarity, ceil)
			}
			if m.Rarity < 0 {
				t.Fatalf("zone %d negative rarity", zone)
			}
		}
	}
}

// Второй уровень (зоны 8-15): мобы в рамках капа зоны — 8-14 от Square до
// Chaos, Chaos II (15) — полный спектр. Зона 16 (TestRoom) не тир — пропускаем.
func TestSecondTierRarityRange(t *testing.T) {
	initWorld()
	for zone := secondTierOffset; zone < testRoomZone; zone++ {
		for n := 0; n < 600; n++ {
			m := makeMob(zone, n)
			if m.Rarity < zoneRarityFloor[zone] {
				t.Fatalf("L2 zone %d mob rarity %d below floor %d", zone, m.Rarity, zoneRarityFloor[zone])
			}
			if m.Rarity > zoneRarityCeil[zone] {
				t.Fatalf("L2 zone %d mob rarity %d exceeds ceil %d", zone, m.Rarity, zoneRarityCeil[zone])
			}
		}
	}
}

// Применение капа: веса ниже пола сливаются в пол, выше потолка отрезаются.
func TestApplyRarityClamp(t *testing.T) {
	// Ниже пола — в пол (порядок сохранён).
	w := []int{10, 20, 30, 40, 50, 60, 70, 80}
	got := applyRarityClamp(w, 3, 7)
	if got[3] != 10+20+30+40 {
		t.Fatalf("floor fold to Square = %d, want %d", got[3], 10+20+30+40)
	}
	for i := 4; i < 8; i++ {
		if got[i] != w[i] {
			t.Fatalf("index %d weight = %d, want %d", i, got[i], w[i])
		}
	}
	// Обрезка сверху (первый уровень, до Square): выше потолка отбрасывается,
	// градиент внутри тира сохраняется.
	got2 := applyRarityClamp(w, 0, 3)
	if got2[0] != 10 || got2[1] != 20 || got2[2] != 30 || got2[3] != 40 {
		t.Fatalf("ceiling cut broken: %v", got2)
	}
	for i := 4; i < len(w); i++ {
		if got2[i] != 0 {
			t.Fatalf("index %d above ceil = %d, want 0", i, got2[i])
		}
	}
	// Пустым результат не бывает: пол всегда непустой при floor > 0.
	w3 := []int{90, 10, 0, 0, 0, 0, 0, 0}
	got3 := applyRarityClamp(w3, 3, 7)
	if got3[3] != 100 {
		t.Fatalf("empty bottom band floor fold = %d, want 100", got3[3])
	}
}

// Портал «второй уровень»: в каждой первой-зоне есть портал вверх в её L2,
// а в L2 — портал вниз обратно; оба по центру X и L2 лежит над родителем.
func TestVerticalPortalsTiers(t *testing.T) {
	initWorld()
	if len(zones) != len(zoneNames) {
		t.Fatalf("zones = %d, want %d", len(zones), len(zoneNames))
	}
	for i := 0; i < secondTierOffset; i++ {
		z := zones[i]
		cx := z.X + z.W/2
		var up, down *Portal
		for j := range portals {
			pt := &portals[j]
			if pt.From == i && pt.To == i+secondTierOffset {
				up = pt
			}
			if pt.From == i+secondTierOffset && pt.To == i {
				down = pt
			}
		}
		if up == nil || down == nil {
			t.Fatalf("zone %d missing vertical portal (up=%v down=%v)", i, up != nil, down != nil)
		}
		// Портал по центру X.
		if up.X != cx || down.X != cx {
			t.Fatalf("zone %d portals not centered: up.X=%v down.X=%v cx=%v", i, up.X, down.X, cx)
		}
		// Портал «вверх» стоит у верха первого уровня, «вниз» — у низа второго.
		if up.Y != z.Y+vertPortalInset {
			t.Fatalf("zone %d up portal Y = %v, want %v", i, up.Y, z.Y+vertPortalInset)
		}
		if down.Y != zones[i+secondTierOffset].Y+zones[i+secondTierOffset].H-vertPortalInset {
			t.Fatalf("zone %d down portal Y = %v", i, down.Y)
		}
		// Второй уровень лежит над первым, с пустой полосой zoneRowGap между рядами.
		if zones[i+secondTierOffset].Y != -zoneH-zoneRowGap {
			t.Fatalf("L2 zone %d Y = %v, want %v", i+secondTierOffset, zones[i+secondTierOffset].Y, -zoneH-zoneRowGap)
		}
		if zones[i+secondTierOffset].X != z.X {
			t.Fatalf("L2 zone %d X = %v, want %v", i+secondTierOffset, zones[i+secondTierOffset].X, z.X)
		}
	}
}

// Chaos (зона 7) — как остальные первые зоны (до Square), а Chaos II (15) —
// единственная зона полного спектра. Зона 16 — TestRoom-витрина без порталов.
func TestChaosTiers(t *testing.T) {
	initWorld()
	if len(zones) != 17 {
		t.Fatalf("zones = %d, want 17", len(zones))
	}
	if zoneRarityFloor[7] != 0 || zoneRarityCeil[7] != 3 {
		t.Fatalf("chaos cap = [%d,%d], want [0,3] (полное зеркало первых зон)", zoneRarityFloor[7], zoneRarityCeil[7])
	}
	if zoneRarityFloor[15] != 0 || zoneRarityCeil[15] != 7 {
		t.Fatalf("chaos II cap = [%d,%d], want [0,7]", zoneRarityFloor[15], zoneRarityCeil[15])
	}
	if zones[15].Name != "Chaos II" {
		t.Fatalf("zone 15 name = %q, want Chaos II", zones[15].Name)
	}
	if zones[15].X != zones[7].X || zones[15].Y != -zoneH-zoneRowGap {
		t.Fatalf("Chaos II pos = (%v,%v), want over Chaos (X=%v,Y=%v)", zones[15].X, zones[15].Y, zones[7].X, -zoneH-zoneRowGap)
	}
	// Боковой портал в Chaos только из Fractal(6); вертикальный — из Chaos II.
	var up7, down7 *Portal
	for i := range portals {
		pt := &portals[i]
		if pt.Dir != 0 && pt.To == 7 && pt.From != 6 {
			t.Fatalf("unexpected horizontal portal to chaos: from %d", pt.From)
		}
		if pt.From == 7 && pt.To == 15 {
			up7 = pt
		}
		if pt.From == 15 && pt.To == 7 {
			down7 = pt
		}
	}
	if up7 == nil || down7 == nil {
		t.Fatalf("chaos vertical portal missing (up=%v down=%v)", up7 != nil, down7 != nil)
	}
	full := makeMob(15, 1)
	if full.Rarity < 0 || full.Rarity > 7 {
		t.Fatalf("chaos II mob rarity %d out of range", full.Rarity)
	}
}

// TestRoom (зона 16): витрина с 16 статичными мокапами (1-15 старые, 16 Prism),
// без порталов и обычных мобов — попасть можно только админ-телепортом.
func TestRooms(t *testing.T) {
	initWorld()
	if zones[testRoomZone].Name != "TestRoom" {
		t.Fatalf("zone 16 name = %q, want TestRoom", zones[testRoomZone].Name)
	}
	for i := range portals {
		if portals[i].From == testRoomZone || portals[i].To == testRoomZone {
			t.Fatalf("testroom must be isolated, got portal %d (from %d to %d)", portals[i].ID, portals[i].From, portals[i].To)
		}
	}
	if zoneMobLimits[testRoomZone] != 0 {
		t.Fatalf("zone 16 mob limit = %d, want 0", zoneMobLimits[testRoomZone])
	}
	show := makeShowcaseMobs(testRoomZone)
	if len(show) != 16 {
		t.Fatalf("showcase mobs = %d, want 16", len(show))
	}
	for i, m := range show {
		if !m.Showcase {
			t.Fatalf("showcase %d not marked Showcase", i)
		}
		if m.Label != strconv.Itoa(i+1) {
			t.Fatalf("showcase %d label = %q, want %q", i, m.Label, strconv.Itoa(i+1))
		}
		if m.Rarity != 5 {
			t.Fatalf("showcase %d rarity = %d, want 5 (Hexagon)", i, m.Rarity)
		}
		// Раскладка сетки внутри зоны (все мокапы раздельно и в границах).
		if m.X <= zones[testRoomZone].X || m.X >= zones[testRoomZone].X+zones[testRoomZone].W {
			t.Fatalf("showcase %d X = %v out of zone", i, m.X)
		}
		if m.Y <= zones[testRoomZone].Y || m.Y >= zones[testRoomZone].Y+zones[testRoomZone].H {
			t.Fatalf("showcase %d Y = %v out of zone", i, m.Y)
		}
	}
	// Два соседних мокапа не пересекаются (радиус ~42).
	for i := 0; i < len(show); i++ {
		for j := i + 1; j < len(show); j++ {
			dx := show[i].X - show[j].X
			dy := show[i].Y - show[j].Y
			if dx*dx+dy*dy < (show[i].Radius+show[j].Radius)*(show[i].Radius+show[j].Radius) {
				t.Fatalf("showcase %d and %d overlap", i+1, j+1)
			}
		}
	}
}

// TestZoneCorners — скругление углов зон II и I: центр не заходит за дугу
// (держится на дуге касания crs[i] - радиус, ВНУТРИ зоны). У зон первого
// уровня скруглены верхние углы зоны и нижние углы МОБ-зоны (на ЛИНИИ
// разделительной стены); углы Safe Zone ниже стены остаются прямыми.
func TestZoneCorners(t *testing.T) {
	initWorld()
	z := zones[9] // Line II — все 4 угла скруглены
	if z.crs[0] != zoneCornerR || z.crs[3] != zoneCornerR {
		t.Fatalf("zone 9 crs = %v, want all %v", z.crs, zoneCornerR)
	}
	// Зона первого уровня: верхние углы зоны скруглены, нижние Safe Zone
	// остаются прямыми (моб-зона скругляется отдельными дугами на линии стены).
	if zones[1].crs[0] != zoneCornerR || zones[1].crs[1] != zoneCornerR {
		t.Fatalf("zone 1 (Line) top crs = %v, want %v", zones[1].crs, zoneCornerR)
	}
	if zones[1].crs[2] != 0 || zones[1].crs[3] != 0 {
		t.Fatalf("zone 1 (Line) bottom crs = %v, want 0 (Safe Zone прямые)", zones[1].crs)
	}
	const r = 42.0
	cr := z.crs[0] - r

	// Точка в «кармане» угла (за дугой 700, но в плоских границах зоны) должна
	// лечь на дугу касания ВНУТРИ зоны: dist = crs[0] - r.
	cx, cy := z.X+z.crs[0], z.Y+z.crs[0]
	x, y := clampZoneRounded(&z, cx-565.69, cy-565.69, r) // dist 800 от центра угла
	dx, dy := x-cx, y-cy
	d := math.Sqrt(dx*dx + dy*dy)
	if math.Abs(d-cr) > 0.5 {
		t.Fatalf("corner clamp dist = %v, want ≈%v (point %v,%v)", d, cr, x, y)
	}
	if x <= z.X || y <= z.Y {
		t.Fatalf("corner clamp pushed OUT of zone: %v,%v", x, y)
	}

	// Точка в вырезанном углу (за пределами плоских границ) втягивается внутрь.
	x2, y2 := clampZoneRounded(&z, cx-900, cy-900, r)
	dx2, dy2 := x2-cx, y2-cy
	d2 := math.Sqrt(dx2*dx2 + dy2*dy2)
	if math.Abs(d2-cr) > 0.5 {
		t.Fatalf("void corner clamp dist = %v, want ≈%v (point %v,%v)", d2, cr, x2, y2)
	}

	// Низ зоны I по центру остаётся плоским (скругление только возле углов).
	z1 := zones[1]
	yBot := z1.Y + z1.H - r
	xBot := z1.X + z1.W/2
	x, y = clampZoneRounded(&z1, xBot, yBot+r*2, r)
	if y != yBot {
		t.Fatalf("L1 bottom clamp y = %v, want %v (центр низа плоский)", y, yBot)
	}
	_ = x

	// НИЖНИЙ ЛЕВЫЙ угол МОБ-зоны (на линии стены) скруглён, как у L2: точка в
	// «кармане» у линии стены (выше неё) ложится на дугу касания zoneCornerR-r.
	mb := mobZoneBottom(&z1)
	blCx, blCy := z1.X+zoneCornerR, mb-zoneCornerR
	blx, bly := clampZoneRounded(&z1, blCx-565.69, blCy+565.69, r) // dist 800
	dx3, dy3 := blx-blCx, bly-blCy
	d3 := math.Sqrt(dx3*dx3 + dy3*dy3)
	crB := zoneCornerR - r
	if math.Abs(d3-crB) > 0.5 {
		t.Fatalf("L1 mob-zone bottom-left corner clamp dist = %v, want ≈%v (point %v,%v)", d3, crB, blx, bly)
	}
	if bly > mb {
		t.Fatalf("L1 mob-zone bottom-left clamp below wall line: %v,%v (starts in safe zone?)", blx, bly)
	}

	// А в Safe Zone (НИЖЕ стены) угол остаётся ПРЯМЫМ: плоская нижняя граница.
	sx, sy := clampZoneRounded(&z1, z1.X-300, z1.Y+z1.H+r*2, r)
	if sy != yBot || sx != z1.X+r {
		t.Fatalf("L1 safe-zone bottom-left corner must stay SQUARE, got %v,%v (want %v,%v)", sx, sy, z1.X+r, yBot)
	}
}

// Второй уровень: редкость жёстко «по высоте» (большие прямоугольники).
// Ниже верхней четверти Hexagon/Fractal/Chaos спавниться НЕ могут; нижние
// полосы начинаются с Square (Point/Line у Chaos II).
func TestL2RarityVerticalBands(t *testing.T) {
	initWorld()

	// Пул редкости нижних прямоугольников (0-2) не содержит Hexagon+.
	for i := 0; i < 3; i++ {
		for r := 5; r <= 7; r++ {
			if l2RarityBands[i][r] != 0 {
				t.Fatalf("l2RarityBands[%d][%d]=%d, want 0 (Hexagon+ только верхняя четверть)", i, r, l2RarityBands[i][r])
			}
		}
	}
	sumHigh := 0
	for r := 5; r <= 7; r++ {
		sumHigh += l2RarityBands[3][r]
	}
	if sumHigh <= 0 {
		t.Fatalf("верхняя полоса без Hexagon+ весов")
	}

	z8 := zones[8]
	bottom := mobZoneBottom(&z8)
	top := z8.Y
	// У самого входа (низ зоны): только Square (полоса 0, вес 100).
	for n := 0; n < 2000; n++ {
		if r := pickL2RarityAtY(&z8, bottom-1); r != 3 {
			t.Fatalf("низ зоны 8 дал редкость %d, want 3 (Square)", r)
		}
	}
	// Полоса 2 (верхние 50-75% высоты): Square/Polygon, но не Hexagon+.
	for n := 0; n < 3000; n++ {
		y := top + z8.H*0.4 // t=0.4 от верха -> интенсивность 0.6 -> полоса 2
		r := pickL2RarityAtY(&z8, y)
		if r > 4 {
			t.Fatalf("полоса 2 дала редкость %d, want <=4", r)
		}
	}
	// Верхняя четверть: внутри 3..7, и Hexagon+ встречается (шанс >=52%).
	seenHigh := false
	for n := 0; n < 10000; n++ {
		r := pickL2RarityAtY(&z8, top+10)
		if r < 3 || r > 7 {
			t.Fatalf("верх зоны 8 дал редкость %d, want 3..7", r)
		}
		if r >= 5 {
			seenHigh = true
		}
	}
	if !seenHigh {
		t.Fatalf("верхняя полоса зоны 8 не дала ни одного Hexagon+ за 10000 попыток")
	}

	// Chaos II (15): низ без Square — только Point/Line.
	z15 := zones[15]
	b15 := mobZoneBottom(&z15)
	for n := 0; n < 2000; n++ {
		if r := pickL2RarityAtY(&z15, b15-1); r > 1 {
			t.Fatalf("низ Chaos II дал редкость %d, want <=1", r)
		}
	}
}

// Спавн-позиция совпадает с редкостью: моб появляется в той полосе, по
// которой выпала его редкость. Раньше редкость крутилась по одной случайной
// высоте, а позиция бралась по ДРУГОЙ — из-за этого Hexagon-моб (в т.ч. Tear)
// заспавнивался прямо у входа внизу зоны II.
func TestL2SpawnBindsRarityToHeight(t *testing.T) {
	initWorld()
	z := zones[8] // Point II
	topQ := z.Y + z.H*0.25
	top75 := z.Y + z.H*0.75
	for n := 0; n < 8000; n++ {
		m := makeMob(8, n)
		if m.Rarity >= 5 && m.Y > topQ {
			t.Fatalf("rarity %d (type %s) заспавнен в Y=%v ниже верхней четверти %v", m.Rarity, m.Type, m.Y, topQ)
		}
		if m.Rarity == 4 && m.Y > top75 {
			t.Fatalf("rarity 4 (type %s) заспавнен в Y=%v ниже верхних 3/4 %v", m.Type, m.Y, top75)
		}
		if m.Rarity == 3 && m.Y < z.Y+z.H*0.25+1 {
			// Square у входа — норма (низ = Square).
		}
	}
}