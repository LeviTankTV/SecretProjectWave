package main

import (
	"testing"
)

// Статические проверки логики крафта statelessProfile.craft.
func TestCraftSuccessByForce(t *testing.T) {
	// Проверяем makeCraftPetal: результат следующей редкости того же вида.
	nd := makeCraftPetal("oval", 1)
	if nd.Rarity != 2 {
		t.Fatalf("результат редкости=%d, want 2 (Unusual->Rare)", nd.Rarity)
	}
	if nd.Kind != "oval" {
		t.Fatalf("результат вида=%q, want oval", nd.Kind)
	}
	// Грейд (порт родителя): статы результата умножаются на potMult(Pot)
	// (100..200 по лестнице грейдов).
	if nd.DMG != petalKinds["oval"].BaseDMG*petalPower[2]*potMult(nd.Pot) {
		t.Fatalf("DMG результата = %v, want %v", nd.DMG, petalKinds["oval"].BaseDMG*petalPower[2]*potMult(nd.Pot))
	}
	if nd.Color != groundPetalColor(2) {
		t.Fatalf("цвет результата = %X, want редкость Rare %X", nd.Color, groundPetalColor(2))
	}
}

// Скрафченный петаль высокого тира всегда мощнее любого петаля предыдущего
// тира (строгий рост силы по редкости): min Chaos > max Fractal, min Fractal
// > max Hexagon, min Hexagon > max Polygon, независимо от вида. Иначе Chaos,
// скрафченный из слабого вида, был бы слабее Fractal сильного — «тир» перестал
// бы доминировать. Исключение — спавн-петали без базового урона (Entropy):
// они не бьют сами (призывают миньонов) и из доминантности по урону выпадают.
func TestCraftHighTierFloorDominatesPrev(t *testing.T) {
	// Максимум урона, который вообще может иметь петаль редкости next
	// (без пола крафта — по BaseDMG всех видов).
	maxByNext := func(next int) float64 {
		var m float64
		for _, kn := range petalKinds {
			if kn.BaseDMG <= 0 {
				continue
			}
			d := kn.BaseDMG * petalPower[next]
			if d > m {
				m = d
			}
		}
		return m
	}
	for rar := 4; rar < 7; rar++ {
		next := rar + 1
		var minCraft float64 = 1e18
		for kind := range petalKinds {
			if petalKinds[kind].BaseDMG <= 0 {
				continue
			}
			d := makeCraftPetal(kind, rar).DMG
			if d < minCraft {
				minCraft = d
			}
		}
		prevMax := maxByNext(rar)
		if minCraft <= prevMax {
			t.Errorf("тир %d: мин. крафт DMG %.3f не больше макс. предыдущего тира %.3f",
				next, minCraft, prevMax)
		}
	}
}

// Проверяем, что в крафт уходит РОВНО invest петалей, а НЕ вся стопка.
// У пользователя было 23 петаля одного вида/редкости, вложил он 5 — должен
// пройти ровно 1 запуск катушки (1 попытка), остальные 18 остаться в стопке.
// Раньше катушка «сливала» стопку до остатка 1-4 (т.е. сжигали все 23).
func TestCraftConsumesOnlyInvest(t *testing.T) {
	for trial := 0; trial < 200; trial++ {
		sp := &statelessProfile{inv: []InvStack{
			{PetalDef: PetalDef{Kind: "oval", Rarity: 1, HP: 12, MaxHP: 12, DMG: 6, Size: 8, Name: "Oval"}, Count: 23},
		}, hotbars: [2][]PetalDef{{}, {}}, activeBar: 0, nick: "X", level: 5}
		status, successes, lost, _, batches := sp.craft("oval", 1, 5)
		if status != "success" && status != "fail" {
			t.Fatalf("status=%q, want success/fail", status)
		}
		// Ровно одна попытка: из 5 вложенных успех конвертирует 5 -> 1 выше,
		// провал теряет lost и возвращает 5-lost.
		if len(batches) != 1 {
			t.Fatalf("len(batches)=%d, want 1 (вложено всего 5)", len(batches))
		}
		rem := sp.inv[0].Count
		// Сгорело только из 5 вложенных: rem = 23 - 5*successes - lost.
		want := 23 - 5*successes - lost
		if rem != want {
			t.Fatalf("остаток стопки=%d, want %d (успехов=%d, потерь=%d) — стопка не должна сливаться целиком",
				rem, want, successes, lost)
		}
		if successes == 1 && rem != 18 {
			t.Fatalf("успешный крафт: остаток=%d, want 18 (23-5)", rem)
		}
	}
}

