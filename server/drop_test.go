package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// Дроп-таблица: «максимум — редкость самого моба». Дроп никогда не выше своего
// тира; Chaos-моб (7) максимум Fractal (6); Chaos-столбец не выпадает никогда.
func TestPickDropRarityBounds(t *testing.T) {
	for mobRar := 0; mobRar < len(rarityNames); mobRar++ {
		max := mobRar
		if max > rarityFractal {
			max = rarityFractal // Chaos-моб максимум Fractal
		}
		for i := 0; i < 5000; i++ {
			r := pickDropRarity(mobRar)
			if r > max {
				t.Fatalf("mob rar %d dropped %d (макс %d)", mobRar, r, max)
			}
		}
	}
}

// Топ-линии дропа нарочно редкие: Fractal-моб → 0.2% собственного Fractal,
// Chaos-моб → ~2% Fractal. Проверяем частоты на большом количестве бросков.
func TestPickDropRarityHeadRates(t *testing.T) {
	const N = 60000
	ff := 0
	for i := 0; i < N; i++ {
		if pickDropRarity(rarityFractal) == rarityFractal {
			ff++
		}
	}
	pF := float64(ff) / float64(N)
	if pF < 0.0008 || pF > 0.005 {
		t.Fatalf("Fractal-моб: Fractal rate = %.4f, want ~0.002", pF)
	}
	cf := 0
	for i := 0; i < N; i++ {
		if pickDropRarity(rarityChaos) == rarityFractal {
			cf++
		}
	}
	pC := float64(cf) / float64(N)
	if pC < 0.012 || pC > 0.03 {
		t.Fatalf("Chaos-моб: Fractal rate = %.4f, want ~0.02", pC)
	}
}

// Качели по шестиугольнику (см. mobRarityDrop, ребаланс):
// Hexagon-моб кидает свой Hexagon ультра-редко (~0.67%), основной дроп —
// Square (~80%). Fractal-моб — Hexagon = 16.7% (1/6), Chaos-моб — основной.
// Polygon-моб: свой Polygon жёстко порезан до ~5%. Chaos-моб НИКОГДА
// не кидает ниже Pentagon (4).
func TestPickDropRarityNewRates(t *testing.T) {
	const N = 60000
	const hexIdx = 5 // Pentагон=4, Hexagon=5
	const fracMob = 6
	const chaosMob = 7
	const polyMob = 4

	hexOwn := pickRateN(t, hexIdx, hexIdx, N)
	if hexOwn < 0.014 || hexOwn > 0.021 {
		t.Fatalf("Hexagon-моб: свой Hexagon = %.4f, want ~0.0167", hexOwn)
	}
	frHex := pickRateN(t, fracMob, hexIdx, N)
	if frHex < 0.14 || frHex > 0.19 {
		t.Fatalf("Fractal-моб: Hexagon = %.4f, want ~0.167", frHex)
	}
	polyOwn := pickRateN(t, polyMob, 4, N)
	if polyOwn < 0.005 || polyOwn > 0.02 {
		t.Fatalf("Polygon-моб: свой Polygon = %.4f, want ~0.012 (сильно порезан)", polyOwn)
	}
	chHex := pickRateN(t, chaosMob, hexIdx, N)
	if chHex < 0.70 || chHex > 0.76 {
		t.Fatalf("Chaos-моб: Hexagon = %.4f, want ~0.73 (основной дроп)", chHex)
	}

	// Chaos-моб: дроп никогда не ниже Pentagon (4).
	for i := 0; i < N; i++ {
		if r := pickDropRarity(chaosMob); r < 4 {
			t.Fatalf("Chaos-моб дропнул ниже Pentagon: %d", r)
		}
	}
}

// pickRateN оценивает частоту выпадения редкости want у моба редкости mobRar.
func pickRateN(t *testing.T, mobRar, want, n int) float64 {
	t.Helper()
	c := 0
	for i := 0; i < n; i++ {
		if pickDropRarity(mobRar) == want {
			c++
		}
	}
	return float64(c) / float64(n)
}

