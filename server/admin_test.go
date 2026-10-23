package main

import (
	"math"
	"testing"
)

// После отключения авто-спавна петалей на полу мир создаётся без них:
// петали появляются только как дроп с убитых мобов.
func TestWorldHasNoAutoGroundPetals(t *testing.T) {
	initWorld()
	r := NewRoom(20)
	if len(r.mobs) == 0 {
		t.Fatal("нет мобов при создании комнаты")
	}
	if len(r.gp) != 0 {
		t.Fatalf("нашлось %d петалей на полу без дропа (должно быть 0)", len(r.gp))
	}
}

func TestIsAdmin(t *testing.T) {
	if !isAdmin(adminNick) {
		t.Fatalf("isAdmin(%q) should be true", adminNick)
	}
	if isAdmin("not_admin") {
		t.Fatal("isAdmin of non-admin nick should be false")
	}
}

// makeAdminMob создаёт моба заданного вида и редкости, прижав к границам зоны.
func TestMakeAdminMob(t *testing.T) {
	initWorld()
	m := makeAdminMob("square", 4, 0, -5000, -5000, 99, "nothing")
	if m.Type != "square" {
		t.Fatalf("type = %q", m.Type)
	}
	if m.Rarity != 4 {
		t.Fatalf("rarity = %d", m.Rarity)
	}
	z := zones[0]
	if m.X < z.X || m.X > z.X+z.W || m.Y < z.Y || m.Y > z.Y+z.H {
		t.Fatalf("mob outside zone: %v %v in %v", m.X, m.Y, z)
	}
	if m.MaxHP <= 0 {
		t.Fatalf("non-positive HP: %v", m.MaxHP)
	}
}

// Админ-боссы той же редкости, что и реальные: HP считается единой формулой
// (scaleMobForRarity), включая множители Fractal ×10 / Chaos ×32, и босс
// получает орбитальные петали — иначе призванный админом босс слабее настоящего.
func TestMakeAdminMobBossStatsMatchReal(t *testing.T) {
	initWorld()

	type tok struct{ kind string; rar int; mult float64 }
	cases := []tok{
		{"actal", rarityFractal, fractalHPMult},
		{"entropy", rarityChaos, chaosHPMult},
	}
	for _, c := range cases {
		m := makeAdminMob(c.kind, c.rar, 7, 0, 0, 1, "nothing")
		mt := mobTypes[c.kind]
		wantHP := mt.BaseHP * rarityDanger[c.rar] * mobHPScale * 2.0 * c.mult
		if math.Abs(m.MaxHP-wantHP) > 1 {
			t.Fatalf("%s r%d: MaxHP=%v, want %v (множитель босса)", c.kind, c.rar, m.MaxHP, wantHP)
		}
		if len(m.petals) == 0 {
			t.Fatalf("%s r%d: админ-босс без орбитальных петалей", c.kind, c.rar)
		}
	}
}