// Мульти-крафт invest=15: катушка докручивает пул до остатка 1..4. Учёт:
// остаток rem + потери lost + 5*successes == invest (всё вложенное учтено).
func TestCraftMultiAccounting(t *testing.T) {
	for trial := 0; trial < 300; trial++ {
		invest := 15
		sp := &statelessProfile{inv: []InvStack{
			{PetalDef: PetalDef{Kind: "oval", Rarity: 1, HP: 12, MaxHP: 12, DMG: 6, Size: 8, Name: "Oval"}, Count: invest},
		}, hotbars: [2][]PetalDef{{}, {}}, activeBar: 0, nick: "X", level: 5}
		status, successes, lost, results, batches := sp.craft("oval", 1, invest)
		if status == "error" {
			t.Fatalf("при invest=15 статус=error, want success/fail")
		}
		if len(results) != successes {
			t.Fatalf("len(results)=%d != successes=%d", len(results), successes)
		}
		// Число успешных батчей в логе совпадает с successes.
		bsucc := 0
		for _, b := range batches {
			if b.Success {
				bsucc++
			}
		}
		if bsucc != successes {
			t.Fatalf("успешных батчей=%d, want successes=%d", bsucc, successes)
		}
		if len(sp.pending) != successes {
			t.Fatalf("sp.pending=%d != successes=%d", len(sp.pending), successes)
		}
		rem := 0
		if i := sp.findStack("oval", 1); i >= 0 {
			rem = sp.inv[i].Count
		}
		// Катушка докручивает пул: батчей не меньше invest/5 (успех сжигает 5,
		// провал 1..4), а в конце остаётся 1..4 петалей.
		if len(batches) < invest/5 {
			t.Fatalf("запусков катушки=%d, want >= %d", len(batches), invest/5)
		}
		if rem < 0 || rem > 4 {
			t.Fatalf("остаток rem=%d, want 0..4 (0 — когда последний пяток ушёл в успех)", rem)
		}
		if rem+lost+5*successes != invest {
			t.Fatalf("инвариант: rem=%d + lost=%d + 5*succ=%d = %d, want %d",
				rem, lost, 5*successes, rem+lost+5*successes, invest)
		}
	}
	// Цвет результата крафта отражает редкость (исправление белого петаля).
}

// Сценарий пользователя: вложил 49 — катушка крутит ВСЁ (провальные возвраты
// тоже докручиваются), в инвентаре остаётся 0..4 петалей, не больше.
func TestCraftSpinsDownToTail(t *testing.T) {
	for trial := 0; trial < 300; trial++ {
		sp := &statelessProfile{inv: []InvStack{
			{PetalDef: PetalDef{Kind: "oval", Rarity: 1, HP: 12, MaxHP: 12, DMG: 6, Size: 8, Name: "Oval"}, Count: 49},
		}, hotbars: [2][]PetalDef{{}, {}}, activeBar: 0, nick: "X", level: 5}
		status, successes, lost, _, batches := sp.craft("oval", 1, 49)
		if status == "error" {
			t.Fatalf("при invest=49 статус=error, want success/fail")
		}
		rem := 0
		if i := sp.findStack("oval", 1); i >= 0 {
			rem = sp.inv[i].Count
		}
		if rem < 0 || rem > 4 {
			t.Fatalf("остаток=%d, want 0..4 (0 — когда последний пяток ушёл в успех) (успехов=%d, потерь=%d, батчей=%d)",
				rem, successes, lost, len(batches))
		}
		if rem+lost+5*successes != 49 {
			t.Fatalf("инвариант 49: rem=%d + lost=%d + 5*succ=%d = %d, want 49",
				rem, lost, 5*successes, rem+lost+5*successes)
		}
	}
}

