package main

import (
	"math"
	"testing"
)

// --- Oval: лайфстил ---

func TestOvalLifesteal(t *testing.T) {
	r := &Room{players: map[string]*Player{}, clients: map[string]*Client{}}
	r.players["p1"] = &Player{
		State: PlayerState{ID: "p1", X: 0, Y: 0, Size: playerSize, Zone: 0, HP: 5, MaxHP: 10},
	}
	r.players["p1"].petals = append(r.players["p1"].petals, Petal{Kind: "oval", HP: 100, MaxHP: 100, DMG: 4, Size: 8, Angle: 0, Dist: 50})
	r.mobs = append(r.mobs, Mob{ID: "m1", Type: "dot", Rarity: 0, HP: 100, DMG: 2, Radius: 18, Zone: 0, Damage: map[string]float64{}, X: 50, Y: 0})
	r.fightPetalsVsMobs(0.016)
	p := r.players["p1"]
	if p.State.HP != 5+4*petalDmgMult*lifestealPct {
		t.Errorf("лайфстил: HP=%v, want %v", p.State.HP, 5+4*petalDmgMult*lifestealPct)
	}
}

// --- Rock: броня (износ вдвое меньше) ---

func TestRockArmor(t *testing.T) {
	r := &Room{players: map[string]*Player{}, clients: map[string]*Client{}}
	r.players["p1"] = &Player{State: PlayerState{ID: "p1", X: 0, Y: 0, Size: playerSize, Zone: 0}}
	r.players["p1"].petals = append(r.players["p1"].petals, Petal{Kind: "rock", HP: 100, MaxHP: 100, DMG: 2.5, Size: 8, Angle: 0, Dist: 50})
	r.mobs = append(r.mobs, Mob{ID: "m1", Type: "dot", Rarity: 0, HP: 100, DMG: 10, Radius: 18, Zone: 0, Damage: map[string]float64{}, X: 50, Y: 0})
	r.fightPetalsVsMobs(0.016)
	pet := &r.players["p1"].petals[0]
	if pet.HP != 100-10*0.3*0.5 {
		t.Errorf("броня: HP петеля=%v, want %v", pet.HP, 100-10*0.3*0.5)
	}
}

// --- Ball: бомба (мини-АОЕ) ---

func TestBallBomb(t *testing.T) {
	r := &Room{players: map[string]*Player{}, clients: map[string]*Client{}}
	r.players["p1"] = &Player{State: PlayerState{ID: "p1", X: 0, Y: 0, Size: playerSize, Zone: 0}}
	r.players["p1"].petals = append(r.players["p1"].petals, Petal{Kind: "ball", HP: 100, MaxHP: 100, DMG: 3, Size: 8, Angle: 0, Dist: 50})
	r.mobs = append(r.mobs,
		Mob{ID: "a", Type: "dot", Rarity: 0, HP: 100, DMG: 2, Radius: 18, Zone: 0, Damage: map[string]float64{}, X: 50, Y: 0},
		Mob{ID: "b", Type: "dot", Rarity: 0, HP: 100, DMG: 2, Radius: 18, Zone: 0, Damage: map[string]float64{}, X: 100, Y: 0},
	)
	r.fightPetalsVsMobs(0.016)
	if r.mobs[0].HP != 100-3*petalDmgMult {
		t.Errorf("основная цель: HP=%v, want %v", r.mobs[0].HP, 100-3*petalDmgMult)
	}
	if r.mobs[1].HP != 100-3*petalDmgMult*bombDmgFrac {
		t.Errorf("сосед от бомбы: HP=%v, want %v", r.mobs[1].HP, 100-3*petalDmgMult*bombDmgFrac)
	}
}

// --- Pebble: праща (сильный толчок + почти нет кулдауна) ---

