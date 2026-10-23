// Тест: цепь Spark должна бить 6 мобов рядом с крупным мобом (polygon/hexagon).
package main

import (
	"math"
	"testing"
)

func TestSparkChainSixMobs(t *testing.T) {
	initWorld()
	r := &Room{
		players: map[string]*Player{},
		clients: map[string]*Client{},
	}
	// Крупный hexagon-моб в центре, вокруг 6 мобов по кругу r=60.
	r.mobs = append(r.mobs, Mob{
		ID: "hex", Type: "hexagon", Rarity: 4, HP: 500, DMG: 3, Radius: 40, Zone: 8,
		Damage: map[string]float64{}, X: 1000, Y: 1000,
	})
	for i := 0; i < 6; i++ {
		a := float64(i) / 6 * 2 * math.Pi
		r.mobs = append(r.mobs, Mob{
			ID:     "n" + string(rune('a'+i)),
			Type:   "dot", Rarity: 3, HP: 500, DMG: 3, Radius: 18, Zone: 8,
			Damage: map[string]float64{},
			X:      1000 + math.Cos(a)*60, Y: 1000 + math.Sin(a)*60,
		})
	}
	r.sparkChain("p1", 8, 1000, 1000, 50, "hex")
	// Все 6 соседей должны получить урон (falloff: 50×0.75^k).
	for i := 1; i <= 6; i++ {
		m := &r.mobs[i]
		if m.HP >= 500 {
			t.Errorf("моб %s (%v,%v) не получил цепной урон: HP=%v", m.ID, m.X, m.Y, m.HP)
		}
	}
	if len(r.chains) != 1 {
		t.Fatalf("цепь не нарисована: %d", len(r.chains))
	}
	if len(r.chains[0].Points) != 7 {
		t.Errorf("точек в цепи = %d, want 7", len(r.chains[0].Points))
	}
}
