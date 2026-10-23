package main

import "testing"

// --- Таблица призыва миньонов (summonSpec): редкость моба / усиление / MK ---

func TestSummonSpecBasic(t *testing.T) {
	// Point..Square петали призывают ТУ ЖЕ редкость моба, без усиления и MK.
	for rar, wantRar := range []int{0, 1, 2, 3} {
		mobRar, mult, label, interval := summonSpec(rar)
		if mobRar != wantRar || mult != 1 || label != "" || interval != minionSpawnInterval {
			t.Errorf("petal r%d spec: %d/%.1f/%q/%.1f, want %d/1/\"\"/%.1f",
				rar, mobRar, mult, label, interval, wantRar, minionSpawnInterval)
		}
	}
}

func TestSummonSpecPentagon(t *testing.T) {
	mobRar, mult, label, interval := summonSpec(4)
	if mobRar != 3 || mult != 2 || label != "Square MK-II" || interval != minionSpawnInterval {
		t.Errorf("petal r4 spec: %d/%.1f/%q/%.1f, want 3/2/Square MK-II/%.1f",
			mobRar, mult, label, interval, minionSpawnInterval)
	}
}

func TestSummonSpecHexagon(t *testing.T) {
	mobRar, mult, label, interval := summonSpec(5)
	if mobRar != 4 || mult != 1 || label != "Pentagon" || interval != minionSpawnInterval {
		t.Errorf("petal r5 spec: %d/%.1f/%q/%.1f, want 4/1/Pentagon/%.1f",
			mobRar, mult, label, interval, minionSpawnInterval)
	}
}

func TestSummonSpecFractal(t *testing.T) {
	mobRar, mult, label, interval := summonSpec(6)
	if mobRar != 4 || mult != 2 || label != "Pentagon MK-II" || interval != minionSpawnIntervalFractal {
		t.Errorf("petal r6 spec: %d/%.1f/%q/%.1f, want 4/2/Pentagon MK-II/%.1f",
			mobRar, mult, label, interval, minionSpawnIntervalFractal)
	}
}

func TestSummonSpecChaos(t *testing.T) {
	mobRar, mult, label, interval := summonSpec(7)
	if mobRar != 4 || mult != 5 || label != "Pentagon MK-III" || interval != minionSpawnInterval {
		t.Errorf("petal r7 spec: %d/%.1f/%q/%.1f, want 4/5/Pentagon MK-III/%.1f",
			mobRar, mult, label, interval, minionSpawnInterval)
	}
}

// Статы якоря миньона: точные статы враждебного моба того же типа и редкости,
// умноженные на множитель усиления (MK-версии).
func TestSummonAnchorStats(t *testing.T) {
	initWorld()
	rar := 4 // Pentagon
	mobRar, mult, label, _ := summonSpec(rar)
	p := &Player{
		State: PlayerState{ID: "p1", Zone: 0, X: 4000, Y: 2500},
		petals: []Petal{
			{Kind: "entropy", Rarity: rar, HP: 100, MaxHP: 100, OwnerKey: "e1"},
		},
	}
	r := &Room{players: map[string]*Player{"p1": p}}
	anchors := r.collectMinionAnchors()
	a := anchors["e1"]
	if a == nil {
		t.Fatal("anchor for entropy r4 not created")
	}
	wantHP, wantDmg := mobStatsForRarity("entropy", mobRar)
	wantHP *= mult
	wantDmg *= mult
	// Множитель должен реально применяться (r4 = Square MK-II ×2).
	realHP, realDMG := mobStatsForRarity("entropy", mobRar)
	if wantHP != realHP*2 || wantDmg != realDMG*2 {
		t.Fatalf("mult not applied: hp %.3f vs %.3f, dmg %.3f vs %.3f", wantHP, realHP, wantDmg, realDMG)
	}
	if a.hp != wantHP || a.dmg != wantDmg {
		t.Errorf("r4 anchor hp/dmg = %.3f/%.3f, want %.3f/%.3f", a.hp, a.dmg, wantHP, wantDmg)
	}
	if a.rarity != mobRar {
		t.Errorf("r4 anchor rarity = %d, want %d", a.rarity, mobRar)
	}
	if a.rarityLabel != label {
		t.Errorf("r4 anchor label = %q, want %q", a.rarityLabel, label)
	}
}

// HP и DMG миньона совпадают со статами враждебного моба того же типа и
// редкости (без MK-усиления): Point..Square петали.
func TestSummonAnchorStatsMatchRealMob(t *testing.T) {
	initWorld()
	for petRar := 0; petRar <= 3; petRar++ {
		p := &Player{
			State: PlayerState{ID: "p1", Zone: 0, X: 4000, Y: 2500},
			petals: []Petal{
				{Kind: "entropy", Rarity: petRar, HP: 100, MaxHP: 100, OwnerKey: "e1"},
				{Kind: "singularity", Rarity: petRar, HP: 100, MaxHP: 100, OwnerKey: "s1"},
			},
		}
		r := &Room{players: map[string]*Player{"p1": p}}
		anchors := r.collectMinionAnchors()
		ae, as := anchors["e1"], anchors["s1"]
		if ae == nil || as == nil {
			t.Fatalf("r%d: anchors not created", petRar)
		}
		wantHP, wantDMG := mobStatsForRarity("entropy", petRar)
		if ae.hp != wantHP || ae.dmg != wantDMG {
			t.Errorf("r%d entropy minion hp/dmg = %.3f/%.3f, want real-mob %.3f/%.3f",
				petRar, ae.hp, ae.dmg, wantHP, wantDMG)
		}
		wantHP, wantDMG = mobStatsForRarity("blackhole", petRar)
		if as.hp != wantHP || as.dmg != wantDMG {
			t.Errorf("r%d blackhole minion hp/dmg = %.3f/%.3f, want real-mob %.3f/%.3f",
				petRar, as.hp, as.dmg, wantHP, wantDMG)
		}
	}
}

