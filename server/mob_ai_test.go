package main

import (
	"testing"
)

// Shield+Pentagon: во время патруля, если рядом есть сосед той же группы,
// моб наследует общее направление (двигаются компанией в одном направлении).
func TestUpdateMobGroupDirection(t *testing.T) {
	initWorld()
	// Два щита рядом и один далеко — щиты должны смотреть в одну сторону,
	// как соседи, а далёкий (вне 600) остаётся со своим направлением.
	r := newTestRoom()
	r.mobs = []Mob{
		{ID: "sh1", Type: "shield", Zone: 0, Rarity: 0, HP: 1, X: 4000, Y: 2500,
			Speed: 10, Radius: 10, dirX: 1, dirY: 0, state: "patrol", moveTimer: 5},
		{ID: "sh2", Type: "shield", Zone: 0, Rarity: 0, HP: 1, X: 4020, Y: 2500,
			Speed: 10, Radius: 10, dirX: -1, dirY: 0, state: "patrol", moveTimer: 5},
		{ID: "sh3", Type: "shield", Zone: 0, Rarity: 0, HP: 1, X: 5000, Y: 2500,
			Speed: 10, Radius: 10, dirX: 0, dirY: 1, state: "patrol", moveTimer: 5},
	}
	// Одна итерация: sh1 (индекс 0) — вожак; sh2 рядом идёт в точку позади вожака
// (занял позицию формирования), а не копирует курс. sh3 далеко — не тронут.
	r.updateMob(&r.mobs[0], map[string]*Player{}, 1.0/60)
	r.updateMob(&r.mobs[1], map[string]*Player{}, 1.0/60)
	if r.mobs[1].dirX == 0 && r.mobs[1].dirY == 0 {
		t.Fatalf("follower should steer toward formation point behind leader")
	}
	// Вожак перед ш2: (4000,2500); его задняя точка (3945,2500); ш2 в (4020,2500) →
	// ведомый должен идти влево (dirX<0).
	if r.mobs[1].dirX >= 0 {
		t.Fatalf("follower should steer left to formation point, got dirX=%v", r.mobs[1].dirX)
	}
	// Далёкий моб не затронут компанией.
	if r.mobs[2].dirX != 0 || r.mobs[2].dirY != 1 {
		t.Fatalf("distant shield should keep its own direction")
	}
}

// Block/Hive со скоростью 0 не двигаются даже при ненулевом направлении.
func TestUpdateMobZeroSpeedImmovable(t *testing.T) {
	initWorld()
	r := newTestRoom()
	r.mobs = []Mob{
		{ID: "h1", Type: "hive", Zone: 0, Rarity: 0, HP: 1,
			X: 500, Y: 500, Speed: 0, Radius: 10, dirX: 1, dirY: 1, state: "patrol", moveTimer: 5},
	}
	startX, startY := r.mobs[0].X, r.mobs[0].Y
	for i := 0; i < 600; i++ {
		r.updateMob(&r.mobs[0], map[string]*Player{}, 1.0/60)
	}
	// Timer не сбрасывается до 0, моб никуда не движется (Speed = 0).
	if r.mobs[0].X != startX || r.mobs[0].Y != startY {
		t.Fatalf("immobile mob moved: (%v,%v) -> (%v,%v)", startX, startY, r.mobs[0].X, r.mobs[0].Y)
	}
}

// actal (редкость >= 1) при смерти спавнит 2-3 мини-копии с уменьшенными статами.
func TestActalFractalMinis(t *testing.T) {
	initWorld()
	r := newTestRoom()
	// Петали-дроп всё равно создадутся; мобы нужны для проверки мини-копий.
	r.mobs = []Mob{}
	m := &Mob{ID: "m1", Type: "actal", Rarity: 1, X: 100, Y: 100, Zone: 0,
		HP: 300, MaxHP: 300, Damage: map[string]float64{"p1": 50}, KillerID: "p1"}
	r.mobIDSeq = 500
	r.awardMobKill(m)

	minis := 0
	for _, mob := range r.mobs {
		if mob.Type == "actal" && len(mob.ID) >= 7 && mob.ID[:8] == "fractal_" {
			minis++
			if mob.HP > m.MaxHP {
				t.Fatalf("mini HP %.2f bigger than parent %.2f", mob.HP, m.MaxHP)
			}
		}
	}
	if minis < 2 || minis > 3 {
		t.Fatalf("actal spawned %d minis, want 2-3", minis)
	}
}

