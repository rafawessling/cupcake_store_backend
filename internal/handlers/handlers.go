package handlers

import (
	"net/http"
	"strconv"

	"cupcake_store/backend/internal/auth"
	"cupcake_store/backend/internal/db"
	"cupcake_store/backend/internal/httpx"
)

type Handler struct {
	store  db.Store
	tokens *auth.Tokens
}

func New(store db.Store, tokens *auth.Tokens) *Handler {
	return &Handler{store: store, tokens: tokens}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/saude", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("POST /api/usuarios", h.register)
	mux.HandleFunc("POST /api/login", h.login)
	mux.HandleFunc("GET /api/produtos", h.listProducts)
	mux.HandleFunc("GET /api/produtos/{id}", h.getProduct)
	mux.HandleFunc("POST /api/pedidos", h.tokens.Require(h.createOrder))
	mux.HandleFunc("GET /api/pedidos/{id}", h.tokens.Require(h.getOrder))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		httpx.Error(w, http.StatusNotFound, "Rota não encontrada")
	})
	return mux
}

// pathID lê {id} da URL e responde 400 se não for um inteiro positivo.
func pathID(w http.ResponseWriter, r *http.Request) (int, bool) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		httpx.Error(w, http.StatusBadRequest, "ID inválido")
		return 0, false
	}
	return id, true
}
