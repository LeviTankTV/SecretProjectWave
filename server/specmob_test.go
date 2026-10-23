package main

import "testing"

// Тесты спецспособностей мобов: Comet (заряд→рывок→отдых), Arrow
// (разгон→крейсер→торможение→отдых) и энергетические шары Beam.

// makeMobInitComet — детерминированная инициализация кометы (без рандома).
func makeMobInitComet(m *Mob) {
	m.cometPhase = 0
	m.cometT = cometWaitDur * 0.9
}

func TestCometPhaseCycle(t *testing.T) {
	m := Mob{Type: "comet", Speed: 100, dirX: 1, dirY: 0, state: "chase"}
	makeMobInitComet(&m)

	// Фаза 0 — обычная скорость, пока не кончилось ожидание.
	if s := cometPhaseSpeed(&m, 0.01); s != 100 {
		t.Fatalf("комета в полёте: speed=%v, want 100", s)
	}
	// Форсируем переход к зарядке.
	m.cometPhase = 1
	m.cometDur = cometChargeDur
	m.cometT = cometChargeDur
	if s := cometPhaseSpeed(&m, 0); s != 100 {
		t.Fatalf("зарядка старт: speed=%v, want 100", s)
	}
	// В конце зарядки скорость почти 0.
	m.cometT = cometChargeDur * 0.01
	if s := cometPhaseSpeed(&m, 0); s >= 3 {
		t.Fatalf("зарядка в конце: speed=%v, want ~0", s)
	}
	// Переход к рывку: скорость = Speed × dashMult.
	if s := cometPhaseSpeed(&m, cometChargeDur); s != 100*cometDashMult {
		t.Fatalf("рывок: speed=%v, want %v", s, 100*cometDashMult)
	}
	// Направление рывка зафиксировано (dirX=1, dirY=0).
	if m.cometDX != 1 || m.cometDY != 0 {
		t.Fatalf("dash dir=(%v,%v), want (1,0)", m.cometDX, m.cometDY)
	}
	// Погоня перезаписывает dir каждый тик — рывок держит своё направление.
	m.dirX, m.dirY = 0.3, 0.9
	cometPhaseSpeed(&m, 0)
	if m.dirX != 1 || m.dirY != 0 {
		t.Fatalf("dash dir перекручен погоней: (%v,%v)", m.dirX, m.dirY)
	}
	// После рывка — отдых (0), затем снова обычный полёт.
	m.cometT = 0
	if s := cometPhaseSpeed(&m, 0.01); s != 0 {
		t.Fatalf("отдых: speed=%v, want 0", s)
	}
	m.cometT = 0
	cometPhaseSpeed(&m, 0.01)
	if s := cometPhaseSpeed(&m, 0.001); s != 100 {
		t.Fatalf("после отдыха: speed=%v, want 100 (полёт)", s)
	}
}

func TestArrowPhaseCycle(t *testing.T) {
	m := Mob{Type: "arrow", Speed: 160, dirX: 1, dirY: 0}
	m.arrowPhase = 0
	m.arrowDur = arrowAccelDur
	m.arrowT = arrowAccelDur

	// Разгон: почти 0 в начале.
	if s := arrowPhaseSpeed(&m, 0.01); s > 160*0.05 {
		t.Fatalf("разгон старт: speed=%v, want ~0", s)
	}
	// Ближе к концу разгона скорость почти полная (спринт ×3).
	m.arrowT = arrowAccelDur * 0.05
	if s := arrowPhaseSpeed(&m, arrowAccelDur*0.04); s < 160*arrowSprintMult*0.9 {
		t.Fatalf("разгон в конце: speed=%v, want почти максимум спринта", s)
	}
	// Переход к крейсеру — полная спринт-скорость (×3).
	m.arrowT = 0
	if s := arrowPhaseSpeed(&m, 0.01); s != 160*arrowSprintMult {
		t.Fatalf("крейсер: speed=%v, want %v", s, 160*arrowSprintMult)
	}
	if m.arrowPhase != 1 {
		t.Fatalf("phase=%d, want 1 (cruise)", m.arrowPhase)
	}
	// Торможение в конце → 0.
	m.arrowPhase = 2
	m.arrowDur = arrowBrakeDur
	m.arrowT = arrowBrakeDur
	if s := arrowPhaseSpeed(&m, arrowBrakeDur); s != 0 {
		t.Fatalf("торможение в конце: speed=%v, want 0", s)
	}
	// Отдых: стоит, затем снова разгон.
	m.arrowPhase = 3
	m.arrowDur = arrowRestDur
	m.arrowT = arrowRestDur
	if s := arrowPhaseSpeed(&m, arrowRestDur); s != 0 {
		t.Fatalf("отдых: speed=%v, want 0", s)
	}
	if m.arrowPhase != 0 {
		t.Fatalf("после отдыха phase=%d, want 0 (accel)", m.arrowPhase)
	}
}