// Golden-«рассадники» наследуют Golden своим призывам: Golden-Hive призывает
// ЗОЛОТЫХ мобов, Golden-Actal распадается на ЗОЛОТЫХ мини-копий.
func TestGoldenSpawnerLegacy(t *testing.T) {
	initWorld()
	r := newTestRoom()

	// Hive-summon: призыв зоны становится Golden.
	gh := &Mob{ID: "gh1", Type: "hive", Rarity: 3, X: 200, Y: 200, Zone: 5,
		HP: 500, MaxHP: 500, Variant: "gold", Radius: 30}
	before := len(r.mobs)
	hiveSummonMob(r, gh)
	if len(r.mobs) != before+1 {
		t.Fatalf("hive summon didn't add a mob")
	}
	if r.mobs[before].Variant != "gold" {
		t.Fatalf("hive summon should be Golden, got %+v", r.mobs[before])
	}

	// Actal minis: распад на ЗОЛОТЫХ мини-копий.
	r.mobs = r.mobs[:before]
	ga := &Mob{ID: "ga1", Type: "actal", Rarity: 6, X: 300, Y: 300, Zone: 0,
		HP: 500, MaxHP: 500, Variant: "gold", Damage: map[string]float64{"p1": 50}, KillerID: "p1"}
	before2 := len(r.mobs)
	r.awardMobKill(ga)
	goldMinis := 0
	for _, mob := range r.mobs[before2:] {
		if mob.Type == "actal" && mob.Variant == "gold" {
			goldMinis++
		}
	}
	if goldMinis < 2 || goldMinis > 3 {
		t.Fatalf("golden actal spawned %d golden minis, want 2-3", goldMinis)
	}
}

// buildState: редкость петаля на земле сериализуется именами редкостей
// (Hexagon/Fractal/Chaos), а не именами зон (Square/Polygon).
func TestGroundPetalRarityName(t *testing.T) {
	initWorld()
	r := newTestRoom()
	r.players["p"] = &Player{State: PlayerState{ID: "p", Name: "P", Zone: 0}, hotbars: [2][]PetalDef{{}, {}}}
	r.gp = []GroundPetal{
		{ID: "g6", Kind: "oval", Rarity: 6, X: 1, Y: 1, Color: 0xecf0f1, Zone: 0},
		{ID: "g7", Kind: "oval", Rarity: 7, X: 2, Y: 2, Color: 0xecf0f1, Zone: 0},
	}
	state := r.buildState("p")
	if string(state) == "" {
		t.Fatalf("empty state")
	}
	// Ищем имена редкостей в сериализованном JSON.
	s := string(state)
	if containsStr(s, `"Polygon"`) || containsStr(s, `"Hexagon"`) {
		t.Fatalf("ground petal rarity leaked zone-name: %s", s)
	}
	if !containsStr(s, `"Fractal"`) || !containsStr(s, `"Chaos"`) {
		t.Fatalf("ground petal rarity missing rarity-name: %s", s)
	}
}

// Если ударили одного из стаи Pentagon/Shield, вся ближайшая стая агрится на
// обидчика (aggroTo), а далёкие и не-стайные мобы — нет.
func TestAggroPack(t *testing.T) {
	initWorld()
	r := newTestRoom()
	r.mobs = []Mob{
		{ID: "p1", Type: "pentagon", Zone: 0, Rarity: 0, HP: 1, X: 100, Y: 100,
			Speed: 10, Radius: 10, state: "patrol"},
		{ID: "s1", Type: "shield", Zone: 0, Rarity: 0, HP: 1, X: 150, Y: 110,
			Speed: 10, Radius: 10, state: "patrol"},
		{ID: "p2", Type: "pentagon", Zone: 0, Rarity: 0, HP: 1, X: 2500, Y: 100,
			Speed: 10, Radius: 10, state: "patrol"},
		{ID: "sq", Type: "square", Zone: 0, Rarity: 0, HP: 1, X: 120, Y: 100,
			Speed: 10, Radius: 10, state: "patrol"},
	}
	r.aggroPack(&r.mobs[0], "attacker1")
	if r.mobs[1].aggroTo != "attacker1" {
		t.Fatalf("nearby shield not aggroed on attacker")
	}
	if r.mobs[2].aggroTo != "" {
		t.Fatalf("distant pentagon should not aggro")
	}
	if r.mobs[3].aggroTo != "" {
		t.Fatalf("non-pack mob should not aggro")
	}
	if r.mobs[0].aggroTo != "" {
		t.Fatalf("the hit mob itself shouldn't need aggroTo")
	}
}

// FragWall (стена) двигается строго по осям: диагональное направление
// сглаживается до ближайшей горизонтали/вертикали.
func TestFragWallAxisMovement(t *testing.T) {
	initWorld()
	r := newTestRoom()
	r.mobs = []Mob{
		{ID: "fw1", Type: "fragwall", Zone: 0, Rarity: 0, HP: 1,
			X: 500, Y: 500, Speed: 10, Radius: 10, dirX: 1, dirY: 0.4, state: "patrol",
			moveTimer: 9999, Damage: map[string]float64{}},
	}
	r.updateMob(&r.mobs[0], map[string]*Player{}, 1.0/60)
	m := r.mobs[0]
	if m.dirY != 0 {
		t.Fatalf("fragwall should snap to horizontal axis, got dir=(%v,%v)", m.dirX, m.dirY)
	}
	if m.X <= 500 || m.Y != 500 {
		t.Fatalf("fragwall should move only along X, got pos=(%v,%v)", m.X, m.Y)
	}
}

func containsStr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}