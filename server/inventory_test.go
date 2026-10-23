package main

import "testing"

// Поднятая петаль попадает в инвентарь как стак (по виду+редкости);
// орбита автозаполняется до капа уровня, излишки копятся в стаке.
func TestAddPetalToPlayer(t *testing.T) {
	p := &Player{}
	p.State.ID = "t0"
	p.State.Level = 0 // cap = 5
	p.hotbars = [2][]PetalDef{{}, {}}
	r := newTestRoom()
	r.players["t0"] = p

	g := func() GroundPetal {
		return GroundPetal{Kind: "oval", HP: 5, MaxHP: 5, DMG: 3, Size: 8, Color: 0xecf0f1, Rarity: 0}
	}

	for i := 0; i < 8; i++ {
		r.addPetalToPlayer(p, g())
	}
	// Подобранное заполняет пустые слоты обеих линеек (5+3), инвентарь пуст.
	if len(p.hotbars[0]) != 5 {
		t.Fatalf("bar0 = %d, want 5", len(p.hotbars[0]))
	}
	if len(p.hotbars[1]) != 3 {
		t.Fatalf("bar1 = %d, want 3", len(p.hotbars[1]))
	}
	if len(p.inventory) != 0 {
		t.Fatalf("inventory stacks = %d, want 0 (всё надето)", len(p.inventory))
	}
	if len(p.petals) != petalCapForLevel(0) {
		t.Fatalf("orbit = %d, want cap %d", len(p.petals), petalCapForLevel(0))
	}
}

// Разные редкости одного вида — разные стаки; сортируются по убыванию редкости.
func TestInventoryStackAndSort(t *testing.T) {
	p := &Player{}
	p.State.Level = 10 // большая ёмкость, чтобы всё падало в запас
	r := &Room{}

	r.forceIntoInventory(p, "oval", 0)
	r.forceIntoInventory(p, "oval", 2)
	r.forceIntoInventory(p, "oval", 2)
	r.forceIntoInventory(p, "thorns", 4)
	r.forceIntoInventory(p, "oval", 0)

	// 5 предметов → 3 стака: {thorns r4, oval r2, oval r0}
	if len(p.inventory) != 3 {
		t.Fatalf("stacks = %d, want 3", len(p.inventory))
	}
	if p.inventory[0].Rarity != 4 {
		t.Fatalf("first stack rarity = %d, want 4 (sorted desc)", p.inventory[0].Rarity)
	}
	if p.inventory[1].Rarity != 2 {
		t.Fatalf("second stack rarity = %d, want 2", p.inventory[1].Rarity)
	}
	if p.inventory[2].Rarity != 0 {
		t.Fatalf("third stack rarity = %d, want 0", p.inventory[2].Rarity)
	}
	if p.inventory[1].Count != 2 {
		t.Fatalf("oval r2 count = %d, want 2", p.inventory[1].Count)
	}
	if p.inventory[2].Count != 2 {
		t.Fatalf("oval r0 count = %d, want 2", p.inventory[2].Count)
	}
}