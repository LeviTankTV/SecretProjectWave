import io

p = "room.go"
s = io.open(p, encoding='utf-8').read()

def rep(old, new, tag, count=1):
    global s
    assert s.count(old) == count, "ANCHOR FAIL (%d found): %s" % (s.count(old), tag)
    s = s.replace(old, new, count)
    print("ok:", tag)

# 9) buildState: wave bar + ростер живых мобов
rep("""	b, _ := json.Marshal(msg)
	return b
}""",
"""	// Wave bar + ростер живых мобов (волновой режим).
	if r.waveMode {
		switch {
		case r.waveRunning && r.waveToSpawn > 0:
			msg.WavePhase = "spawn"
			if r.waveTotal > 0 {
				msg.WaveF1 = float64(r.waveSpawned) / float64(r.waveTotal)
			}
		case r.waveRunning:
			msg.WavePhase = "hold"
			msg.WaveF1 = 1
			if waveHoldDuration > 0 {
				msg.WaveF2 = 1 - r.waveHoldT/waveHoldDuration
				if msg.WaveF2 < 0 {
					msg.WaveF2 = 0
				}
			}
		}
		msg.WaveNum = r.waveNum
		// Ростер: живые мобы по паре (тип, редкость).
		type rosterKey struct {
			t string
			r int
		}
		counts := map[rosterKey]int{}
		for i := range r.mobs {
			if r.mobs[i].HP > 0 {
				counts[rosterKey{r.mobs[i].Type, r.mobs[i].Rarity}]++
			}
		}
		for k, n := range counts {
			mt := mobTypes[k.t]
			msg.WaveRoster = append(msg.WaveRoster, WaveRosterEntry{
				Type: k.t, Name: mt.Name, Color: mt.Color, Rarity: k.r, Count: n,
			})
		}
	}

	b, _ := json.Marshal(msg)
	return b
}""", "buildState wave")

# 10) awardMobKill: командный дроп (в wave-режиме — персональная копия каждому)
rep("""	for _, d := range list {
		if d.d < lootThreshold {
			continue // не нанёс порог от HP моба — лут не положен (анти-лутстил)
		}""",
"""	if r.waveMode {
		// Командный режим: дроп с мба падает ДЛЯ КАЖДОГО члена команды —
		// у каждого своя персональная копия (OwnerID), все могут подобрать.
		// Рецепт дропа считается один раз (по мобу), а не по вкладу игроков.
		if m.Type == "blackhole" && rand.Float64() >= blackHoleDropChance {
			// Singularity не выпал — дропа нет ни у кого.
		} else {
			extra := 0
			switch m.Rarity {
			case len(rarityNames) - 2:
				if rand.Float64() < 0.35 {
					extra = 1
				}
			case len(rarityNames) - 1:
				if rand.Float64() < 0.5 {
					extra = 1
				}
				if extra == 1 && rand.Float64() < 0.2 {
					extra = 2
				}
			}
			total := (1 + extra) * goldRolls
			for _, p := range r.players {
				owner := p.State.ID
				for n := 0; n < total; n++ {
					r.gpIDSeq++
					r.gp = append(r.gp, makeDropPetal(m, owner, r.gpIDSeq))
				}
			}
		}
	} else {
	for _, d := range list {
		if d.d < lootThreshold {
			continue // не нанёс порог от HP моба — лут не положен (анти-лутстил)
		}""", "awardMobKill team drops")

io.open(p, 'w', encoding='utf-8', newline='\n').write(s)
print("done")
