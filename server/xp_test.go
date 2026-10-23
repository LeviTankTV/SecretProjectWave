package main

import "testing"

func TestPetalCapForLevel(t *testing.T) {
	// Кривая родителя (порт): ранние капы быстрее (волны короткие).
	cases := []struct{ lvl, want int }{
		{0, 5}, {1, 5}, {2, 6}, {3, 6}, {4, 7}, {7, 7},
		{8, 8}, {15, 8}, {16, 9}, {24, 9},
		{25, 10}, {49, 10}, {99, 10},
	}
	for _, c := range cases {
		if got := petalCapForLevel(c.lvl); got != c.want {
			t.Errorf("petalCapForLevel(%d)=%d, want %d", c.lvl, got, c.want)
		}
	}
}

func TestAwardXPLeveling(t *testing.T) {
	// Кривая родителя (порт): с 0 на 1 нужно 45×1.11^0 = 45 XP.
	p := &Player{}
	p.State.Level = 0
	p.State.XPMax = xpForLevel(0)
	if !p.awardXP(45) {
		t.Errorf("expected levelup")
	}
	if p.State.Level != 1 {
		t.Errorf("level=%d, want 1", p.State.Level)
	}
	if p.State.XP != 0 {
		t.Errorf("xp=%v, want 0", p.State.XP)
	}
	if p.awardXP(5) {
		t.Errorf("unexpected levelup")
	}
	if p.State.XP != 5 {
		t.Errorf("xp=%v, want 5", p.State.XP)
	}
}

func TestMobXP(t *testing.T) {
	// killXPFor (порт родителя): детерминированный, растёт с редкостью,
	// анти-фарм tierGap режет редкости ниже тира игрока.
	m := &Mob{Type: "dot", Rarity: 0}
	if killXPFor(m, 0) <= 0 {
		t.Errorf("killXPFor should be > 0, got %v", killXPFor(m, 0))
	}
	// Rare mob should give more than common.
	rare := &Mob{Type: "dot", Rarity: 2}
	if killXPFor(rare, 0) < killXPFor(m, 0) {
		t.Errorf("rare mob XP should be > common mob XP")
	}
	// Анти-фарм: игрок 40-го уровня (тир 5) с Point-моба получает меньше.
	if killXPFor(m, 40) >= killXPFor(m, 0) {
		t.Errorf("tierGap should reduce low-rarity XP for high-level player")
	}
	// Детерминизм: два вызова — одинаковое число.
	if killXPFor(rare, 10) != killXPFor(rare, 10) {
		t.Errorf("killXPFor must be deterministic")
	}
}