// makeDropPetal создаёт петаль нужного вида (по мобу) с владельцем.
func TestMakeDropPetalOwnerAndKind(t *testing.T) {
	mobTypesInZone = [][]string{
		{"square"}, {"triangle"}, {"dot"}, {"square"}, {"square"}, {"square"}, {"square"}, {"square"},
		{"square"}, {"square"}, {"square"}, {"square"}, {"square"}, {"square"}, {"square"}, {"square"},
	}
// Square -> cube, Triangle -> blade, Dot -> orb.
	m := &Mob{Type: "dot", Rarity: 0, X: 100, Y: 100, Zone: 0}
	initWorld()
	g := makeDropPetal(m, "playerX", 1)
	if g.Kind != "orb" {
		t.Fatalf("Dot mob drop kind = %q, want orb", g.Kind)
	}
	if g.OwnerID != "playerX" {
		t.Fatalf("owner = %q, want playerX", g.OwnerID)
	}
	if g.Rarity != 0 && g.Rarity != 1 {
		t.Fatalf("common mob drop rarity = %d, want 0..1", g.Rarity)
	}

	m2 := &Mob{Type: "square", Rarity: 0, X: 5, Y: 5, Zone: 0}
	g2 := makeDropPetal(m2, "p2", 2)
if g2.Kind != "cube" {
		t.Fatalf("Square mob drop kind = %q, want cube", g2.Kind)
	}

	m3 := &Mob{Type: "triangle", Rarity: 0, X: 5, Y: 5, Zone: 0}
	g3 := makeDropPetal(m3, "p3", 3)
	if g3.Kind != "blade" {
		t.Fatalf("Triangle mob drop kind = %q, want blade", g3.Kind)
	}
}

// awardMobKill: анти-лутстил — топ-4 дамагеры получают дроп ТОЛЬКО если нанесли
// ≥5% от MaxHP моба. Моб с MaxHP 1000 (порог 50): p1..p3 выше порога, p4 = 40
// (4%) — хоть и в топ-4, дропа не получает.
func TestAwardMobKillTop4Drops(t *testing.T) {
	r := newTestRoom()
	r.gpIDSeq = 1000
	for _, id := range []string{"p1", "p2", "p3", "p4", "p5"} {
		r.players[id] = &Player{State: PlayerState{ID: id, Name: id, Level: 1, Zone: 0}, hotbars: [2][]PetalDef{{}, {}}}
		r.players[id].State.XP = 0
	}
	m := &Mob{ID: "m1", Type: "square", Rarity: 0, X: 50, Y: 50, Zone: 0,
		Damage: map[string]float64{"p1": 100, "p2": 80, "p3": 60, "p4": 40, "p5": 20},
		MaxHP:  1000}
	m.KillerID = "p1"
	r.awardMobKill(m)

	// Обычная редкость (Point): порог 25%. Никто не набрал (макс 10%) —
	// fallback: лут уходит ОДНОМУ топ-дамагеру (p1).
	owners := map[string]bool{}
	for _, g := range r.gp {
		owners[g.OwnerID] = true
	}
	if len(r.gp) != 1 {
		t.Fatalf("drops = %d, want 1 (fallback: топ-1 дамагер p1)", len(r.gp))
	}
	if !owners["p1"] {
		t.Fatalf("expected drop for top damager p1")
	}
}

