package main

import "testing"

// --- Флаги механик ---

func TestPetalMechFlags(t *testing.T) {
	cases := map[string]string{
		"orb":      "Regen",
		"bulwark":  "MaxHP",
		"hex":      "Magnet",
		"hiveling": "Split",
		"blade":    "Missile",
		"twirl":    "Twirl",
	}
	for kind, want := range cases {
		kn := petalKinds[kind]
		switch want {
		case "Regen":
			if !kn.Regen {
				t.Errorf("%s должен иметь Regen", kind)
			}
		case "MaxHP":
			if !kn.MaxHP {
				t.Errorf("%s должен иметь MaxHP", kind)
			}
		case "Magnet":
			if !kn.Magnet {
				t.Errorf("%s должен иметь Magnet", kind)
			}
		case "Split":
			if !kn.Split {
				t.Errorf("%s должен иметь Split", kind)
			}
		case "Missile":
			if !kn.Missile {
				t.Errorf("%s должен иметь Missile", kind)
			}
		case "Twirl":
			if !kn.Twirl {
				t.Errorf("%s должен иметь Twirl (разворот орбиты)", kind)
			}
		}
	}
	for _, kind := range []string{"oval", "rock", "ball", "viper", "entropy", "void"} {
		kn := petalKinds[kind]
		if kn.Regen || kn.MaxHP || kn.Magnet || kn.Split || kn.Missile {
			t.Errorf("%s не должен нести активную механику (Regen/MaxHP/Magnet/Split/Missile)", kind)
		}
	}
}

// --- Реген (orb) ---

func TestOrbRegen(t *testing.T) {
	r := &Room{players: map[string]*Player{}, clients: map[string]*Client{}}
	r.players["p1"] = &Player{
		State: PlayerState{ID: "p1", HP: 5, MaxHP: playerMaxHP, Zone: 0},
	}
	r.players["p1"].petals = append(r.players["p1"].petals, Petal{
		Kind: "orb", Rarity: 0, HP: 10, MaxHP: 10, Recharge: 0,
	})
	r.applyPetalPassives(r.players["p1"], 1.0)
	p := r.players["p1"]
	want := 5 + playerMaxHP*orbRegenFrac*rarityDanger[0]
	if p.State.HP != want {
		t.Errorf("реген orb: HP=%v, want %v", p.State.HP, want)
	}
}

func TestOrbRegenCapped(t *testing.T) {
	r := &Room{players: map[string]*Player{}, clients: map[string]*Client{}}
	r.players["p1"] = &Player{State: PlayerState{ID: "p1", HP: 9.5, MaxHP: playerMaxHP, Zone: 0}}
	r.players["p1"].petals = append(r.players["p1"].petals, Petal{Kind: "orb", Rarity: 0, HP: 10, MaxHP: 10})
	r.applyPetalPassives(r.players["p1"], 1.0)
	if r.players["p1"].State.HP > playerMaxHP {
		t.Errorf("реген превысил MaxHP: HP=%v", r.players["p1"].State.HP)
	}
}

// --- MaxHP (bulwark) ---

func TestBulwarkMaxHP(t *testing.T) {
	r := &Room{players: map[string]*Player{}, clients: map[string]*Client{}}
	r.players["p1"] = &Player{State: PlayerState{ID: "p1", HP: 10, MaxHP: playerMaxHP, Zone: 0}}
	r.players["p1"].petals = append(r.players["p1"].petals, Petal{Kind: "bulwark", Rarity: 0, HP: 10, MaxHP: 10})
	r.applyPetalPassives(r.players["p1"], 1.0)
	p := r.players["p1"]
	wantMax := playerMaxHP + playerMaxHP*bulwarkMaxHPFrac*rarityDanger[0]
	if p.State.MaxHP != wantMax {
		t.Errorf("bulwark MaxHP=%v, want %v", p.State.MaxHP, wantMax)
	}
}

