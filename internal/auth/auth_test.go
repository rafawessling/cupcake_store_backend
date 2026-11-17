package auth

import (
	"testing"
	"time"
)

func TestTokenRoundTrip(t *testing.T) {
	tk := NewTokens("segredo-de-teste-123")
	tok, err := tk.Issue(42)
	if err != nil {
		t.Fatal(err)
	}
	if id, err := tk.Parse(tok); err != nil || id != 42 {
		t.Fatalf("id=%d err=%v", id, err)
	}
}

func TestTokenRejected(t *testing.T) {
	tk := NewTokens("segredo-de-teste-123")
	tok, _ := tk.Issue(1)

	other := NewTokens("outro-segredo-123456")
	if _, err := other.Parse(tok); err == nil {
		t.Error("aceitou token de outro segredo")
	}
	if _, err := tk.Parse(tok + "x"); err == nil {
		t.Error("aceitou assinatura adulterada")
	}
	if _, err := tk.Parse("lixo"); err == nil {
		t.Error("aceitou token malformado")
	}

	tk.now = func() time.Time { return time.Now().Add(TokenTTL + time.Minute) }
	if _, err := tk.Parse(tok); err == nil {
		t.Error("aceitou token expirado")
	}
}

func TestPassword(t *testing.T) {
	h, err := HashPassword("abc123")
	if err != nil || h == "abc123" {
		t.Fatalf("hash inválido: %v", err)
	}
	if !CheckPassword(h, "abc123") || CheckPassword(h, "errada") {
		t.Error("verificação de senha incorreta")
	}
}