func TestBeamOrbBodyDamageAndPop(t *testing.T) {
	r := newTestRoom()
	initWorld()
	p := &Player{State: PlayerState{ID: "p1", X: 4000, Y: 2500, Size: 30, Zone: 0}}
	p.State.HP = 1000
	p.State.MaxHP = 1000
	r.players["p1"] = p

	r.orbs = append(r.orbs, BeamOrb{
		ID: "orb_1", X: 4020, Y: 2500, VX: -100, VY: 0,
		DMG: 50, HP: 100, MaxHP: 100, Size: 10, Zone: 0, Life: 6,
	})
	r.updateBeamOrbs(0.05)
	// Шар достал тело игрока: урон нанесён и шар разбился.
	if p.State.HP >= 1000 {
		t.Fatalf("HP игрока не уменьшился: %v", p.State.HP)
	}
	if len(r.orbs) != 0 {
		t.Fatalf("шар должен разбиться о тело, осталось %d", len(r.orbs))
	}
}

func TestBeamOrbPetalDestroy(t *testing.T) {
	r := newTestRoom()
	initWorld()
	p := &Player{State: PlayerState{ID: "p1", X: 4000, Y: 2500, Size: 15, Zone: 0}}
	p.State.HP = 1000
	p.State.MaxHP = 1000
	// Петаль на Dist 200 от игрока (игрок далеко от шара; Size=15=playerSize,
	// чтобы petalWorldPos дал ровно (4200,2500)).
	p.petals = append(p.petals, Petal{Kind: "oval", Rarity: 0, HP: 10, MaxHP: 10, DMG: 9, Size: 8, Angle: 0, Dist: 200})
	r.players["p1"] = p

	// Шар прямо у петаля (петаль в (4200, 2500), шар в (4205, 2500)). Тело игрока
	// далеко (dist 195 > 15+10) — урон телу невозможен.
	r.orbs = append(r.orbs, BeamOrb{
		ID: "orb_2", X: 4205, Y: 2500, VX: 0, VY: 0,
		DMG: 5, HP: 5, MaxHP: 5, Size: 10, Zone: 0, Life: 6,
	})
	r.updateBeamOrbs(0.016)
	if len(r.orbs) != 0 {
		t.Fatalf("шар должен быть сбит петалями, осталось %d", len(r.orbs))
	}
	// Игрок НЕ получил урона телом от шара.
	if p.State.HP != 1000 {
		t.Fatalf("HP игрока изменился без контакта: %v", p.State.HP)
	}
}

func TestBeamOrbHomesToNearestPlayer(t *testing.T) {
	r := newTestRoom()
	initWorld()
	r.players["p1"] = &Player{State: PlayerState{ID: "p1", X: 900, Y: -300, Size: 30, Zone: 0}}

	r.orbs = append(r.orbs, BeamOrb{
		ID: "orb_3", X: 250, Y: 250, VX: 0, VY: 190,
		DMG: 5, HP: 100, MaxHP: 100, Size: 10, Zone: 0, Life: 6,
	})
	for i := 0; i < 40; i++ {
		r.updateBeamOrbs(1 / 20.0)
	}
	// Хоминг повернул скорость в сторону игрока (X+, Y−).
	if r.orbs[0].VX <= 0 || r.orbs[0].VY >= 0 {
		t.Fatalf("хоминг не повернул шар к игроку: v=(%v,%v)", r.orbs[0].VX, r.orbs[0].VY)
	}
}