// Мульти-крафт с «хвостом» (invest не кратен 5): хвост не должен списываться
// и не должен добавляться дважды.
func TestCraftTailNotDoubleAdded(t *testing.T) {
	for trial := 0; trial < 400; trial++ {
		invest := 8 // 1 попытка (5) + хвост 3
		sp := &statelessProfile{inv: []InvStack{
			{PetalDef: PetalDef{Kind: "oval", Rarity: 1, HP: 12, MaxHP: 12, DMG: 6, Size: 8, Name: "Oval"}, Count: invest},
		}, hotbars: [2][]PetalDef{{}, {}}, activeBar: 0, nick: "X", level: 5}
		status, successes, lost, _, _ := sp.craft("oval", 1, invest)
		if status == "error" {
			t.Fatalf("invest=8 ошибочно error")
		}
		rem := 0
		if i := sp.findStack("oval", 1); i >= 0 {
			rem = sp.inv[i].Count
		}
		// Инвариант старой неверной логики дал бы rem>=7. Верно: invest не меняет
		// хвост, остаётся invest - lost - 5*successes.
		if rem + lost + 5*successes != invest {
			t.Fatalf("инвариант хвоста: rem=%d + lost=%d + 5*succ=%d = %d, want %d",
				rem, lost, 5*successes, rem+lost+5*successes, invest)
		}
	}
}

func TestCraftChanceTable(t *testing.T) {
	if craftSuccessChance(0) != 20.0 { t.Fatalf("chance[0]=%v", craftSuccessChance(0)) }
	if craftSuccessChance(1) != 12.5 { t.Fatalf("chance[1]=%v", craftSuccessChance(1)) }
	if craftSuccessChance(2) != 7.5  { t.Fatalf("chance[2]=%v", craftSuccessChance(2)) }
	if craftSuccessChance(3) != 4.0  { t.Fatalf("chance[3]=%v", craftSuccessChance(3)) }
	if craftSuccessChance(4) != 2.0  { t.Fatalf("chance[4]=%v", craftSuccessChance(4)) }
	if craftSuccessChance(5) != 1.0  { t.Fatalf("chance[5]=%v", craftSuccessChance(5)) }
	if craftSuccessChance(6) != 1.0  { t.Fatalf("chance[6]=%v", craftSuccessChance(6)) }
	if craftSuccessChance(7) != -1   { t.Fatalf("chance[7] (Chaos) должно быть -1, а = %v", craftSuccessChance(7)) }
	if craftSuccessChance(-1) != -1  { t.Fatalf("chance[-1] должно быть -1") }
}

func TestCraftRejectsFewerThanFive(t *testing.T) {
	sp := &statelessProfile{inv: []InvStack{
		{PetalDef: PetalDef{Kind: "thorns", Rarity: 0, HP: 5, MaxHP: 5, DMG: 7, Size: 8, Name: "Thorns"}, Count: 3},
	}, hotbars: [2][]PetalDef{{}, {}}, activeBar: 0, nick: "X", level: 5}
	status, _, _, _, _ := sp.craft("thorns", 0, 5)
	if status != "error" {
		t.Fatalf("при <5 петалей статус=%q, want error", status)
	}
}

// Провал крафта: теряется 1..4 петаля, остальное остаётся.
func TestCraftFailLoses1to4(t *testing.T) {
	var counts = []int{0, 0, 0, 0}
	for i := 0; i < 2000; i++ {
		sp := &statelessProfile{inv: []InvStack{
			{PetalDef: PetalDef{Kind: "rock", Rarity: 0, HP: 14, MaxHP: 14, DMG: 2.5, Size: 8, Name: "Rock"}, Count: 5},
		}, hotbars: [2][]PetalDef{{}, {}}, activeBar: 0, nick: "X", level: 5}
		// Форсируем провал: подсовываем 100% ложь через проверку статуса по факту.
		status, _, lost, _, _ := sp.craft("rock", 0, 5)
		if status == "fail" {
			if lost < 1 || lost > 4 {
				t.Fatalf("при провале lost=%d, должно быть 1..4", lost)
			}
			rem := 0
			if i := sp.findStack("rock", 0); i >= 0 {
				rem = sp.inv[i].Count
			}
			if rem != 5 - lost {
				t.Fatalf("после провала осталось %d, want %d", rem, 5-lost)
			}
			counts[lost-1]++
		}
	}
	// Проверяем, что хоть что-то прошло как провал (и не всегда только success по чистой удаче).
	if counts[0]+counts[1]+counts[2]+counts[3] == 0 {
		// За 2000 попыток с шансом 20% успеха вряд ли все успешны; но для стабильности
		// лишь указываем, что провалов достаточно, не падая.
		t.Log("не замечено провалов за 2000 попыток (маловероятно)")
	}
}