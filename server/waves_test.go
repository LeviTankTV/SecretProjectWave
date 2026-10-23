package main

// waves_test.go - тесты волнового директора и лобби.

import "testing"

func TestWaveSpecialFor(t *testing.T) {
	// 2.0: каждая 10-я — босс лестницы, каждая 25-я (не кратная 10) — элита.
	cases := map[int]string{
		1: "", 3: "", 4: "",
		5:  "swarm",
		10: "boss",
		15: "hunters",
		20: "boss",
		25: "elite",
		30: "boss",
		75: "elite",
	}
	for n, want := range cases {
		if got := waveSpecialFor(n); got != want {
			t.Errorf("waveSpecialFor(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestWaveLadder(t *testing.T) {
	// Лестница боссов 2.0: 10 -> Hexagon(5), 30 -> Chaos(7), 40 -> Dot(8),
	// 110+ -> Origin(15).
	if got := bossRarityFor(10); got != 5 {
		t.Errorf("bossRarityFor(10) = %d, want 5", got)
	}
	if got := bossRarityFor(30); got != 7 {
		t.Errorf("bossRarityFor(30) = %d, want 7", got)
	}
	if got := bossRarityFor(40); got != 8 {
		t.Errorf("bossRarityFor(40) = %d, want 8 (Dot)", got)
	}
	if got := bossRarityFor(110); got != 15 {
		t.Errorf("bossRarityFor(110) = %d, want 15 (Origin)", got)
	}
	// Полная лестница: Origin в потолке с волны 28, Dot в пуле с ~16-й.
	if _, ceil := waveRarityFor(28); ceil != 15 {
		t.Errorf("waveRarityFor(28) ceil = %d, want 15", ceil)
	}
	if _, ceil := waveRarityFor(16); ceil != 9 {
		t.Errorf("waveRarityFor(16) ceil = %d, want 9 (Dot+)", ceil)
	}
	// Пулы истинной лестницы валидны.
	for rar := 8; rar <= 15; rar++ {
		pool := waveTrueLadderPool(rar)
		if len(pool) == 0 {
			t.Fatalf("waveTrueLadderPool(%d) empty", rar)
		}
		for _, k := range pool {
			if _, ok := mobTypes[k]; !ok {
				t.Fatalf("waveTrueLadderPool(%d): unknown mob %q", rar, k)
			}
		}
	}
	// Арена растёт и capped.
	if got := arenaRadiusFor(0); got != waveArenaRadius {
		t.Errorf("arenaRadiusFor(0) = %v, want %v", got, waveArenaRadius)
	}
	if got := arenaRadiusFor(200); got != waveArenaMaxR {
		t.Errorf("arenaRadiusFor(200) = %v, want %v", got, waveArenaMaxR)
	}
}

func TestWaveRarityBounds(t *testing.T) {
	for n := 1; n <= 60; n++ {
		floor, ceil := waveRarityFor(n)
		if floor < 0 || floor >= len(rarityNames) || ceil < 0 || ceil > len(rarityNames)-1 {
			t.Fatalf("waveRarityFor(%d) out of range: %d..%d", n, floor, ceil)
		}
		if ceil < floor {
			t.Fatalf("waveRarityFor(%d): ceil < floor", n)
		}
		// Ролл всегда внутри диапазона.
		for i := 0; i < 50; i++ {
			r := rollWaveRarity(floor, ceil)
			if r < floor || r > ceil {
				t.Fatalf("rollWaveRarity(%d,%d) = %d", floor, ceil, r)
			}
		}
	}
}

func TestZoneMobPoolsValid(t *testing.T) {
	// Пулы всех зон арены (0-7) непусты и состоят из известных типов.
	for zone := 0; zone <= 7; zone++ {
		for i := 0; i < 200; i++ {
			name := randMobType(zone)
			if _, ok := mobTypes[name]; !ok {
				t.Fatalf("randMobType(%d): unknown mob type %q", zone, name)
			}
		}
	}
}

func TestGenLobbyCode(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		code := genLobbyCode()
		if len(code) != 6 {
			t.Fatalf("code %q is not 6 chars", code)
		}
		for _, c := range code {
			if c < '0' || c > '9' {
				t.Fatalf("code %q has non-digit", code)
			}
		}
		seen[code] = true
	}
}

func TestTrimCode(t *testing.T) {
	cases := map[string]string{
		"123456":    "123456",
		"12ab34":    "1234",
		" 77-77":    "7777",
		"123456789": "123456",
		"":          "",
	}
	for in, want := range cases {
		if got := trimCode(in); got != want {
			t.Errorf("trimCode(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLobbyCreateJoinLeave(t *testing.T) {
	l := createLobby("testerHost", true)
	if l.Code == "" || len(l.Code) != 6 {
		t.Fatalf("bad lobby code %q", l.Code)
	}
	if lobbyForNick("testerHost") != l {
		t.Fatal("host not in own lobby")
	}
	l2 := joinLobby("testerGuest", l.Code)
	if l2 != l {
		t.Fatal("guest failed to join by code")
	}
	if len(l.Members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(l.Members))
	}
	if joinLobby("testerOther", "000000") != nil {
		t.Fatal("join to nonexistent lobby must fail")
	}
	leaveLobby("testerGuest")
	leaveLobby("testerHost")
	if lobbyForNick("testerHost") != nil {
		t.Fatal("host still in lobby after leave")
	}
}

func TestWaveRoomBasic(t *testing.T) {
	r := newWaveRoom(20)
	defer r.close()
	if !r.waveMode {
		t.Fatal("wave room must have waveMode")
	}
	// Пустая комната не должна стартовать волны.
	r.mu.Lock()
	r.updateWaves(1.0)
	r.updateWaves(1.0)
	if r.waveRunning || r.waveNum != 0 {
		t.Fatal("waves must not start without players")
	}
	r.mu.Unlock()
}

func TestWaveProgression(t *testing.T) {
	r := newWaveRoom(20)
	defer r.close()
	r.mu.Lock()
	defer r.mu.Unlock()
	// Волну нельзя начать без игроков — симулируем вручную последовательность.
	for n := 1; n <= 12; n++ {
		r.startWave(n)
		if r.waveNum != n || !r.waveRunning {
			t.Fatalf("wave %d did not start", n)
		}
		if r.waveToSpawn <= 0 {
			t.Fatalf("wave %d has nothing to spawn", n)
		}
		if !r.waveMode {
			t.Fatal("waveMode lost")
		}
	}
}
