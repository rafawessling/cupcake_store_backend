package config

import (
	"errors"
	"os"
	"strings"
)

const (
	minSecretLen      = 32
	secretPlaceholder = "troque-este-valor"
	defaultCORSOrigin = "http://localhost:5173"
)

type Config struct {
	DatabaseURL string
	JWTSecret   string
	Port        string
	CORSOrigins []string
}

func Load() (Config, error) {
	c := Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		Port:        os.Getenv("PORT"),
		CORSOrigins: parseOrigins(os.Getenv("CORS_ORIGIN")),
	}
	if c.Port == "" {
		c.Port = "8080"
	}
	if len(c.CORSOrigins) == 0 {
		c.CORSOrigins = []string{defaultCORSOrigin}
	}
	if c.DatabaseURL == "" {
		return c, errors.New("DATABASE_URL não definida")
	}
	if len(c.JWTSecret) < minSecretLen {
		return c, errors.New("JWT_SECRET deve ter ao menos 32 caracteres (gere com: openssl rand -base64 48)")
	}
	if strings.Contains(c.JWTSecret, secretPlaceholder) {
		return c, errors.New("JWT_SECRET ainda é o valor de exemplo; gere um novo com: openssl rand -base64 48")
	}
	return c, nil
}

// parseOrigins aceita uma lista separada por vírgula e remove espaços e a barra final.
func parseOrigins(raw string) []string {
	var origins []string
	for _, o := range strings.Split(raw, ",") {
		if o = strings.TrimRight(strings.TrimSpace(o), "/"); o != "" {
			origins = append(origins, o)
		}
	}
	return origins
}
