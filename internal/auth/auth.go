package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"cupcake_store/backend/internal/httpx"
)

const TokenTTL = 24 * time.Hour

var ErrInvalidToken = errors.New("token inválido")

// Cabeçalho fixo: qualquer outro "alg" é rejeitado.
var header = base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))

func HashPassword(senha string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(senha), bcrypt.DefaultCost)
	return string(b), err
}

func CheckPassword(hash, senha string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(senha)) == nil
}

type Tokens struct {
	secret []byte
	now    func() time.Time
}

func NewTokens(secret string) *Tokens {
	return &Tokens{secret: []byte(secret), now: time.Now}
}

type claims struct {
	Sub string `json:"sub"`
	Iat int64  `json:"iat"`
	Exp int64  `json:"exp"`
}

func (t *Tokens) sign(data string) string {
	mac := hmac.New(sha256.New, t.secret)
	mac.Write([]byte(data))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (t *Tokens) Issue(userID int) (string, error) {
	now := t.now()
	payload, err := json.Marshal(claims{
		Sub: strconv.Itoa(userID),
		Iat: now.Unix(),
		Exp: now.Add(TokenTTL).Unix(),
	})
	if err != nil {
		return "", err
	}
	data := header + "." + base64.RawURLEncoding.EncodeToString(payload)
	return data + "." + t.sign(data), nil
}

func (t *Tokens) Parse(token string) (int, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] != header {
		return 0, ErrInvalidToken
	}
	want := t.sign(parts[0] + "." + parts[1])
	if !hmac.Equal([]byte(want), []byte(parts[2])) {
		return 0, ErrInvalidToken
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return 0, ErrInvalidToken
	}
	var c claims
	if err := json.Unmarshal(raw, &c); err != nil || t.now().Unix() >= c.Exp {
		return 0, ErrInvalidToken
	}
	id, err := strconv.Atoi(c.Sub)
	if err != nil {
		return 0, ErrInvalidToken
	}
	return id, nil
}

type ctxKey struct{}

// UserID devolve o usuário autenticado colocado no contexto por Require.
func UserID(ctx context.Context) int {
	id, _ := ctx.Value(ctxKey{}).(int)
	return id
}

// Require exige "Authorization: Bearer <token>" válido.
func (t *Tokens) Require(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		scheme, token, ok := strings.Cut(r.Header.Get("Authorization"), " ")
		if !ok || !strings.EqualFold(scheme, "Bearer") {
			httpx.Error(w, http.StatusUnauthorized, "Token de acesso ausente ou inválido")
			return
		}
		id, err := t.Parse(strings.TrimSpace(token))
		if err != nil {
			httpx.Error(w, http.StatusUnauthorized, "Token de acesso ausente ou inválido")
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, id)))
	}
}
