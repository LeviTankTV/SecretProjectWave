package main

import "testing"

// --- Враждебность команд ---

func TestTeamAreEnemies(t *testing.T) {
	cases := []struct {
		a, b int
		want bool
	}{
		{0, 0, false}, // обычные — друзья
		{0, 1, true},  // обычный vs рого
		{1, 0, true},
		{1, 1, true},  // два рого дерутся
		{2, 0, true},  // команда мобов vs обычный
		{2, 1, true},
		{2, 2, false}, // две команды мобов — друзья
		{0, 2, true},
	}
	for _, c := range cases {
		if got := teamAreEnemies(c.a, c.b); got != c.want {
			t.Errorf("teamAreEnemies(%d,%d)=%v, want %v", c.a, c.b, got, c.want)
		}
	}
}

// --- Снаряд враждебного игрока бьёт вражеского игрока и изнашивается ---

func TestProjectilePvPHitsPlayer(t *testing.T) {
	r := &Room{
		players: map[string]*Player{},
		clients: map[string]*Client{},
		mobs:    []Mob{},
	}
	r.players["attacker"] = &Player{State: PlayerState{ID: "attacker", X: 0, Y: 0, Size: playerSize, Zone: 0}, Team: 1}
	r.players["victim"] = &Player{State: PlayerState{ID: "victim", X: 50, Y: 50, Size: playerSize, Zone: 0, HP: 100, MaxHP: 100}, Team: 0}
	// Снаряд владельца (команда 1) летит в тело жертвы.
	r.proj = append(r.proj, r.newProjectile("attacker", 1, 0, 50, 50, 0, 0, 8, 7, 8, 0xffffff, "blade", bladeFlyLife, true))
	r.updateProjectiles(1.0 / 20.0)
	if r.players["victim"].State.HP != 92 {
		t.Errorf("жертва не получила урон: HP=%v, want 92", r.players["victim"].State.HP)
	}
	if len(r.proj) != 1 {
		t.Fatalf("пробивающий снаряд исчез на теле: proj=%d, want 1", len(r.proj))
	}
	if r.proj[0].HP != 7-pvpMissileBodyWear {
		t.Errorf("снаряд не износился от тела: HP=%v, want %v", r.proj[0].HP, 7-pvpMissileBodyWear)
	}
}

// --- Снаряд дружелюбного игрока игнорирует других игроков ---

func TestProjectileFriendlyIgnoresPlayers(t *testing.T) {
	r := &Room{
		players: map[string]*Player{},
		clients: map[string]*Client{},
		mobs:    []Mob{},
	}
	r.players["p1"] = &Player{State: PlayerState{ID: "p1", X: 0, Y: 0, Size: playerSize, Zone: 0}, Team: 0}
	r.players["p2"] = &Player{State: PlayerState{ID: "p2", X: 50, Y: 50, Size: playerSize, Zone: 0, HP: 100, MaxHP: 100}, Team: 0}
	// Дружелюбный снаряд (команда 0) пролетает через другого игрока без урона.
	r.proj = append(r.proj, r.newProjectile("p1", 0, 0, 50, 50, 0, 0, 8, 7, 8, 0xffffff, "blade", bladeFlyLife, true))
	r.updateProjectiles(1.0 / 20.0)
	if r.players["p2"].State.HP != 100 {
		t.Errorf("дружелюбный снаряд нанёс урон: HP=%v, want 100", r.players["p2"].State.HP)
	}
}

// --- Снаряд бьёт петали вражеского игрока (взаимный износ) ---

func TestProjectilePvPHitsPetal(t *testing.T) {
	r := &Room{
		players: map[string]*Player{},
		clients: map[string]*Client{},
		mobs:    []Mob{},
	}
	r.players["attacker"] = &Player{State: PlayerState{ID: "attacker", X: 0, Y: 0, Size: playerSize, Zone: 0}, Team: 1}
	r.players["victim"] = &Player{State: PlayerState{ID: "victim", X: 0, Y: 0, Size: playerSize, Zone: 0}, Team: 0}
	// Петаль жертвы на орбите в точке попадания снаряда.
	r.players["victim"].petals = append(r.players["victim"].petals, Petal{
		Kind: "oval", HP: 10, MaxHP: 10, DMG: 5, Size: 8, Angle: 0, Dist: 50,
	})
	// Снаряд владельца летит на позицию петеля жертвы (50,0).
	r.proj = append(r.proj, r.newProjectile("attacker", 1, 0, 50, 0, 0, 0, 8, 7, 8, 0xffffff, "blade", bladeFlyLife, true))
	r.updateProjectiles(1.0 / 20.0)
	oPet := &r.players["victim"].petals[0]
	if oPet.HP != 2 {
		t.Errorf("петаль жертвы не получила урон: HP=%v, want 2 (10-8)", oPet.HP)
	}
	if r.proj[0].HP != 7-5*0.3 {
		t.Errorf("снаряд не износился от петеля: HP=%v, want %v", r.proj[0].HP, 7-5*0.3)
	}
}
