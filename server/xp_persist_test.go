package main

import (
	"math/rand"
	"strconv"
	"testing"
)

// Тест сохранения/загрузки уровня через реально зарегистрированного юзера.
func TestPersistProgress(t *testing.T) {
	initDB()
	nick := "persist_test_" + strconv.Itoa(rand.Intn(100000))
	if tok, err := registerUser(nick, "pass1234"); err != "" || tok == "" {
		t.Fatalf("registerUser failed: %q", err)
	}
	saveProgress(nick, 7, 123.5)
	lvl, xp := loadProgress(nick)
	if lvl != 7 {
		t.Errorf("level=%d, want 7", lvl)
	}
	if xp != 123.5 {
		t.Errorf("xp=%v, want 123.5", xp)
	}
	if tok, err := loginUser(nick, "pass1234"); err != "" || tok == "" {
		t.Errorf("login after save failed: %q — пароль испорчен!", err)
	}
}

// Тест полного профиля: запас + раскладка хотбаров сохраняются и загружаются.
func TestPersistFullProfile(t *testing.T) {
	initDB()
	nick := "persistprof_test_" + strconv.Itoa(rand.Intn(100000))
	if tok, err := registerUser(nick, "pass1234"); err != "" || tok == "" {
		t.Fatalf("registerUser failed: %q", err)
	}

	inv := []InvStack{
		{PetalDef: PetalDef{ID: "", Kind: "thorns", Rarity: 3, HP: 9, MaxHP: 9, DMG: 7, Size: 5, Color: 100, Name: "thorns"}, Count: 4},
		{PetalDef: PetalDef{ID: "", Kind: "rock", Rarity: 1, HP: 30, MaxHP: 30, DMG: 3, Size: 14, Color: 200, Name: "rock"}, Count: 2},
	}
	hot0 := []PetalDef{
		{ID: "h0", Kind: "rock", Rarity: 2, HP: 30, MaxHP: 30, DMG: 3, Size: 14, Color: 200, Name: "rock"},
		{ID: "h1", Kind: "oval", Rarity: 0, HP: 5, MaxHP: 5, DMG: 3, Size: 8, Color: 255, Name: "oval"},
	}
	hot1 := []PetalDef{{ID: "h2", Kind: "thorns", Rarity: 4, HP: 9, MaxHP: 9, DMG: 7, Size: 5, Color: 100, Name: "thorns"}}
	hotbars := [2][]PetalDef{hot0, hot1}

	saveProfile(nick, 12, 55.5, inv, hotbars, 1)

	lvl, xp, inv2, hotbars2, activeBar := loadProfile(nick)
	if lvl != 12 || xp != 55.5 {
		t.Errorf("level/xp = %d/%v, want 12/55.5", lvl, xp)
	}
	if activeBar != 1 {
		t.Errorf("activeBar=%d, want 1", activeBar)
	}
	if len(inv2) != 2 {
		t.Fatalf("len(inv)=%d, want 2", len(inv2))
	}
	if inv2[0].Kind != "thorns" || inv2[0].Count != 4 {
		t.Errorf("inv[0] = %+v, want thorns×4", inv2[0])
	}
	if inv2[1].Kind != "rock" || inv2[1].Count != 2 {
		t.Errorf("inv[1] = %+v, want rock×2", inv2[1])
	}
	if len(hotbars2[0]) != 2 || hotbars2[0][0].Kind != "rock" || hotbars2[0][1].Kind != "oval" {
		t.Errorf("hotbars[0] = %+v, want [rock, oval]", hotbars2[0])
	}
	if len(hotbars2[1]) != 1 || hotbars2[1][0].Kind != "thorns" || hotbars2[1][0].Rarity != 4 {
		t.Errorf("hotbars[1] = %+v, want [thorns r4]", hotbars2[1])
	}
	if tok, err := loginUser(nick, "pass1234"); err != "" || tok == "" {
		t.Errorf("login after saveProfile failed — пароль испорчен: %q", err)
	}
}