// Обычная редкость (Point): порог 25% — все, кто набрал, получают дроп.
func TestAwardMobKill25PercentThreshold(t *testing.T) {
	r := newTestRoom()
	r.gpIDSeq = 2000
	for _, id := range []string{"p1", "p2", "p3", "p4"} {
		r.players[id] = &Player{State: PlayerState{ID: id, Name: id, Level: 1, Zone: 0}, hotbars: [2][]PetalDef{{}, {}}}
	}
	m := &Mob{ID: "m1", Type: "square", Rarity: 4, X: 50, Y: 50, Zone: 0,
		Damage: map[string]float64{"p1": 400, "p2": 260, "p3": 250, "p4": 200},
		MaxHP:  1000}
	m.KillerID = "p1"
	r.awardMobKill(m)
	owners := map[string]bool{}
	for _, g := range r.gp {
		owners[g.OwnerID] = true
	}
	if len(r.gp) != 3 || !owners["p1"] || !owners["p2"] || !owners["p3"] {
		t.Fatalf("drops = %d (%v), want 3: p1 (40%%), p2 (26%%), p3 ровно 25%% включительно; p4 20%% — нет", len(r.gp), owners)
	}
}

// Глубокая редкость (Hexagon+): порог 5%; если НИКТО не набрал — дроп уходит
// топ-4 дамагерам без порога (толпа по 3-4%).
func TestAwardMobKillBossFallbackTop4(t *testing.T) {
	r := newTestRoom()
	r.gpIDSeq = 3000
	for _, id := range []string{"p1", "p2", "p3", "p4", "p5", "p6"} {
		r.players[id] = &Player{State: PlayerState{ID: id, Name: id, Level: 1, Zone: 0}, hotbars: [2][]PetalDef{{}, {}}}
	}
	m := &Mob{ID: "m1", Type: "square", Rarity: 5, X: 50, Y: 50, Zone: 0,
		Damage: map[string]float64{"p1": 40, "p2": 35, "p3": 30, "p4": 25, "p5": 20, "p6": 10},
		MaxHP:  1000}
	m.KillerID = "p1"
	r.awardMobKill(m)
	owners := map[string]bool{}
	for _, g := range r.gp {
		owners[g.OwnerID] = true
	}
	if len(r.gp) != 4 {
		t.Fatalf("drops = %d, want 4 (fallback топ-4 для Hexagon+)", len(r.gp))
	}
	for _, want := range []string{"p1", "p2", "p3", "p4"} {
		if !owners[want] {
			t.Fatalf("expected drop for %s", want)
		}
	}
	if owners["p5"] || owners["p6"] {
		t.Fatalf("p5/p6 не в топ-4 — дропа быть не должно")
	}
}

// Глубокая редкость: обычный путь — все, кто набрал 5%+.
func TestAwardMobKillBossFivePercent(t *testing.T) {
	r := newTestRoom()
	r.gpIDSeq = 4000
	for _, id := range []string{"p1", "p2", "p3", "p4"} {
		r.players[id] = &Player{State: PlayerState{ID: id, Name: id, Level: 1, Zone: 0}, hotbars: [2][]PetalDef{{}, {}}}
	}
	m := &Mob{ID: "m1", Type: "square", Rarity: 5, X: 50, Y: 50, Zone: 0,
		Damage: map[string]float64{"p1": 300, "p2": 60, "p3": 50, "p4": 40},
		MaxHP:  1000}
	m.KillerID = "p1"
	r.awardMobKill(m)
	owners := map[string]bool{}
	for _, g := range r.gp {
		owners[g.OwnerID] = true
	}
	if len(r.gp) != 3 || !owners["p1"] || !owners["p2"] || !owners["p3"] {
		t.Fatalf("drops = %d (%v), want 3 (p1..p3 >=5%%, p3 ровно 5%% включительно; p4 4%% — нет)", len(r.gp), owners)
	}
}