// Баф MaxHP (bulwark) ПОСТОЯННЫЙ: действует, пока петаль надета, даже если она
// сломана/на перезарядке (ломается только «боевая» часть — удары).
func TestBulwarkMaxHPRemovedOnBreak(t *testing.T) {
	r := &Room{players: map[string]*Player{}, clients: map[string]*Client{}}
	r.players["p1"] = &Player{State: PlayerState{ID: "p1", HP: 30, MaxHP: playerMaxHP, Zone: 0}}
	r.players["p1"].petals = append(r.players["p1"].petals, Petal{Kind: "bulwark", Rarity: 0, HP: 0, MaxHP: 10, Recharge: 2})
	r.applyPetalPassives(r.players["p1"], 1.0)
	p := r.players["p1"]
	wantMax := playerMaxHP + playerMaxHP*bulwarkMaxHPFrac*rarityDanger[0]
	if p.State.MaxHP != wantMax {
		t.Errorf("после поломки MaxHP=%v, want %v (баф постоянный)", p.State.MaxHP, wantMax)
	}
	if p.State.HP > p.State.MaxHP {
		t.Errorf("HP не подрезан под новый MaxHP: HP=%v MaxHP=%v", p.State.HP, p.State.MaxHP)
	}
}

// --- Магнит (hex) ---

func TestHexMagnet(t *testing.T) {
	r := &Room{players: map[string]*Player{}, clients: map[string]*Client{}}
	r.players["p1"] = &Player{State: PlayerState{ID: "p1", X: 100, Y: 100, Zone: 0}}
	r.players["p1"].petals = append(r.players["p1"].petals, Petal{Kind: "hex", Rarity: 0, HP: 10, MaxHP: 10})
	// Дроп в радиусе — должен притянуться к игроку.
	r.gp = append(r.gp, GroundPetal{ID: "d1", Kind: "oval", X: 180, Y: 100, Zone: 0, Size: 8})
	// Дроп вне радиуса — не трогаем.
	r.gp = append(r.gp, GroundPetal{ID: "d2", Kind: "oval", X: 500, Y: 500, Zone: 0, Size: 8})

	// 0.1 с магнита.
	r.applyMagnets(0.1)
	if r.gp[0].X >= 180 {
		t.Errorf("дроп в радиусе не притянулся: X=%v", r.gp[0].X)
	}
	if r.gp[1].X != 500 || r.gp[1].Y != 500 {
		t.Errorf("дроп вне радиуса сдвинут: X=%v Y=%v", r.gp[1].X, r.gp[1].Y)
	}
}

func TestHexMagnetRadiusScalesWithRarity(t *testing.T) {
	r := &Room{players: map[string]*Player{}, clients: map[string]*Client{}}
	r.players["p1"] = &Player{State: PlayerState{ID: "p1", X: 100, Y: 100, Zone: 0}}
	r.players["p1"].petals = append(r.players["p1"].petals, Petal{Kind: "hex", Rarity: 4, HP: 10, MaxHP: 10})
	// Дроп на расстоянии, доступном только при редком гексе.
	dist := hexMagnetBaseRadius + 4*hexMagnetRadiusPerRar - 20
	r.gp = append(r.gp, GroundPetal{ID: "d1", Kind: "oval", X: 100 + dist, Y: 100, Zone: 0, Size: 8})
	r.applyMagnets(0.05)
	if r.gp[0].X >= 100+dist {
		t.Errorf("редкий гекс не притянул дальний дроп: X=%v", r.gp[0].X)
	}
}

// --- Миссил (blade) ---

