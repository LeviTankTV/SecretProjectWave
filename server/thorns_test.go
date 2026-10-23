package main

import "testing"

// TestThornsFlags — петаль-«шип» отмечена Thorns, обычные виды — нет.
func TestThornsFlags(t *testing.T) {
	if !petalKinds["thorns"].Thorns {
		t.Errorf("thorns должен иметь Thorns=true (механика «размен»)")
	}
	for _, kind := range []string{"oval", "rock", "ball", "viper", "blade", "entropy", "void"} {
		if petalKinds[kind].Thorns {
			t.Errorf("%s не должен быть шипастым", kind)
		}
	}
}

// TestThornsReflectMob — шип отражает часть износа обратно мобу: за контакт моб
// получает pet.DMG×petalDmgMult + m.DMG*0.3*thornsReflectRatio, и весь этот урон
// идёт игроку в Damage-карту (для топ-дамагеров/дропа).
func TestThornsReflectMob(t *testing.T) {
	r := &Room{
		players: map[string]*Player{},
		clients: map[string]*Client{},
	}
	r.players["p1"] = &Player{
		State: PlayerState{ID: "p1", Name: "T", Level: 50, XP: 0, X: 0, Y: 0, Size: playerSize, Zone: 0},
	}
	r.players["p1"].petals = append(r.players["p1"].petals, Petal{
		ID: "petA", Kind: "thorns", HP: 100, MaxHP: 100, DMG: 7, Size: 8,
		Angle: 0, Dist: 50,
	})
	r.mobs = append(r.mobs, Mob{
		ID: "m1", Type: "dot", Rarity: 0,
		HP: 1000, MaxHP: 1000, DMG: 10, Speed: 10, Radius: 18, Zone: 0,
		Damage: map[string]float64{},
		X:      50, Y: 0,
	})

	r.fightPetalsVsMobs(0.016)

	m := &r.mobs[0]
	ref := 10.0 * 0.3 * thornsReflectRatio
	if m.HP != 1000-7*petalDmgMult-ref {
		t.Errorf("HP моба = %v, want %v (урон %v + отражение %v)", m.HP, 1000-7*petalDmgMult-ref, 7*petalDmgMult, ref)
	}
	if m.Damage["p1"] != 7*petalDmgMult+ref {
		t.Errorf("Damage[%q] = %v, want %v", "p1", m.Damage["p1"], 7*petalDmgMult+ref)
	}
	pet := &r.players["p1"].petals[0]
	if pet.HP != 100-3.0 {
		t.Errorf("HP шипа после износа = %v, want 97", pet.HP)
	}
}

// TestThornsNoReflectNonThorns — обычный (не шипастый) петаль НЕ отражает урон.
func TestThornsNoReflectNonThorns(t *testing.T) {
	r := &Room{
		players: map[string]*Player{},
		clients: map[string]*Client{},
	}
	r.players["p1"] = &Player{
		State: PlayerState{ID: "p1", Name: "T", Level: 50, XP: 0, X: 0, Y: 0, Size: playerSize, Zone: 0},
	}
	r.players["p1"].petals = append(r.players["p1"].petals, Petal{
		ID: "petA", Kind: "oval", HP: 100, MaxHP: 100, DMG: 4, Size: 8,
		Angle: 0, Dist: 50,
	})
	r.mobs = append(r.mobs, Mob{
		ID: "m1", Type: "dot", Rarity: 0,
		HP: 1000, MaxHP: 1000, DMG: 10, Speed: 10, Radius: 18, Zone: 0,
		Damage: map[string]float64{},
		X:      50, Y: 0,
	})

	r.fightPetalsVsMobs(0.016)

	m := &r.mobs[0]
	if m.HP != 1000-4*petalDmgMult {
		t.Errorf("HP моба = %v, want %v (без отражения)", m.HP, 1000-4*petalDmgMult)
	}
	if m.Damage["p1"] != 4*petalDmgMult {
		t.Errorf("Damage[%q] = %v, want %v", "p1", m.Damage["p1"], 4*petalDmgMult)
	}
}

// TestThornsReflectBossPetal — шип «колет» и орбитальную петаль босса: за контакт
// петаль босса получает pet.DMG×petalDmgMult + bp.DMG*0.3*thornsReflectRatio.
func TestThornsReflectBossPetal(t *testing.T) {
	r := &Room{
		players: map[string]*Player{},
		clients: map[string]*Client{},
	}
	const (
		bossX = 100.0
		bossY = 100.0
		bossR = 30.0
	)
	bpX := bossX + bossR*mobPetalOrbitMult
	r.players["p1"] = &Player{
		State: PlayerState{ID: "p1", Name: "T", Level: 50, XP: 0, X: bpX - 50, Y: bossY, Size: playerSize, Zone: 0},
	}
	r.players["p1"].petals = append(r.players["p1"].petals, Petal{
		ID: "petA", Kind: "thorns", HP: 100, MaxHP: 100, DMG: 7, Size: 8,
		Angle: 0, Dist: 50,
	})
	r.mobs = append(r.mobs, Mob{
		ID: "m1", Type: "actal", Rarity: len(rarityNames) - 2,
		HP: 1000, MaxHP: 1000, DMG: 10, Speed: 10, Radius: bossR, Zone: 0,
		Damage: map[string]float64{},
		X:      bossX, Y: bossY,
	})
	r.mobs[0].petals = []BossPetal{{Idx: 0, Angle: 0, HP: 50, MaxHP: 50, DMG: 5, Size: 31.2}}

	r.fightPetalsVsMobs(0.016)

	bp := &r.mobs[0].petals[0]
	ref := 5.0 * 0.3 * thornsReflectRatio
	// Tier-гейт редкости: Fractal-босса (rarity 6) обычная (rarity 0) петаль
	// бьёт в 10 раз слабее; отражённый износ (по урону босса) не гейтится.
	gated := 7 * petalDmgMult * petalVsMobMult(r.players["p1"].petals[0].Rarity, r.mobs[0].Rarity)
	if bp.HP != 50-gated-ref {
		t.Errorf("HP петеля босса = %v, want %v (урон %v + отражение %v)", bp.HP, 50-gated-ref, gated, ref)
	}
	pet := &r.players["p1"].petals[0]
	if pet.HP != 100-1.5 {
		t.Errorf("HP шипа после износа = %v, want 98.5", pet.HP)
	}
}
