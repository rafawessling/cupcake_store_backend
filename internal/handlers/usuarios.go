package handlers

import (
	"errors"
	"net/http"
	"net/mail"
	"strings"
	"unicode/utf8"

	"cupcake_store/backend/internal/auth"
	"cupcake_store/backend/internal/db"
	"cupcake_store/backend/internal/httpx"
	"cupcake_store/backend/internal/models"
)

const (
	minPassword = 6
	maxPassword = 72 // limite do bcrypt
)

type registerRequest struct {
	Nome             string `json:"nome"`
	Email            string `json:"email"`
	Senha            string `json:"senha"`
	ConfirmacaoSenha string `json:"confirmacao_senha"`
}

func (r *registerRequest) validate() string {
	r.Nome = strings.TrimSpace(r.Nome)
	r.Email = strings.ToLower(strings.TrimSpace(r.Email))
	switch {
	case r.Nome == "":
		return "Nome é obrigatório"
	case utf8.RuneCountInString(r.Nome) > 120:
		return "Nome deve ter no máximo 120 caracteres"
	case !validEmail(r.Email):
		return "E-mail inválido"
	case len(r.Senha) < minPassword:
		return "Senha deve ter ao menos 6 caracteres"
	case len(r.Senha) > maxPassword:
		return "Senha deve ter no máximo 72 caracteres"
	case r.Senha != r.ConfirmacaoSenha:
		return "As senhas não conferem"
	}
	return ""
}

func validEmail(s string) bool {
	if len(s) > 150 {
		return false
	}
	a, err := mail.ParseAddress(s)
	return err == nil && a.Address == s
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if !httpx.Decode(w, r, &req) {
		return
	}
	if msg := req.validate(); msg != "" {
		httpx.Error(w, http.StatusBadRequest, msg)
		return
	}
	hash, err := auth.HashPassword(req.Senha)
	if err != nil {
		httpx.Internal(w, err, "Não foi possível concluir o cadastro")
		return
	}
	user, err := h.store.CreateUser(r.Context(), req.Nome, req.Email, hash)
	switch {
	case errors.Is(err, db.ErrEmailTaken):
		httpx.Error(w, http.StatusConflict, "E-mail já cadastrado")
	case err != nil:
		httpx.Internal(w, err, "Não foi possível concluir o cadastro")
	default:
		httpx.JSON(w, http.StatusCreated, user)
	}
}

type loginRequest struct {
	Email string `json:"email"`
	Senha string `json:"senha"`
}

type loginResponse struct {
	Token   string      `json:"token"`
	Usuario models.User `json:"usuario"`
}

// Hash válido usado para igualar o tempo de resposta quando o e-mail não existe.
var dummyHash, _ = auth.HashPassword("cupcake-store-dummy")

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if !httpx.Decode(w, r, &req) {
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" || req.Senha == "" {
		httpx.Error(w, http.StatusBadRequest, "Informe e-mail e senha")
		return
	}

	user, hash, err := h.store.UserByEmail(r.Context(), email)
	if errors.Is(err, db.ErrNotFound) {
		auth.CheckPassword(dummyHash, req.Senha)
		httpx.Error(w, http.StatusUnauthorized, "E-mail ou senha inválidos")
		return
	}
	if err != nil {
		httpx.Internal(w, err, "Não foi possível fazer login")
		return
	}
	if !auth.CheckPassword(hash, req.Senha) {
		httpx.Error(w, http.StatusUnauthorized, "E-mail ou senha inválidos")
		return
	}
	token, err := h.tokens.Issue(user.ID)
	if err != nil {
		httpx.Internal(w, err, "Não foi possível fazer login")
		return
	}
	httpx.JSON(w, http.StatusOK, loginResponse{Token: token, Usuario: user})
}
