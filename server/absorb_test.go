package main

import "testing"

// petalDefTest собирает PetalDef как это делает серверный инвентарь:
// реальные статы = Base* × множитель редкости (rarityDanger).
func petalDefTest(kind string, rarity int) PetalDef {
	kn := petalKinds[kind]
	mult := petalPower[rarity]
	return PetalDef{
		Kind:   kind,
		Rarity: rarity,
		DMG:    kn.BaseDMG * mult,
		MaxHP:  kn.BaseHP * mult,
		Name:   kn.Name,
	}
}

// TestPetalStackXPAllRarities — все 8 редкостей должны давать XP за 1 штуку,
// и XP обязан строго расти с редкостью (для любого вида петеля).
// Прогрессия должна быть геометрической: XP(r+1)/XP(r) ~ absorbXPRatio (2.0).
func TestPetalStackXPAllRarities(t *testing.T) {
	for kind := range petalKinds {
		prev := 0.0
		for rar := 0; rar < 8; rar++ {
			xp := petalStackXP(petalDefTest(kind, rar))
			if xp <= 0 {
				t.Errorf("petalStackXP(%s,%d)=%v, want > 0", kind, rar, xp)
			}
			if xp <= prev {
				t.Errorf("petalStackXP(%s,%d)=%v не больше предыдущего %v", kind, rar, xp, prev)
			}
			if rar > 0 {
				ratio := xp / prev
				// Полутора процентов допуска на ошибки округления double.
				if ratio < absorbXPRatio*0.985 || ratio > absorbXPRatio*1.015 {
					t.Errorf("petalStackXP(%s,%d)=%v / предыдущая = %.3f, want геометрия с ratio %.1f",
						kind, rar, xp, ratio, absorbXPRatio)
				}
			}
			prev = xp
		}
	}
}

// TestAbsorbGrantsXP — абсорб списывает петали из инвентаря и начисляет
// ровно petalStackXP*count за реально списанные штуки.
func TestAbsorbGrantsXP(t *testing.T) {
	sp := &statelessProfile{}
	sp.addCountToStack("rock", 1, 4)
	sp.addCountToStack("rock", 1, 2)

	got := sp.absorb([]absorbItem{{Kind: "rock", Rarity: 1, Count: 10}})
	want := petalStackXP(petalDefTest("rock", 1)) * 6
	if got != want {
		t.Errorf("absorb got %v, want %v (должны списаться только 6 реальных)", got, want)
	}
	if i := sp.findStack("rock", 1); i >= 0 {
		t.Errorf("стак rock/1 не должен остаться после абсорба")
	}
}