func TestBladeFireSpawnsProjectile(t *testing.T) {
	r := &Room{
		players: map[string]*Player{},
		clients: map[string]*Client{},
	}
	r.players["p1"] = &Player{
		State: PlayerState{ID: "p1", X: 0, Y: 0, Size: playerSize, Zone: 0},
	}
	r.players["p1"].petals = append(r.players["p1"].petals, Petal{
		Kind: "blade", HP: 10, MaxHP: 10, DMG: 8, Size: 8, Angle: 0, Dist: 50,
	})
	r.fireBlades("p1")
	if len(r.proj) != 1 {
		t.Fatalf("blade не выстрелил: proj=%d, want 1", len(r.proj))
	}
	pr := r.proj[0]
	if pr.Kind != "blade" {
		t.Errorf("kind снаряда = %q, want blade", pr.Kind)
	}
	if pr.DMG != 8*petalDmgMult {
		t.Errorf("урон снаряда = %v, want %v", pr.DMG, 8*petalDmgMult)
	}
	// Клинок ушёл в полную перезарядку (вылетел сам), чтобы нельзя было спамить.
	pet := &r.players["p1"].petals[0]
	if pet.Recharge != bladeCooldown {
		t.Errorf("кулдаун петеля = %v, want %v", pet.Recharge, bladeCooldown)
	}
}

func TestBladeFireOnlyActive(t *testing.T) {
	r := &Room{players: map[string]*Player{}, clients: map[string]*Client{}}
	r.players["p1"] = &Player{State: PlayerState{ID: "p1", X: 0, Y: 0, Size: playerSize, Zone: 0}}
	r.players["p1"].petals = append(r.players["p1"].petals, Petal{Kind: "blade", HP: 10, MaxHP: 10, Angle: 0, Dist: 50, Recharge: 1})
	r.fireBlades("p1")
	if len(r.proj) != 0 {
		t.Errorf("клинок на перезарядке не должен стрелять: proj=%d", len(r.proj))
	}
}

// --- Разрыв (hiveling) ---

func TestHivelingSplitSpawnsShards(t *testing.T) {
	r := &Room{
		players: map[string]*Player{},
		clients: map[string]*Client{},
		mobs:    []Mob{},
	}
	r.players["p1"] = &Player{State: PlayerState{ID: "p1", X: 0, Y: 0, Size: playerSize, Zone: 0}}
	r.players["p1"].petals = append(r.players["p1"].petals, Petal{
		Kind: "hiveling", HP: 0, MaxHP: 10, DMG: 5, Size: 8, Angle: 0, Dist: 50,
	})
	// Моб рядом — осколки должны лететь в него (в какую-то из сторон).
	r.mobs = append(r.mobs, Mob{ID: "m1", Type: "dot", Rarity: 0, HP: 100, DMG: 2, Radius: 18, Zone: 0, X: 60, Y: 0, Damage: map[string]float64{}})
	r.spawnHivelingShards(r.players["p1"], &r.players["p1"].petals[0])
	if len(r.proj) != hivelingShardCount {
		t.Fatalf("осколков = %d, want %d", len(r.proj), hivelingShardCount)
	}
	if r.proj[0].Kind != "shard" {
		t.Errorf("kind осколка = %q, want shard", r.proj[0].Kind)
	}
}

// --- Снаряд бьёт моба ---

func TestProjectileHitsMob(t *testing.T) {
	r := &Room{
		players: map[string]*Player{},
		clients: map[string]*Client{},
		mobs:    []Mob{},
	}
	r.mobs = append(r.mobs, Mob{ID: "m1", Type: "dot", Rarity: 0, HP: 100, DMG: 2, Radius: 18, Zone: 0, X: 50, Y: 50, Damage: map[string]float64{}})
	r.proj = append(r.proj, r.newProjectile("p1", 0, 0, 50, 50, 0, 0, 10, 0, 8, 0xffffff, "blade", bladeFlyLife, false))
	r.updateProjectiles(0.016)
	if r.mobs[0].HP != 90 {
		t.Errorf("снаряд не нанёс урон: HP=%v, want 90", r.mobs[0].HP)
	}
	if r.mobs[0].Damage["p1"] != 10 {
		t.Errorf("урон не записан владельцу: Damage=%v", r.mobs[0].Damage["p1"])
	}
	if len(r.proj) != 0 {
		t.Errorf("снаряд не исчез после попадания: proj=%d", len(r.proj))
	}
}