// Попадание в топ-4 без прохождения порога не влияет на самого убийцу:
// опыт за кил убийца получает ВСЕГДА (независимо от 5%).
func TestAwardMobKillKillerXPIndependentOfThreshold(t *testing.T) {
	r := newTestRoom()
	r.players["p1"] = &Player{State: PlayerState{ID: "p1", Name: "p1", Level: 1, Zone: 0}, hotbars: [2][]PetalDef{{}, {}}}
	m := &Mob{ID: "m1", Type: "square", Rarity: 0, X: 50, Y: 50, Zone: 0,
		Damage: map[string]float64{"p1": 1, "p2": 1000}, MaxHP: 1000}
	m.KillerID = "p1" // убил последним (1 урона), но взял дроп ниже порога — XP всё равно его
	r.awardMobKill(m)
	if r.players["p1"].State.XP <= 0 {
		t.Fatalf("убийца ниже 5%% порога не получил XP: %v", r.players["p1"].State.XP)
	}
}

// Боссы: базовый дроп гарантирован допущенному дамагеру; редкость дропа у
// Fractal/Chaos-моба не выше их тира/максимума, а Chaos-столбец не выпадает.
func TestAwardMobKillBossBonus(t *testing.T) {
	// Сплав Actal зовёт scaleMobForRarity -> limitMobSizeForZone — нужен мир.
	initWorld()
	r := newTestRoom()
	r.gpIDSeq = 1
	r.players["p1"] = &Player{State: PlayerState{ID: "p1", Name: "p1", Level: 1, Zone: 0}, hotbars: [2][]PetalDef{{}, {}}}
	m := &Mob{ID: "f", Type: "actal", Rarity: rarityFractal, X: 0, Y: 0, Zone: 0, HP: 1, MaxHP: 1,
		Damage: map[string]float64{"p1": 1}}
	m.KillerID = "p1"
	r.awardMobKill(m)
	if len(r.gp) < 1 {
		t.Fatalf("boss моб не дал дроп")
	}
	for _, g := range r.gp {
		if g.OwnerID != "p1" || g.Rarity > rarityFractal {
			t.Fatalf("плохой дроп босса: owner=%s rarity=%d", g.OwnerID, g.Rarity)
		}
	}

	// Chaos-моб: дроп не может быть Chaos (7).
	r.gp = nil
	r.gpIDSeq = 1
	mc := &Mob{ID: "c", Type: "entropy", Rarity: rarityChaos, X: 0, Y: 0, Zone: 0, HP: 1, MaxHP: 1,
		Damage: map[string]float64{"p1": 1}}
	mc.KillerID = "p1"
	r.awardMobKill(mc)
	if len(r.gp) < 1 {
		t.Fatalf("Chaos моб не дал дроп")
	}
	for _, g := range r.gp {
		if g.Rarity >= rarityChaos {
			t.Fatalf("Chaos моб дропнул Chaos (r=%d)", g.Rarity)
		}
	}
}

// buildState: приватный дроп (владелец != "") виден только владельцу.
func TestBuildStateOwnerFiltering(t *testing.T) {
	r := newTestRoom()
	r.players = map[string]*Player{
		"owner": {State: PlayerState{ID: "owner", Name: "O", Zone: 0}, hotbars: [2][]PetalDef{{}, {}}},
		"other": {State: PlayerState{ID: "other", Name: "Z", Zone: 0}, hotbars: [2][]PetalDef{{}, {}}},
	}
	r.gp = []GroundPetal{
		{ID: "pub1", Kind: "oval", Rarity: 0, X: 1, Y: 1, Color: 0xecf0f1, Zone: 0},
		{ID: "priv1", Kind: "rock", Rarity: 0, X: 2, Y: 2, Color: 0xecf0f1, Zone: 0, OwnerID: "owner"},
	}

	ownerState := r.buildState("owner")
	var om StateMsg
	json.Unmarshal(ownerState, &om)
	ids := map[string]bool{}
	for _, g := range om.Ground {
		ids[g.ID] = true
	}
	if !ids["pub1"] || !ids["priv1"] {
		t.Fatalf("owner must see pub+priv, got %v", ids)
	}

	otherState := r.buildState("other")
	var osm StateMsg
	json.Unmarshal(otherState, &osm)
	oIDs := map[string]bool{}
	for _, g := range osm.Ground {
		oIDs[g.ID] = true
	}
	if !oIDs["pub1"] {
		t.Fatalf("other must see public petal")
	}
	if oIDs["priv1"] {
		t.Fatalf("other must NOT see owner's private petal")
	}
}

