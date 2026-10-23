package main

import "testing"

func newTestRoom() *Room {
	return &Room{
		players:  map[string]*Player{},
		clients:  map[string]*Client{},
		usedNiks: map[string]int{},
		usedIPs:  map[string]bool{},
	}
}

func registerPlayer(r *Room, id string, level int) *Player {
	p := &Player{State: PlayerState{ID: id, Level: level}}
	p.hotbars = [2][]PetalDef{{}, {}}
	p.inventory = []InvStack{}
	r.players[id] = p
	return p
}

// equip: ставит предмет из стака в слот хотбара, списывая один из стака.
func TestEquipPetal(t *testing.T) {
	r := newTestRoom()
	registerPlayer(r, "t1", 0)
	r.forceIntoInventory(r.players["t1"], "oval", 0)
	r.forceIntoInventory(r.players["t1"], "thorns", 1)

	r.EquipPetal("t1", 0, 0, "oval", 0)
	if len(r.players["t1"].hotbars[0]) == 0 || r.players["t1"].hotbars[0][0].Kind != "oval" {
		t.Fatalf("hotbar0[0] = %+v, want oval", r.players["t1"].hotbars[0])
	}
	if i := r.findStack(r.players["t1"], "oval", 0); i >= 0 {
		t.Fatalf("oval stack should be empty after equip")
	}
	if len(r.players["t1"].hotbars[0]) != 1 {
		t.Fatalf("hotbar0 len = %d, want 1", len(r.players["t1"].hotbars[0]))
	}
}

// equip в занятый слот возвращает старый предмет в запас.
func TestEquipReplacesSlot(t *testing.T) {
	r := newTestRoom()
	registerPlayer(r, "t2", 0)
	r.forceIntoInventory(r.players["t2"], "oval", 0)
	r.forceIntoInventory(r.players["t2"], "rock", 2)

	r.EquipPetal("t2", 0, 0, "oval", 0)
	r.EquipPetal("t2", 0, 0, "rock", 2)
	if r.players["t2"].hotbars[0][0].Kind != "rock" {
		t.Fatalf("hotbar0[0] = %+v, want rock", r.players["t2"].hotbars[0][0])
	}
	if i := r.findStack(r.players["t2"], "oval", 0); i < 0 || r.players["t2"].inventory[i].Count != 1 {
		t.Fatalf("oval should be back in inventory count=1")
	}
}

// unequip снимает петаль со слота в запас.
func TestUnequipPetal(t *testing.T) {
	r := newTestRoom()
	registerPlayer(r, "t3", 0)
	r.forceIntoInventory(r.players["t3"], "thorns", 1)
	r.EquipPetal("t3", 0, 0, "thorns", 1)

	r.UnequipPetal("t3", 0, 0)
	if _, has := r.hotbarAt(r.players["t3"], 0, 0); has {
		t.Fatalf("slot 0 should be empty after unequip")
	}
	if i := r.findStack(r.players["t3"], "thorns", 1); i < 0 || r.players["t3"].inventory[i].Count != 1 {
		t.Fatalf("thorns should be back in inventory count=1")
	}
}

// drop снимает один из стака и начисляет XP.
func TestDropPetal(t *testing.T) {
	r := newTestRoom()
	registerPlayer(r, "t4", 0)
	r.forceIntoInventory(r.players["t4"], "oval", 0)
	before := r.players["t4"].State.XP

	r.DropPetal("t4", "oval", 0)
	if i := r.findStack(r.players["t4"], "oval", 0); i >= 0 {
		t.Fatalf("oval stack should be removed after drop")
	}
	if r.players["t4"].State.XP <= before {
		t.Fatalf("XP not increased after drop: %v -> %v", before, r.players["t4"].State.XP)
	}
}

// dropHotbar выбрасывает надетую петаль и начисляет XP.
func TestDropHotbar(t *testing.T) {
	r := newTestRoom()
	registerPlayer(r, "t5", 0)
	r.forceIntoInventory(r.players["t5"], "thorns", 3)
	r.EquipPetal("t5", 0, 0, "thorns", 3)

	r.DropHotbar("t5", 0, 0)
	if _, has := r.hotbarAt(r.players["t5"], 0, 0); has {
		t.Fatalf("slot should be empty after dropHotbar")
	}
	if i := r.findStack(r.players["t5"], "thorns", 3); i >= 0 {
		t.Fatalf("thorns should not return to inventory after drop")
	}
}

// move: перетаскивание хотбар->хотбар меняет петали местами (swap).
func TestMoveSwapHotbar(t *testing.T) {
	r := newTestRoom()
	registerPlayer(r, "t6", 0)
	r.forceIntoInventory(r.players["t6"], "oval", 0)
	r.forceIntoInventory(r.players["t6"], "thorns", 1)
	r.EquipPetal("t6", 0, 0, "oval", 0)
	r.EquipPetal("t6", 0, 1, "thorns", 1)

	// Тащим петаль из слота 0 в слот 1 — должны поменяться местами.
	from := map[string]interface{}{"src": "hot", "bar": float64(0), "slot": float64(0)}
	r.MovePetal("t6", from, 0, 1)
	if r.players["t6"].hotbars[0][0].Kind != "thorns" {
		t.Fatalf("slot0 = %+v, want thorns", r.players["t6"].hotbars[0][0])
	}
	if r.players["t6"].hotbars[0][1].Kind != "oval" {
		t.Fatalf("slot1 = %+v, want oval", r.players["t6"].hotbars[0][1])
	}
	if i := r.findStack(r.players["t6"], "oval", 0); i >= 0 {
		t.Fatalf("oval stack should be empty (всё надето)")
	}
}