// Глубокие редкости (5+): призывы БЕЗ боссовых множителей HP и с дебафом ×0.5 —
// не должны быть неубиваемыми.
func TestSummonAnchorDeepRarityDebuff(t *testing.T) {
	initWorld()
	for _, petRar := range []int{5, 6, 7} {
		p := &Player{
			State: PlayerState{ID: "p1", Zone: 0, X: 4000, Y: 2500},
			petals: []Petal{
				{Kind: "singularity", Rarity: petRar, HP: 100, MaxHP: 100, OwnerKey: "s1"},
			},
		}
		r := &Room{players: map[string]*Player{"p1": p}}
		anchors := r.collectMinionAnchors()
		a := anchors["s1"]
		if a == nil {
			t.Fatalf("r%d: anchor not created", petRar)
		}
		mobRar, mult, _, _ := summonSpec(petRar)
		wantHP, wantDMG := summonMobStats("blackhole", mobRar)
		wantHP *= mult
		wantDMG *= mult
		// Петал редкости 5+ дополнительно ослаблен вдвое.
		wantHP *= summonDeepDebuff
		wantDMG *= summonDeepDebuff
		if a.hp != wantHP || a.dmg != wantDMG {
			t.Errorf("r%d anchor hp/dmg = %.0f/%.1f, want %.0f/%.1f",
				petRar, a.hp, a.dmg, wantHP, wantDMG)
		}
	}
}

// Миньоны НЕ проходят сквозь разделительную стену (Mob Zone | Safe Zone).
func TestMinionDividerWallClamp(t *testing.T) {
	initWorld()
	z := zones[0]
	segs := wallSegments(&z)
	if len(segs) == 0 {
		t.Fatal("zone 0 должна иметь разделительную стену")
	}
	s := segs[0]
	mn := &Minion{
		ID: "m1", Zone: 0, Alive: true,
		X: s.X + s.W/2, Y: s.Y + s.H/2, // центр внутри левой половины стены
		Size: 10, HP: 10, MaxHP: 10,
	}
	clampMinionInZone(mn)
	for _, seg := range segs {
		nx := clampf(mn.X, seg.X, seg.X+seg.W)
		ny := clampf(mn.Y, seg.Y, seg.Y+seg.H)
		dx := mn.X - nx
		dy := mn.Y - ny
		if dx*dx+dy*dy < mn.Size*mn.Size {
			t.Errorf("миньон остался пересекать стену после клампа: (%v, %v)", mn.X, mn.Y)
		}
	}
}

// Void-рывок не может протолкнуть игрока сквозь разделительную стену.
func TestVoidDashBlockedByWall(t *testing.T) {
	initWorld()
	r := &Room{players: map[string]*Player{}, clients: map[string]*Client{}}
	z := zones[0]
	above := mobZoneBottom(&z) - 10
	r.players["p1"] = &Player{
		State: PlayerState{ID: "p1", X: 2000, Y: above, Size: playerSize, Zone: 0},
		lastDir: Vec{X: 0, Y: 1},
	}
	r.players["p1"].petals = []Petal{{Kind: "void", HP: 100, MaxHP: 100, DMG: 10, Size: 8}}
	r.dashVoid("p1")
	p := r.players["p1"]
	if p.State.Y+playerSize > mobZoneBottom(&z) {
		t.Errorf("void-рывок пробил стену: низ игрока=%v, верх стены=%v",
			p.State.Y+playerSize, mobZoneBottom(&z))
	}
}

// Реген вне боя: 20с без урона → медленный реген; при уроне — нет.
func TestOutOfCombatRegen(t *testing.T) {
	initWorld()
	r := &Room{players: map[string]*Player{}, clients: map[string]*Client{}}
	p := &Player{State: PlayerState{ID: "p1", HP: 10, MaxHP: 30, Zone: 0}, noDamageT: 25}
	r.players["p1"] = p
	dt := 1.0
	r.applyPetalPassives(p, dt)
	// Без bulwark MaxHP обнулится до playerMaxHP; реген = MaxHP×frac×dt.
	want := 10 + playerMaxHP*outOfCombatRegenFrac*dt
	if p.State.HP != want {
		t.Errorf("реген вне боя: HP=%v, want %v", p.State.HP, want)
	}
	// В бою (таймер < 20с) регена нет.
	p2 := &Player{State: PlayerState{ID: "p2", HP: 10, MaxHP: 30, Zone: 0}, noDamageT: 5}
	r.players["p2"] = p2
	r.applyPetalPassives(p2, dt)
	if p2.State.HP != 10 {
		t.Errorf("реген в бою: HP=%v, want 10", p2.State.HP)
	}
	// Любой урон обнуляет таймер.
	p3 := &Player{State: PlayerState{ID: "p3", HP: 18, MaxHP: 18, Shield: 12, MaxShield: 20}, noDamageT: 30}
	applyPlayerDamage(p3, 6)
	if p3.noDamageT != 0 {
		t.Errorf("урон не сбросил таймер: noDamageT=%v, want 0", p3.noDamageT)
	}
}