func TestPebbleSling(t *testing.T) {
	initWorld()
	r := &Room{players: map[string]*Player{}, clients: map[string]*Client{}}
	r.players["p1"] = &Player{State: PlayerState{ID: "p1", X: 0, Y: 0, Size: playerSize, Zone: 0}}
	r.players["p1"].petals = append(r.players["p1"].petals, Petal{Kind: "pebble", HP: 100, MaxHP: 100, DMG: 3, Size: 8, Rarity: 0, Angle: 0, Dist: 50})
	r.mobs = append(r.mobs, Mob{ID: "m1", Type: "dot", Rarity: 0, HP: 100, DMG: 2, Speed: 10, Radius: 18, Zone: 0, Damage: map[string]float64{}, X: 55, Y: 0})
	r.fightPetalsVsMobs(0.016)
	pet := &r.players["p1"].petals[0]
	// Праща: свежая петаль бьёт при первом касании (готова), после удара —
	// идёт зарядка (Charging=true, AtkT растёт от 0 к 1 темпом dt/КД).
	if !pet.Charging || pet.AtkT != 0 {
		t.Errorf("праща: Charging=%v AtkT=%v, want Charging=true AtkT=0 после удара", pet.Charging, pet.AtkT)
	}
	// Моб должен сдвинуться (сильный толчок).
	if r.mobs[0].X == 55 {
		t.Errorf("праща не толкнула моба: X=%v", r.mobs[0].X)
	}
}

// --- Shard: рана (короткий DoT) ---

func TestShardWound(t *testing.T) {
	r := &Room{players: map[string]*Player{}, clients: map[string]*Client{}}
	r.players["p1"] = &Player{State: PlayerState{ID: "p1", X: 0, Y: 0, Size: playerSize, Zone: 0}}
	r.players["p1"].petals = append(r.players["p1"].petals, Petal{Kind: "shard", HP: 100, MaxHP: 100, DMG: 9, Size: 8, Angle: 0, Dist: 50})
	r.mobs = append(r.mobs, Mob{ID: "m1", Type: "dot", Rarity: 0, HP: 100, DMG: 2, Radius: 18, Zone: 0, Damage: map[string]float64{}, X: 50, Y: 0})
	r.fightPetalsVsMobs(0.016)
	m := &r.mobs[0]
	if m.PoisonT != woundDuration {
		t.Errorf("рана: PoisonT=%v, want %v", m.PoisonT, woundDuration)
	}
	if m.PoisonDps != 9*petalDmgMult*woundDpsMult {
		t.Errorf("рана: PoisonDps=%v, want %v", m.PoisonDps, 9*petalDmgMult*woundDpsMult)
	}
}

// --- Cog: «заводка» — укорачивает кулдаун удара всех петалей (стакается) ---

func TestCogOverdrive(t *testing.T) {
	p := &Player{petals: []Petal{
		{Kind: "cog", Rarity: 0, HP: 10, MaxHP: 10},
		{Kind: "cog", Rarity: 2, HP: 10, MaxHP: 10},
	}}
	// Каждый cog умножает КД удара на (1 - 0.08 - 0.04×редкость):
	// cog r0: 1 - 0.08 = 0.92; cog r2: 1 - 0.08 - 0.08 = 0.84.
	want := 0.92 * 0.84
	if got := overdriveMult(p); math.Abs(got-want) > 1e-9 {
		t.Errorf("overdrive mult=%v, want %v", got, want)
	}
	// Сломанное cog не заводит.
	p.petals[1].HP = 0
	if got := overdriveMult(p); got != 0.92 {
		t.Errorf("overdrive mult с мёртвым cog=%v, want 0.92", got)
	}
}

// --- Whirl: носитель ускорителя вращения орбиты (стакается) ---

func TestWhirlSpin(t *testing.T) {
	p := &Player{petals: []Petal{
		{Kind: "whirl", Rarity: 0, HP: 10, MaxHP: 10},
		{Kind: "whirl", Rarity: 2, HP: 10, MaxHP: 10},
	}}
	want := 2*cogSpinBase + 2*cogSpinPerRar // 0.3*2 + 0.1*2 = 0.8
	if got := orbitSpinBonus(p); got != want {
		t.Errorf("spin bonus=%v, want %v", got, want)
	}
	// cog больше не вращает орбиту.
	p.petals = []Petal{{Kind: "cog", Rarity: 0, HP: 10, MaxHP: 10}}
	if got := orbitSpinBonus(p); got != 0 {
		t.Errorf("cog не должен крутить орбиту, spin bonus=%v", got)
	}
}

// --- Twirl: разворот направления вращения орбиты (парность) ---

