import io

p = "room.go"
s = io.open(p, encoding='utf-8').read()

def rep(old, new, tag, count=1):
    global s
    assert s.count(old) == count, "ANCHOR FAIL (%d found): %s" % (s.count(old), tag)
    s = s.replace(old, new, count)
    print("ok:", tag)

# 3) gameLoop: stop select
rep("""	ticker := time.NewTicker(time.Second / time.Duration(r.tickRate))
	defer ticker.Stop()
	for range ticker.C {""",
"""	ticker := time.NewTicker(time.Second / time.Duration(r.tickRate))
	defer ticker.Stop()
	for {
		select {
		case <-r.stopCh:
			return
		case <-ticker.C:
		}""", "gameloop stop")

# 4) autosaveLoop: stop select (аккуратно с исходным secs++)
rep("""	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	secs := 0
	for range ticker.C {
		secs++""",
"""	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	secs := 0
	for {
		select {
		case <-r.stopCh:
			return
		case <-ticker.C:
		}
		secs++""", "autosave stop")

# 5) tick: wave-режим вместо зонного спавна
rep("""	// Периодический спавн мобов (раз в 5-15с), если зона недонаполнена.
	for i := 0; i < len(zones); i++ {""",
"""	// Волновой режим: директор волн вместо зонного спавна + круглая арена.
	if r.waveMode {
		r.updateWaves(dt)
		for i := range r.mobs {
			if r.mobs[i].HP > 0 {
				r.mobs[i].X, r.mobs[i].Y = r.clampArena(r.mobs[i].X, r.mobs[i].Y, r.mobs[i].Radius)
			}
		}
	} else {
	for i := 0; i < len(zones); i++ {""", "tick wave guard")

# 6) Закрываем else-блок перед комментарием про Fractal-боссов (уникальный якорь)
rep("""	// Fractal-боссов больше НЕТ по таймеру: они роллятся обычным спавном
	// 1/1500 в верхней полосе L2-зон (см. l2RarityBands / spawnMob).""",
"""	}
	// Fractal-боссов больше НЕТ по таймеру: они роллятся обычным спавном
	// 1/1500 в верхней полосе L2-зон (см. l2RarityBands / spawnMob).""", "tick else close")

# 7) updatePlayer: чекпоинт/порталы только вне волнового режима + клэмп арены
rep("""	r.movePlayer(p, dt)

	// Чекпоинт (версия Т): встав на нижне-центральную площадку сейф-зоны,
	// игрок закрепляет спавнпоинт этой зоны. Только ПЕРВЫЙ уровень (0-7):
	// со второго уровня смерть/релог всегда возвращают в первый.
	// Портал сам по себе чекпоинт НЕ ставит — надо дойти до площадки.
	if p.transferT <= 0 && p.zone < secondTierOffset {
		z := zones[p.zone]
		if p.State.X > z.X+z.W/2-checkpointPadHalfW &&
			p.State.X < z.X+z.W/2+checkpointPadHalfW &&
			p.State.Y > z.Y+z.H-safeZoneH {
			if p.checkpoint != p.zone {
				p.checkpoint = p.zone
				nick := p.State.Name
				cz := p.zone
				go saveCheckpoint(nick, cz)
			}
		}
	}

	if p.portalCD <= 0 {""",
"""	r.movePlayer(p, dt)

	// Волновой режим: круглый «басейн» — прижимаем тело игрока внутрь круга,
	// чекпоинты/порталы не нужны (весь матч идёт в круге).
	if r.waveMode {
		p.State.X, p.State.Y = r.clampArena(p.State.X, p.State.Y, p.State.Size)
	} else {
	// Чекпоинт (версия Т): встав на нижне-центральную площадку сейф-зоны,
	// игрок закрепляет спавнпоинт этой зоны. Только ПЕРВЫЙ уровень (0-7):
	// со второго уровня смерть/релог всегда возвращают в первый.
	// Портал сам по себе чекпоинт НЕ ставит — надо дойти до площадки.
	if p.transferT <= 0 && p.zone < secondTierOffset {
		z := zones[p.zone]
		if p.State.X > z.X+z.W/2-checkpointPadHalfW &&
			p.State.X < z.X+z.W/2+checkpointPadHalfW &&
			p.State.Y > z.Y+z.H-safeZoneH {
			if p.checkpoint != p.zone {
				p.checkpoint = p.zone
				nick := p.State.Name
				cz := p.zone
				go saveCheckpoint(nick, cz)
			}
		}
	}
	}

	if p.portalCD <= 0 && !r.waveMode {""", "updatePlayer wave")

# 8) buildWorld: волновые поля
rep("""	w.Portals = make([]WorldPortalMsg, 0, len(portals))""",
"""	// Волновой режим: вместо зон/порталов клиент рисует круглую арену.
	if r.waveMode {
		w.Wave = true
		w.ArenaX, w.ArenaY = r.waveArenaCenter()
		w.ArenaR = waveArenaRadius
		w.ArenaZone = r.waveZone
	}
	w.Portals = make([]WorldPortalMsg, 0, len(portals))""", "buildWorld wave")

io.open(p, 'w', encoding='utf-8', newline='\n').write(s)
print("done")
