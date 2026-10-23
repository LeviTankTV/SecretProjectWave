package main

import "testing"

// --- Cube: гравитационный толчок (усиленный отброс) ---

func TestCubePushKnockback(t *testing.T) {
	initWorld()
	r := &Room{players: map[string]*Player{}, clients: map[string]*Client{}}
	r.players["p1"] = &Player{
		State: PlayerState{ID: "p1", X: 4000, Y: 2500, Size: playerSize, Zone: 0},
	}
	r.players["p1"].petals = append(r.players["p1"].petals, Petal{
		Kind: "cube", HP: 100, MaxHP: 100, DMG: 5, Size: 8, Rarity: 0, Angle: 0, Dist: 50,
	})
	r.mobs = append(r.mobs, Mob{
		ID: "m1", Type: "dot", Rarity: 0, HP: 100, DMG: 2, Speed: 10, Radius: 18, Zone: 0,
		Damage: map[string]float64{},
		X:      4055, Y: 2500,
	})
	r.fightPetalsVsMobs(0.016)
	m := &r.mobs[0]
	// Куб (rarity 0) масса 1, моб масса 18: отброс = cubePushK * 1 / 18.
	wantX := 4055 + cubePushK*orbitPetalMass(r.players["p1"].petals[0])/mobMass(m)
	if m.X != wantX {
		t.Errorf("куб не толкнул моба: X=%v, want %v", m.X, wantX)
	}
}

// --- Spark: цепной урон по ближайшим врагам + линия-цепь ---

func TestSparkChain(t *testing.T) {
	r := &Room{players: map[string]*Player{}, clients: map[string]*Client{}}
	r.players["p1"] = &Player{
		State: PlayerState{ID: "p1", X: 0, Y: 0, Size: playerSize, Zone: 0},
	}
	r.players["p1"].petals = append(r.players["p1"].petals, Petal{
		Kind: "spark", HP: 100, MaxHP: 100, DMG: 12, Size: 8, Rarity: 0, Angle: 0, Dist: 50,
	})
	r.mobs = append(r.mobs,
		Mob{ID: "a", Type: "dot", Rarity: 0, HP: 100, DMG: 2, Radius: 18, Zone: 0, Damage: map[string]float64{}, X: 50, Y: 0},
		Mob{ID: "b", Type: "dot", Rarity: 0, HP: 100, DMG: 2, Radius: 18, Zone: 0, Damage: map[string]float64{}, X: 100, Y: 0},
		Mob{ID: "c", Type: "dot", Rarity: 0, HP: 100, DMG: 2, Radius: 18, Zone: 0, Damage: map[string]float64{}, X: 150, Y: 0},
	)
	r.fightPetalsVsMobs(0.016)
	// Основной удар по a, цепь по b и c (урон звена ×(1-sparkChainFalloff));
	// dmg = DMG × petalDmgMult.
	if r.mobs[0].HP != 100-12*petalDmgMult {
		t.Errorf("моб a не получил основной урон: HP=%v, want %v", r.mobs[0].HP, 100-12*petalDmgMult)
	}
	if r.mobs[1].HP != 100-12*petalDmgMult*(1-sparkChainFalloff) {
		t.Errorf("моб b не получил цепной урон: HP=%v, want %v", r.mobs[1].HP, 100-12*petalDmgMult*(1-sparkChainFalloff))
	}
	if r.mobs[2].HP != 100-12*petalDmgMult*(1-sparkChainFalloff)*(1-sparkChainFalloff) {
		t.Errorf("моб c не получил цепной урон: HP=%v, want %v", r.mobs[2].HP, 100-12*petalDmgMult*(1-sparkChainFalloff)*(1-sparkChainFalloff))
	}
	if len(r.chains) != 1 {
		t.Fatalf("линия-цепь не создана: chains=%d, want 1", len(r.chains))
	}
	if len(r.chains[0].Points) < 2 {
		t.Errorf("в цепи мало точек: %d", len(r.chains[0].Points))
	}
}

// --- Void: рывок по пробелу (тратит петаль, двигает игрока) ---

func TestVoidDash(t *testing.T) {
	// dashVoid клампит игрока по зонам на каждом шаге — нужен initWorld().
	initWorld()
	r := &Room{players: map[string]*Player{}, clients: map[string]*Client{}}
	r.players["p1"] = &Player{
		State: PlayerState{ID: "p1", X: 4000, Y: 2500, Size: playerSize, Zone: 0},
		lastDir: Vec{X: 1, Y: 0},
	}
	r.players["p1"].petals = append(r.players["p1"].petals, Petal{
		Kind: "void", HP: 100, MaxHP: 100, DMG: 12, Size: 8,
	})
	r.dashVoid("p1")
	p := r.players["p1"]
	if p.State.X != 4000+voidDashDist {
		t.Errorf("игрок не сделал рывок: X=%v, want %v", p.State.X, 4000+voidDashDist)
	}
	if p.petals[0].Recharge != voidCooldown {
		t.Errorf("void-петаль не ушёл в перезарядку: Recharge=%v, want %v", p.petals[0].Recharge, voidCooldown)
	}
}