// pickupPetals: приватный дроп подбирает только его владелец.
func TestPickupOwnerOnly(t *testing.T) {
	r := newTestRoom()
	// Дроп принадлежит "owner"; рядом с ним стоит "other" (в радиусе подбора).
	// Owner далеко — не может подобрать в этой итерации.
	r.players["owner"] = &Player{State: PlayerState{ID: "owner", Zone: 0, X: 1000, Y: 1000}, hotbars: [2][]PetalDef{{}, {}}}
	r.players["other"] = &Player{State: PlayerState{ID: "other", Zone: 0, X: 100, Y: 100}, hotbars: [2][]PetalDef{{}, {}}}
	r.gp = []GroundPetal{{ID: "priv", Kind: "oval", Rarity: 0, X: 100, Y: 100, Size: 8, HP: 5, DMG: 3, Color: 0xecf0f1, Zone: 0, OwnerID: "owner"}}

	// "other" рядом с чужим дропом — подобрать не может.
	r.pickupPetals(1.0/20.0)
	if len(r.gp) != 1 {
		t.Fatalf("other picked owner's drop")
	}

	// Теперь подводим владельца к своему дропу, other убираем из зоны.
	r.players["owner"].State.X = 100
	r.players["owner"].State.Y = 100
	r.players["other"].State.Zone = 1
	r.pickupPetals(1.0/20.0)
	if len(r.gp) != 0 {
		t.Fatalf("owner could not pick own drop")
	}
}

// Капы редких мобов: понижение редкости (Chaos->Fractal, Fractal->Hexagon)
// пересчитывает статы; хаос активен глобально, фрактал — по зоне.
func TestRarityCaps(t *testing.T) {
	initWorld()
	m := Mob{Type: "dot", Zone: 0}
	scaleMobForRarity(&m, rarityChaos)
	cHP, cR := m.HP, m.Radius
	scaleMobForRarity(&m, rarityFractal)
	if m.HP >= cHP || m.Radius >= cR {
		t.Fatalf("downgrade did not reduce stats: hp %v->%v, r %v->%v", cHP, m.HP, cR, m.Radius)
	}
	var empty Room
	if empty.chaosActive() {
		t.Fatalf("empty room has chaos")
	}
	r1 := Room{mobs: []Mob{{Type: "dot", Rarity: rarityChaos, HP: 10}}}
	if !r1.chaosActive() {
		t.Fatalf("chaos not detected in room")
	}
	r2 := Room{mobs: []Mob{{Type: "dot", Rarity: rarityFractal, HP: 10, Zone: 3}}}
	if !r2.fractalInZone(3) {
		t.Fatalf("fractal in zone 3 not detected")
	}
	if r2.fractalInZone(4) {
		t.Fatalf("fractal detected in wrong zone 4")
	}
	fh := Room{mobs: []Mob{{Type: "dot", Rarity: 5, HP: 10, Zone: 1}}}
	if fh.hexInZone(1) != 1 {
		t.Fatalf("hexInZone = %d, want 1", fh.hexInZone(1))
	}
	if fh.hexInZone(0) != 0 {
		t.Fatalf("hexInZone wrong zone = %d, want 0", fh.hexInZone(0))
	}
}