// --- Снаряд живёт положенное время и летит (регрессия: Life был 0 и снаряд
// удалялся на первом тике, поэтому blade/hiveling «исчезали» без полёта) ---

func TestProjectilePersistsAcrossTicks(t *testing.T) {
	r := &Room{
		players: map[string]*Player{},
		clients: map[string]*Client{},
		mobs:    []Mob{},
	}
	r.players["p1"] = &Player{
		State: PlayerState{ID: "p1", X: 0, Y: 0, Size: playerSize, Zone: 0},
	}
	r.players["p1"].petals = append(r.players["p1"].petals, Petal{
		Kind: "blade", HP: 10, MaxHP: 10, DMG: 8, Size: 8, Angle: 0, Dist: 50,
	})
	r.fireBlades("p1")
	if len(r.proj) != 1 {
		t.Fatalf("после выстрела proj=%d, want 1", len(r.proj))
	}
	for i := 0; i < 10; i++ {
		r.updateProjectiles(1.0 / 20.0)
	}
	if len(r.proj) != 1 {
		t.Fatalf("снаряд исчез раньше времени: proj=%d, want 1", len(r.proj))
	}
	if r.proj[0].X <= 50 {
		t.Errorf("снаряд не полетел: X=%v", r.proj[0].X)
	}
}

// --- Осколок hiveling пробивает моба (не исчезает на первом попадании) ---

func TestShardPiercesMob(t *testing.T) {
	r := &Room{
		players: map[string]*Player{},
		clients: map[string]*Client{},
		mobs:    []Mob{},
	}
	r.mobs = append(r.mobs, Mob{ID: "m1", Type: "dot", Rarity: 0, HP: 100, DMG: 2, Radius: 18, Zone: 0, X: 55, Y: 50, Damage: map[string]float64{}})
	// Пробивающий осколок, летящий в сторону моба.
	r.proj = append(r.proj, r.newProjectile("p1", 0, 0, 50, 50, 60, 0, 5, 0, 8, 0xffffff, "shard", hivelingShardLife, true))
	r.updateProjectiles(1.0 / 20.0)
	if r.mobs[0].HP != 95 {
		t.Errorf("осколок не нанёс урон: HP=%v, want 95", r.mobs[0].HP)
	}
	if len(r.proj) != 1 {
		t.Fatalf("пробивающий осколок исчез на попадании: proj=%d, want 1", len(r.proj))
	}
}

// --- Клинок несёт HP и пробивает: бьёт моба, но не исчезает на первом
// попадании, а изнашивается от урона моба (m.DMG*0.3 за удар) ---

func TestBladePiercesAndWears(t *testing.T) {
	r := &Room{
		players: map[string]*Player{},
		clients: map[string]*Client{},
		mobs:    []Mob{},
	}
	r.mobs = append(r.mobs, Mob{ID: "m1", Type: "dot", Rarity: 0, HP: 100, DMG: 10, Radius: 18, Zone: 0, X: 55, Y: 50, Damage: map[string]float64{}})
	// Клинок с HP 7, пробивающий.
	r.proj = append(r.proj, r.newProjectile("p1", 0, 0, 50, 50, 60, 0, 8, 7, 8, 0xffffff, "blade", bladeFlyLife, true))
	r.updateProjectiles(1.0 / 20.0)
	if r.mobs[0].HP != 92 {
		t.Errorf("клинок не нанёс урон: HP=%v, want 92", r.mobs[0].HP)
	}
	if len(r.proj) != 1 {
		t.Fatalf("клинок исчез на первом попадании: proj=%d, want 1 (должен пробивать)", len(r.proj))
	}
	if r.proj[0].HP != 7-10*0.3 {
		t.Errorf("клинок не износился: HP=%v, want %v", r.proj[0].HP, 7-10*0.3)
	}
}
