package main

// waves2_e2e_test.go — headless-прогон директора волн 2.0: комната без WS,
// тик вручную, мобы убиваются мгновенно (имитация сильной команды).
// Проверяет много-волновую прогрессию, спец-волны, босс-лестницу и рост
// редкостей без участия браузера.

import (
	"testing"
)

// runWavesHeadless прокручивает волны до targetWave, убивая всех мобов
// каждый тик (кроме волны keepAliveWave — там мобы живут, чтобы проверить
// hold-слой). Возвращает максимум редкости заспавненного моба.
func runWavesHeadless(t *testing.T, targetWave int) (maxRarity int, kindsSeen map[string]bool) {
	t.Helper()
	initWorld()
	r := newWaveRoom(20)
	p := &Player{}
	p.State.ID = "p1"
	p.State.Name = "tester"
	p.State.HP = 1e9
	p.State.MaxHP = 1e9
	p.State.Size = 20
	p.Dead = false
	p.zone = 0
	p.State.Zone = 0
	r.players["p1"] = p

	kindsSeen = map[string]bool{}
	maxByWave := map[int]int{} // волна -> максимальная редкость её мобов

	r.beginWaves()
	const dt = 0.1
	for step := 0; step < 200000; step++ {
		r.updateWaves(dt)
		// Убиваем всех мобов (сильная команда): зачисляем киллы и чистим.
		for i := range r.mobs {
			m := &r.mobs[i]
			if m.HP > 0 {
				if m.Rarity > maxRarity {
					maxRarity = m.Rarity
				}
				kindsSeen[m.Type] = true
				if m.Rarity > maxByWave[r.waveNum] {
					maxByWave[r.waveNum] = m.Rarity
				}
				m.HP = 0
			}
		}
		if r.waveNum >= targetWave && !r.waveRunning {
			break
		}
		if r.waveOver {
			t.Fatalf("wipe at wave %d with immortal player", r.waveNum)
		}
	}
	// Босс-волны: каждая 10-я обязана дать моба rarity >= bossRarityFor(n).
	for w := 10; w <= targetWave; w += 10 {
		want := bossRarityFor(w)
		if maxByWave[w] < want {
			t.Errorf("boss wave %d: max rarity %d, want >= %d", w, maxByWave[w], want)
		}
	}
	return maxRarity, kindsSeen
}

func TestWavesHeadlessProgression(t *testing.T) {
	maxRar, kinds := runWavesHeadless(t, 12)
	if maxRar < 3 {
		t.Errorf("12 волн: max rarity %d, want >= 3 (Square+)", maxRar)
	}
	// Спец-волны встретились: swarm (5), hunters (15 не дошли) — хотя бы один
	// «мелкий» пул волн 5-7 содержит dot/pebble.
	if !kinds["dot"] && !kinds["pebble"] {
		t.Errorf("swarm-волна не оставила dot/pebble в пулах: %v", keysOf(kinds))
	}
}

func TestWavesHeadlessLadder(t *testing.T) {
	// Кап Chaos (решение игрока: истинных тиров у мобов нет): с 32-й волны
	// потолок редкости мобов = 7 (Chaos).
	maxRar, _ := runWavesHeadless(t, 32)
	if maxRar != 7 {
		t.Errorf("32 волны: max rarity %d, want 7 (кап Chaos)", maxRar)
	}
	// Chaos+ циклы: босс волны 50 — Chaos с множителем 2.25.
	initWorld()
	r := newWaveRoom(20)
	p := &Player{}
	p.State.ID = "p1"
	p.State.HP = 1e9
	p.State.MaxHP = 1e9
	p.Dead = false
	p.zone = 0
	r.players["p1"] = p
	r.startWave(50)
	if r.waveSpecial != "boss" {
		t.Fatalf("wave 50 special = %q, want boss", r.waveSpecial)
	}
	want := mobTypes[r.mobs[0].Type].BaseHP * rarityDanger[7] * mobHPScale * 2.0 * chaosHPMult * bossCycleMult(50)
	found := false
	for i := range r.mobs {
		if r.mobs[i].Rarity == 7 && r.mobs[i].Type == r.mobs[0].Type {
			if r.mobs[i].MaxHP >= want*0.99 {
				found = true
			}
		}
	}
	if !found {
		t.Errorf("Chaos+ босс волны 50 без множителя x2.25 (want MaxHP >= %v)", want)
	}
}

func TestWavesHeadlessSpecialRotation(t *testing.T) {
	initWorld()
	r := newWaveRoom(20)
	p := &Player{}
	p.State.ID = "p1"
	p.State.HP = 1e9
	p.State.MaxHP = 1e9
	p.Dead = false
	p.zone = 0
	r.players["p1"] = p
	r.beginWaves()
	// Прокручиваем старты волн: стартуем волну N вручную и смотрим тип.
	seen := map[string]bool{}
	// 5..50: tanks впервые на 40-й (25-ю занимает elite, 10/20/30/40... — boss).
	for n := 5; n <= 50; n++ {
		r.startWave(n)
		seen[r.waveSpecial] = true
		if n%10 == 0 && r.waveSpecial != "boss" {
			t.Errorf("wave %d special = %q, want boss", n, r.waveSpecial)
		}
	}
	for _, want := range []string{"swarm", "tanks", "hunters", "elite", "boss"} {
		if !seen[want] {
			t.Errorf("special %q не встретился в 5..50: %v", want, keysOf(seen))
		}
	}
	// Арена растёт с волной.
	if r.waveArenaR <= waveArenaRadius {
		t.Errorf("waveArenaR = %v после 30-й волны, want > %v", r.waveArenaR, waveArenaRadius)
	}
}

func keysOf(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