func TestTwirlOrbitDir(t *testing.T) {
	// Без twirl направление обычное.
	p := &Player{petals: []Petal{{Kind: "oval", Rarity: 0, HP: 10, MaxHP: 10}}}
	if got := orbitDir(p); got != 1 {
		t.Errorf("без twirl направление = %v, want 1", got)
	}
	// Один twirl — орбита развернулась.
	p.petals = append(p.petals, Petal{Kind: "twirl", Rarity: 0, HP: 10, MaxHP: 10})
	if got := orbitDir(p); got != -1 {
		t.Errorf("с twirl направление = %v, want -1", got)
	}
	// Второй twirl — снова по часовой (парность).
	p.petals = append(p.petals, Petal{Kind: "twirl", Rarity: 4, HP: 10, MaxHP: 10})
	if got := orbitDir(p); got != 1 {
		t.Errorf("два twirl направление = %v, want 1", got)
	}
	// Сломанный twirl всё равно разворачивает (пассив действует, пока надета).
	p.petals = append(p.petals, Petal{Kind: "twirl", Rarity: 1, HP: 0, MaxHP: 10})
	if got := orbitDir(p); got != -1 {
		t.Errorf("три twirl (один сломан) направление = %v, want -1", got)
	}
}

// --- Penta: ускорение передвижения (стакается) ---

func TestPentaSwift(t *testing.T) {
	p := &Player{petals: []Petal{
		{Kind: "penta", Rarity: 0, HP: 10, MaxHP: 10},
		{Kind: "penta", Rarity: 1, HP: 10, MaxHP: 10},
	}}
	want := 1 + (0.10+0.02*0) + (0.10+0.02*1) // 1 + 0.10 + 0.12 = 1.22
	if got := moveSpeedMult(p); math.Abs(got-want) > 1e-9 {
		t.Errorf("speed mult=%v, want %v", got, want)
	}
}

// --- Fractalite: самоподобие (флаг) ---

func TestFractaliteTwinFlag(t *testing.T) {
	if !petalKinds["fractalite"].Twin {
		t.Errorf("fractalite должен иметь Twin (двойной удар)")
	}
}

// --- Fragwall: барьер-щит (снижение входящего урона, стакается) ---

func TestFragwallGuard(t *testing.T) {
	p := &Player{petals: []Petal{
		{Kind: "fragwall", Rarity: 0, HP: 10, MaxHP: 10},
		{Kind: "fragwall", Rarity: 1, HP: 10, MaxHP: 10},
	}}
	// 0.05 + 0.05+0.02 = 0.12 → mult = 0.88
	want := 1 - (guardBasePct + guardBasePct + guardPerRarity)
	if got := damageTakenMult(p); math.Abs(got-want) > 1e-9 {
		t.Errorf("damage mult=%v, want %v", got, want)
	}
}

func TestFragwallGuardFloor(t *testing.T) {
	p := &Player{petals: []Petal{
		{Kind: "fragwall", Rarity: 7, HP: 10, MaxHP: 10},
		{Kind: "fragwall", Rarity: 7, HP: 10, MaxHP: 10},
		{Kind: "fragwall", Rarity: 7, HP: 10, MaxHP: 10},
	}}
	// Много фрагволлов — но не ниже 20% урона.
	if got := damageTakenMult(p); got < 0.2 {
		t.Errorf("damage mult=%v, не должен быть ниже 0.2", got)
	}
}

// --- Shield: щит-броня поглощает урон перед HP и перезаряжается ---

func TestShieldAbsorbsDamage(t *testing.T) {
	p := &Player{State: PlayerState{HP: 18, MaxHP: 18, Shield: 12, MaxShield: 20}}
	applyPlayerDamage(p, 6)
	if p.State.HP != 18 || p.State.Shield != 6 {
		t.Errorf("щит не поглотил урон: HP=%v Shield=%v, want 18/6", p.State.HP, p.State.Shield)
	}
}

func TestShieldRecharge(t *testing.T) {
	r := &Room{players: map[string]*Player{}, clients: map[string]*Client{}}
	r.players["p1"] = &Player{State: PlayerState{ID: "p1", HP: 18, MaxHP: 18, Shield: 0}}
	r.players["p1"].petals = append(r.players["p1"].petals, Petal{Kind: "shield", Rarity: 0, HP: 10, MaxHP: 10})
	r.applyPetalPassives(r.players["p1"], 0.5)
	p := r.players["p1"]
	if p.State.MaxShield != shieldCapBase*rarityDanger[0] {
		t.Errorf("MaxShield=%v, want %v", p.State.MaxShield, shieldCapBase*rarityDanger[0])
	}
	if p.State.Shield != shieldChargeBase*rarityDanger[0] {
		t.Errorf("Shield=%v, want %v", p.State.Shield, shieldChargeBase*rarityDanger[0])
	}
}