// Уведомление о спавне Chaos/Fractal-моба приходит из ЛЮБОЙ зоны (оба возможны
// в верхней полосе каждого зона): Chaos — редкость 7, Fractal — 6.
func TestSuperSpawnNotification(t *testing.T) {
	initWorld()
	last := len(zones) - 1
	tri := Mob{Type: "triangle", Rarity: rarityChaos}
	if txt := superSpawnText(0, tri); txt != "A Chaos Triangle has spawned!" {
		t.Fatalf("first-zone chaos msg = %q", txt)
	}
	if txt := superSpawnText(last, tri); txt != "A Chaos Triangle has spawned!" {
		t.Fatalf("super zone triangle msg = %q", txt)
	}
	// Fractal (6) даёт своё уведомление.
	fr := Mob{Type: "square", Rarity: rarityFractal}
	if txt := superSpawnText(last, fr); txt != "A Fractal Square has spawned!" {
		t.Fatalf("fractal rarity msg = %q", txt)
	}
	// Обычная редкость не даёт уведомления.
	sq := Mob{Type: "square", Rarity: 2}
	if txt := superSpawnText(last, sq); txt != "" {
		t.Fatalf("non-super rarity gave msg %q", txt)
	}
}

// Смерть Golden Chaos-моба: анонс обязан содержать слово Golden.
func TestGoldenChaosDefeatLabel(t *testing.T) {
	r := newTestRoom()
	ce := &Client{send: make(chan []byte, 16)}
	r.clients["killer"] = ce
	r.players["killer"] = &Player{
		State:    PlayerState{ID: "killer", Name: "Zhenya", Zone: 8},
		hotbars:  [2][]PetalDef{{}, {}},
		inventory: []InvStack{},
	}
	m := &Mob{
		ID: "mm", Type: "arrow", Rarity: rarityChaos, Variant: "gold",
		X: 100, Y: 100, Zone: 8, MaxHP: 1000,
		Damage: map[string]float64{"killer": 99999}, KillerID: "killer",
	}
	r.awardMobKill(m)

	for {
		select {
		case msg := <-ce.send:
			s := string(msg)
			if !strings.Contains(s, "super") {
				continue
			}
			if !strings.Contains(s, "A Golden Chaos Arrow has been defeated by Zhenya!") {
				t.Fatalf("defeat msg = %s, want Golden prefix", s)
			}
			return
		default:
			t.Fatal("no defeat announcement broadcast")
		}
	}
}

// Подбор Fractal-дропа шлёт всемирный анонс (как анонс крафта); обычный дроп — нет.
func TestPickupFractalBroadcast(t *testing.T) {
	r := newTestRoom()
	ce := &Client{send: make(chan []byte, 16)}
	r.clients["owner"] = ce
	r.players["owner"] = &Player{
		State:    PlayerState{ID: "owner", Name: "Fedor", Zone: 0, X: 100, Y: 100},
		hotbars:  [2][]PetalDef{{}, {}},
		inventory: []InvStack{},
	}
	r.gp = []GroundPetal{
		{ID: "fr", Kind: "arrow", Rarity: 6, X: 100, Y: 100, Size: 8, HP: 5, DMG: 3, Color: 0x9b59b6, Zone: 0, OwnerID: "owner"},
		{ID: "pl", Kind: "oval", Rarity: 0, X: 95, Y: 95, Size: 8, HP: 5, DMG: 3, Color: 0xecf0f1, Zone: 0, OwnerID: "owner"},
	}
	r.pickupPetals(1.0 / 20.0)
	r.pickupPetals(1.0 / 20.0)
	if len(r.gp) != 0 {
		t.Fatalf("drops not picked: %d left", len(r.gp))
	}

	var foundFrac, foundPlain bool
	for {
		select {
		case msg := <-ce.send:
			s := string(msg)
			if !strings.Contains(s, `"type":"sys"`) {
				continue
			}
			if strings.Contains(s, "craftFractal") && strings.Contains(s, "Fedor has found a Fractal") {
				foundFrac = true
			}
			if strings.Contains(s, "craftPoint") {
				foundPlain = true
			}
		default:
			goto drained
		}
	}
drained:
	if !foundFrac {
		t.Fatalf("no craftFractal announcement for Fractal pickup")
	}
	if foundPlain {
		t.Fatalf("unexpected announcement for plain pickup")
	}
}

