package main

import "testing"

// TestPoisonFlags — петаль-«гадюка» отмечена Poison, обычные виды — нет.
func TestPoisonFlags(t *testing.T) {
	if !petalKinds["viper"].Poison {
		t.Errorf("viper должен иметь Poison=true (механика «яд»)")
	}
	for _, kind := range []string{"oval", "thorns", "rock", "ball", "void", "entropy", "blade"} {
		if petalKinds[kind].Poison {
			t.Errorf("%s не должен быть ядовитым", kind)
		}
	}
}

// TestPoisonNoStack — яд НЕ стакается: свежий удар освежает таймер, но сила
// пула только УДЕРЖИВАЕТ максимум (take-max), а не складывает ДПС гадюк.
func TestPoisonNoStack(t *testing.T) {
	m := &Mob{PoisonDps: 0, PoisonT: 0}
	// Первая гадюка: сильный яд.
	poisonMob(m, 100.0, "p1")
	// Вторая гадюка бьёт СЛАБЕЕ — не должна уронить силу пула.
	poisonMob(m, 40.0, "p2")
	if m.PoisonDps != 100 {
		t.Errorf("слабый удар ослабил яд: PoisonDps=%v, want 100 (take-max)", m.PoisonDps)
	}
	// Свежий слабый удар всё равно освежает таймер до полного.
	if m.PoisonT != poisonDuration {
		t.Errorf("таймер не освежён: PoisonT=%v, want %v", m.PoisonT, poisonDuration)
	}
	// И более сильный удар поднимает пул выше.
	poisonMob(m, 250.0, "p1")
	if m.PoisonDps != 250 {
		t.Errorf("более сильный яд не поднял пул: PoisonDps=%v, want 250", m.PoisonDps)
	}
}

// TestPoisonDpsAndKill — яд тикает урон каждый кадр (пишет в Damage-карту),
// умирающий от яда моб приносит дроп/XP отравителю.
func TestPoisonDpsAndKill(t *testing.T) {
	r := &Room{
		players: map[string]*Player{},
		clients: map[string]*Client{},
	}
	r.players["p1"] = &Player{
		State: PlayerState{ID: "p1", Name: "T", Level: 50, XP: 0},
	}
	r.mobs = append(r.mobs, Mob{
		ID: "m1", Type: "dot", Rarity: 0,
		HP: 20, MaxHP: 100, DMG: 2, Speed: 10, Radius: 18, Zone: 0,
		Damage: map[string]float64{},
		X: 100, Y: 100,
	})

	poisonMob(&r.mobs[0], 10.0, "p1")
	m := &r.mobs[0]
	if m.PoisonT != poisonDuration || m.PoisonDps != 10 {
		t.Fatalf("яд не повешался: T=%v DPS=%v", m.PoisonT, m.PoisonDps)
	}

	// 0.5 с тика = 5 урона.
	r.applyPoison(0.5)
	if m.HP != 15 {
		t.Errorf("HP после яда = %v, want 15 (стало умирать от яда)", m.HP)
	}
	if m.Damage["p1"] != 5 {
		t.Errorf("урон в Damage-карте = %v, want 5", m.Damage["p1"])
	}

	// Ещё 2 с = ещё 20 урона → моб умер от яда отравителя.
	r.applyPoison(2.0)
	if m.HP > 0 {
		t.Fatalf("моб выжил: HP=%v", m.HP)
	}
	if m.KillerID != "p1" {
		t.Errorf("убийца = %q, want p1 (смерть от яда)", m.KillerID)
	}
	// Дроп за убийство появился у отравителя.
	if len(r.gp) != 1 {
		t.Errorf("дроп не создан: gp=%d, want 1", len(r.gp))
	}
	if r.players["p1"].State.XP <= 0 {
		t.Errorf("XP убийце не начислен (XP=%v)", r.players["p1"].State.XP)
	}
}

// TestPoisonBossPetal — ядовитый петаль игрока травит орбитальные петали боссов
// (ПВЕ): урон тикает, убитая ядом петаль уходит на респавн (Respawn>0).
func TestPoisonBossPetal(t *testing.T) {
	r := &Room{players: map[string]*Player{}, clients: map[string]*Client{}}
	r.mobs = append(r.mobs, Mob{
		ID: "m1", Type: "actal", Rarity: rarityFractal,
		HP: 1000, MaxHP: 1000, DMG: 10, Speed: 10, Radius: 30, Zone: 0,
		Damage: map[string]float64{},
		X: 100, Y: 100,
	})
	initBossPetals(&r.mobs[0])
	m := &r.mobs[0]
	if len(m.petals) != 4 {
		t.Fatalf("Fractal должен нести 4 орбитальные петали, got %d", len(m.petals))
	}
	bp := &m.petals[0]
	poisonBossPetal(bp, 10.0)
	before := bp.HP
	r.applyPoison(0.5)
	if bp.HP != before-5 {
		t.Errorf("HP петеля после яда = %v, want %v", bp.HP, before-5)
	}
	// Мощный яд добивает петаль — она уходит на респавн.
	poisonBossPetal(bp, 100000.0)
	r.applyPoison(1.0)
	if bp.HP != 0 {
		t.Fatalf("петаль не умерла от яда: HP=%v", bp.HP)
	}
	if bp.Respawn <= 0 {
		t.Errorf("убитая ядом петаль не в респавне: Respawn=%v", bp.Respawn)
	}
	if bp.PoisonT != 0 || bp.PoisonDps != 0 {
		t.Errorf("яд мертвой петали не очищен: T=%v DPS=%v", bp.PoisonT, bp.PoisonDps)
	}
}

// TestPoisonPlayerPetal — в ПВП «гадюка» травит петали других игроков:
// урон тикает, убитая ядом петаль уходит в перезарядку (Recharge).
func TestPoisonPlayerPetal(t *testing.T) {
	r := &Room{
		players: map[string]*Player{},
		clients: map[string]*Client{},
	}
	r.players["p1"] = &Player{}
	r.players["p1"].petals = append(r.players["p1"].petals, Petal{
		ID: "petA", HP: 20, MaxHP: 20, DMG: 5, Size: 8, Kind: "viper",
	})
	pet := &r.players["p1"].petals[0]
	poisonPlayerPetal(pet, 10.0)
	r.applyPoison(0.5)
	if pet.HP != 15 {
		t.Errorf("HP петеля после яда = %v, want 15", pet.HP)
	}
	// Мощный яд добивает петаль — она уходит в перезарядку.
	poisonPlayerPetal(pet, 100000.0)
	r.applyPoison(1.0)
	if pet.HP != 0 {
		t.Fatalf("петаль не умерла от яда: HP=%v", pet.HP)
	}
	// Перезарядка — per-kind (порт родителя): viper 5с×1.5 = 7.5с.
	if pet.Recharge != petalRechargeT*petalRechargeMults[pet.Kind] {
		t.Errorf("убитая ядом петаль не в перезарядке: Recharge=%v", pet.Recharge)
	}
	if pet.PoisonT != 0 || pet.PoisonDps != 0 {
		t.Errorf("яд мертвой петали не очищен: T=%v DPS=%v", pet.PoisonT, pet.PoisonDps)
	}
}