// move: из инвентаря в занятый слот — новый на слот, старый в инвентарь.
func TestMoveInventoryToOccupied(t *testing.T) {
	r := newTestRoom()
	registerPlayer(r, "t7", 0)
	r.forceIntoInventory(r.players["t7"], "oval", 0)
	r.forceIntoInventory(r.players["t7"], "rock", 2)
	r.EquipPetal("t7", 0, 0, "oval", 0)

	// Тащим rock из инвентаря в слот с oval -> овал в инвентарь.
	from := map[string]interface{}{"src": "inv", "kind": "rock", "rarity": float64(2)}
	r.MovePetal("t7", from, 0, 0)
	if r.players["t7"].hotbars[0][0].Kind != "rock" {
		t.Fatalf("slot0 = %+v, want rock", r.players["t7"].hotbars[0][0])
	}
	if i := r.findStack(r.players["t7"], "oval", 0); i < 0 || r.players["t7"].inventory[i].Count != 1 {
		t.Fatalf("oval should be back in inventory count=1")
	}
}

// move: хотбар -> инвентарь (unequip) через пустой слот не нужен: движение на
// инвентарь делается через drop на панель (unequip), но верхнеуровнево проверяем
// что перенос инвентарь -> пустой слот работает.
func TestMoveInventoryToEmpty(t *testing.T) {
	r := newTestRoom()
	registerPlayer(r, "t8", 0)
	r.forceIntoInventory(r.players["t8"], "thorns", 1)

	from := map[string]interface{}{"src": "inv", "kind": "thorns", "rarity": float64(1)}
	r.MovePetal("t8", from, 0, 2)
	if r.players["t8"].hotbars[0][2].Kind != "thorns" {
		t.Fatalf("slot2 = %+v, want thorns", r.players["t8"].hotbars[0][2])
	}
	if i := r.findStack(r.players["t8"], "thorns", 1); i >= 0 {
		t.Fatalf("thorns should be consumed")
	}
}

// move: инвентарь -> верхний слот (кап-1) работает; слот ЗА капом отвергается.
func TestMoveCapBoundary(t *testing.T) {
	r := newTestRoom()
	registerPlayer(r, "t9", 0) // кап = 5 (уровень 0-4)

	// В слот 4 (последний допустимый, cap-1) — ОК.
	r.forceIntoInventory(r.players["t9"], "thorns", 1)
	from := map[string]interface{}{"src": "inv", "kind": "thorns", "rarity": float64(1)}
	r.MovePetal("t9", from, 0, 4)
	if r.players["t9"].hotbars[0][4].Kind != "thorns" {
		t.Fatalf("slot4 = %+v, want thorns (within cap)", r.players["t9"].hotbars[0])
	}

	// В слот 5 (>= cap) — должно быть отклонено, хотбар не должен вырасти до 6.
	r.forceIntoInventory(r.players["t9"], "rock", 0)
	from2 := map[string]interface{}{"src": "inv", "kind": "rock", "rarity": float64(0)}
	r.MovePetal("t9", from2, 0, 5)
	if len(r.players["t9"].hotbars[0]) >= 6 {
		t.Fatalf("hotbar grew beyond cap: %d slots", len(r.players["t9"].hotbars[0]))
	}
	if _, has := r.hotbarAt(r.players["t9"], 0, 5); has {
		t.Fatalf("slot5 should be rejected (cap=5)")
	}
}

// Профильная операция (меню-REST): кап по уровню. Уровень 5 => слот 5 доступен.
func TestProfileMoveCapByLevel(t *testing.T) {
	sp := &statelessProfile{inv: []InvStack{
		{PetalDef: PetalDef{Kind: "thorns", Rarity: 1, HP: 9, MaxHP: 9, DMG: 7, Size: 5, Name: "thorns"}, Count: 1},
	}, hotbars: [2][]PetalDef{{}, {}}, activeBar: 0, nick: "x", level: 5}

	okMove := sp.move(map[string]interface{}{"src": "inv", "kind": "thorns", "rarity": float64(1)}, 0, 5)
	if !okMove {
		t.Fatalf("move into slot5 should succeed at level 5 (cap=6)")
	}
	if sp.hotbars[0][5].Kind != "thorns" {
		t.Fatalf("hotbar[0][5] = %+v, want thorns", sp.hotbars[0])
	}

	// Уровень 0-4 => слот 5 должен быть отклонён.
	sp2 := &statelessProfile{inv: []InvStack{
		{PetalDef: PetalDef{Kind: "rock", Rarity: 0, HP: 30, MaxHP: 30, DMG: 3, Size: 14, Name: "rock"}, Count: 1},
	}, hotbars: [2][]PetalDef{{}, {}}, activeBar: 0, nick: "x", level: 0}
	okMove2 := sp2.move(map[string]interface{}{"src": "inv", "kind": "rock", "rarity": float64(0)}, 0, 5)
	if okMove2 {
		t.Fatalf("move into slot5 should be rejected at level 0 (cap=5)")
	}
}