func TestGoldenHPMult(t *testing.T) {
	norm := makeAdminMob("square", 0, 0, 0, 0, 1, "nothing")
	gold := makeAdminMob("square", 0, 0, 0, 0, 2, "gold")
	if !gold.Golden || norm.Golden {
		t.Fatalf("gold флаг неверный: norm=%v gold=%v", norm.Golden, gold.Golden)
	}
	if gold.HP != norm.HP*goldHPMult {
		t.Fatalf("gold HP=%v, want %v (×%v)", gold.HP, norm.HP*goldHPMult, goldHPMult)
	}
	// Золотой Fractal — ×50.
	nf := makeAdminMob("square", len(rarityNames)-2, 0, 0, 0, 3, "nothing")
	gf := makeAdminMob("square", len(rarityNames)-2, 0, 0, 0, 4, "gold")
	if gf.HP != nf.HP*goldBossHPMult {
		t.Fatalf("gold Fractal HP=%v, want %v", gf.HP, nf.HP*goldBossHPMult)
	}
}

// --- Beam orb: каденция ударов петалей, яд, цепь, бомба, руна ---

func TestBeamOrbPetalCadence(t *testing.T) {
	r := newTestRoom()
	initWorld()
	p := &Player{State: PlayerState{ID: "p1", X: 4000, Y: 2500, Size: 15, Zone: 0}}
	p.State.HP = 1000
	p.State.MaxHP = 1000
	// oval — atkCD=0.10, DMG=5, Dist=200 → петаль в (4200, 2500) рядом с шаром.
	p.petals = append(p.petals, Petal{Kind: "oval", Rarity: 0, HP: 100, MaxHP: 100, DMG: 5, Size: 8, Angle: 0, Dist: 200})
	p.Passive = true // отключаем хоминг шара, чтобы он стоял на месте
	r.players["p1"] = p

	r.orbs = append(r.orbs, BeamOrb{
		ID: "orb_1", X: 4205, Y: 2500, VX: 0, VY: 0,
		DMG: 5, HP: 1000, MaxHP: 1000, Size: 10, Zone: 0, Rarity: 0, Life: 100,
	})
	// Первый тик: OrbT=0 → удар (HP 1000→990). OrbT сбрасывается в atkCD.
	r.updateBeamOrbs(0.005)
	if r.orbs[0].HP != 1000-5*petalDmgMult {
		t.Fatalf("ожидаем один удар: hp=%v", r.orbs[0].HP)
	}
	// 10 следующих тиков по 0.01 = суммарно 0.1 → OrbT сходил до 0 → второй удар.
	for i := 0; i < 10; i++ {
		r.updateBeamOrbs(0.01)
	}
	if r.orbs[0].HP != 1000-10*petalDmgMult {
		t.Fatalf("ожидаем второй удар: hp=%v", r.orbs[0].HP)
	}
}

func TestBeamOrbPoisonDoT(t *testing.T) {
	r := newTestRoom()
	initWorld()
	r.orbs = append(r.orbs, BeamOrb{
		ID: "orb_1", X: 100, Y: 100, VX: 0, VY: 0,
		DMG: 5, HP: 30, MaxHP: 30, Size: 10, Zone: 0, Rarity: 0, Life: 100,
		PoisonDPS: 10, PoisonT: poisonDuration, // poisonDuration=3.5 → 35 урона
	})
	// 2 секунды: яд снимает 10×2=20 HP → HP≈10.
	for i := 0; i < 120; i++ {
		r.updateBeamOrbs(1 / 60.0)
	}
	if len(r.orbs) == 0 || r.orbs[0].HP < 5 || r.orbs[0].HP > 15 {
		t.Fatalf("яд не оттикал: orbs=%d hp=%v", len(r.orbs), r.orbs[0].HP)
	}
	// Ещё 2 секунды — яд добивает шар задолго до Life=100.
	for i := 0; i < 120; i++ {
		r.updateBeamOrbs(1 / 60.0)
	}
	if len(r.orbs) != 0 {
		t.Fatal("шар должен умереть от яда")
	}
}

