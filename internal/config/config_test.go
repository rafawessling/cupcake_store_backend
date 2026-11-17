package config

import (
	"strings"
	"testing"
)

func setEnv(t *testing.T, secret, cors string) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("JWT_SECRET", secret)
	t.Setenv("CORS_ORIGIN", cors)
	t.Setenv("PORT", "")
}

func TestLoadSecretRules(t *testing.T) {
	cases := map[string]struct {
		secret string
		ok     bool
	}{
		"curto":       {strings.Repeat("a", 31), false},
		"mínimo":      {strings.Repeat("a", 32), true},
		"placeholder": {"troque-este-valor-gerado-via-openssl-rand-base64-48", false},
		"gerado":      {"k3Jx9mQ2vL8pR5tY7uW1zN4bC6dF0gHjA3sE5iO8lM2nP4qR", true},
	}
	for name, c := range cases {
		setEnv(t, c.secret, "")
		if _, err := Load(); (err == nil) != c.ok {
			t.Errorf("%s: err=%v, esperado ok=%v", name, err, c.ok)
		}
	}
}

func TestLoadCORSOrigins(t *testing.T) {
	secret := strings.Repeat("a", 32)
	cases := map[string][]string{
		"":                         {"http://localhost:5173"},
		"https://loja.vercel.app/": {"https://loja.vercel.app"},
		" https://a.app/ , http://localhost:5173,": {"https://a.app", "http://localhost:5173"},
	}
	for raw, want := range cases {
		setEnv(t, secret, raw)
		c, err := Load()
		if err != nil || strings.Join(c.CORSOrigins, "|") != strings.Join(want, "|") {
			t.Errorf("%q: %v err=%v, esperado %v", raw, c.CORSOrigins, err, want)
		}
	}
	setEnv(t, secret, "")
	if c, _ := Load(); c.Port != "8080" {
		t.Errorf("porta padrão: %s", c.Port)
	}
}
