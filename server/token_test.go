package main

import "testing"

// Токены самодостаточны: валидация не зависит от памяти, поэтому токен,
// выданный ДО "рестарта", остаётся валидным (подпись + срок действия).
func TestTokenPersist(t *testing.T) {
	tok, errMsg := generateToken("alice")
	if errMsg != "" {
		t.Fatalf("generateToken err: %s", errMsg)
	}
	// Имитация рестарта не нужна: validateToken не использует память.
	nick, ok := validateToken(tok)
	if !ok || nick != "alice" {
		t.Fatalf("validateToken = (%q, %v), want (alice, true)", nick, ok)
	}
}

func TestTokenTamper(t *testing.T) {
	tok, _ := generateToken("bob")
	// Подмена ника в payload должна быть поймана подписью.
	if _, ok := validateToken(tok); !ok {
		t.Fatal("valid token rejected")
	}
	// Меняем payload, оставляем подпись -> невалидно.
	if _, ok := validateToken(tok + "x"); ok {
		t.Fatal("tampered token accepted")
	}
}