func TestBeamOrbViperPetalPoisons(t *testing.T) {
	r := newTestRoom()
	initWorld()
	p := &Player{State: PlayerState{ID: "p1", X: 4000, Y: 2500, Size: 15, Zone: 0}}
	p.State.HP = 1000
	p.State.MaxHP = 1000
	// viper: Poison=true, DMG=10, Dist=200 → (4200,2500); atkCD=0.13.
	p.petals = append(p.petals, Petal{Kind: "viper", Rarity: 0, HP: 100, MaxHP: 100, DMG: 10, Size: 8, Angle: 0, Dist: 200})
	r.players["p1"] = p

	r.orbs = append(r.orbs, BeamOrb{
		ID: "orb_1", X: 4205, Y: 2500, VX: 0, VY: 0,
		DMG: 5, HP: 500, MaxHP: 500, Size: 10, Zone: 0, Rarity: 0, Life: 100,
	})
	r.updateBeamOrbs(0.016)
	if r.orbs[0].PoisonDPS != 10*petalDmgMult*poisonDpsMult {
		t.Fatalf("яд не висит: dps=%v", r.orbs[0].PoisonDPS)
	}
	if r.orbs[0].PoisonT <= 0 {
		t.Fatal("таймер яда нулевой")
	}
}

func TestBeamOrbSparkChainDestroys(t *testing.T) {
	r := newTestRoom()
	initWorld()
	// Орб рядом с точкой удара (dist=10), моб далеко — цепь бьёт орб.
	r.orbs = append(r.orbs, BeamOrb{
		ID: "orb_1", X: 110, Y: 100, VX: 0, VY: 0,
		DMG: 5, HP: 30, MaxHP: 30, Size: 10, Zone: 0, Life: 100,
	})
	r.mobs = append(r.mobs, Mob{
		Type: "hexagon", HP: 1000, Zone: 0, X: 9000, Y: 9000,
		Damage: map[string]float64{},
	})
	r.sparkChain("p1", 0, 100, 100, 20, "mob_1")
	if len(r.orbs) == 0 {
		t.Fatal("орб исчез (не должен)")
	}
	// Цепь дала 20 × (1 - sparkChainFalloff) урона.
	want := 30.0 - 20.0*(1-sparkChainFalloff)
	if r.orbs[0].HP != want {
		t.Fatalf("цепь не бьёт шар: hp=%v, want %v", r.orbs[0].HP, want)
	}
}

func TestBeamOrbBombSplashDamages(t *testing.T) {
	r := newTestRoom()
	initWorld()
	r.orbs = append(r.orbs, BeamOrb{
		ID: "orb_1", X: 150, Y: 100, VX: 0, VY: 0,
		DMG: 5, HP: 50, MaxHP: 50, Size: 10, Zone: 0, Life: 100,
	})
	// skipID == orb_1 → пропустит (бомба не бьёт того, кого уже снесли).
	r.bombSplash("p1", 0, 100, 100, 100, "orb_1")
	if r.orbs[0].HP != 50 {
		t.Fatalf("бомба не должна бить skipID: hp=%v", r.orbs[0].HP)
	}
	// Бомба попадает в шар (dist=50 < bombRadius=80).
	r.bombSplash("p1", 0, 100, 100, 100, "none")
	if r.orbs[0].HP != 50-100*bombDmgFrac {
		t.Fatalf("бомба не задела шар: hp=%v", r.orbs[0].HP)
	}
}

func TestBeamOrbRuneBoomDamages(t *testing.T) {
	r := newTestRoom()
	initWorld()
	src := &Mob{Type: "comet", Zone: 0, X: 100, Y: 100, Damage: map[string]float64{}}
	r.orbs = append(r.orbs, BeamOrb{
		ID: "orb_1", X: 200, Y: 100, VX: 0, VY: 0,
		DMG: 5, HP: 100, MaxHP: 100, Size: 10, Zone: 0, Life: 100,
	})
	r.runeBoom("p1", src, 50)
	if r.orbs[0].HP != 100-50*runeBoomMult {
		t.Fatalf("руна не задела шар: hp=%v", r.orbs[0].HP)
	}
}
