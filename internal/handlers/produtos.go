package handlers

import (
	"errors"
	"net/http"
	"strings"

	"cupcake_store/backend/internal/db"
	"cupcake_store/backend/internal/httpx"
)

func (h *Handler) listProducts(w http.ResponseWriter, r *http.Request) {
	busca := strings.TrimSpace(r.URL.Query().Get("busca"))
	products, err := h.store.ListProducts(r.Context(), busca)
	if err != nil {
		httpx.Internal(w, err, "Não foi possível listar os produtos")
		return
	}
	httpx.JSON(w, http.StatusOK, products)
}

func (h *Handler) getProduct(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	p, err := h.store.ProductByID(r.Context(), id)
	switch {
	case errors.Is(err, db.ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "Produto não encontrado")
	case err != nil:
		httpx.Internal(w, err, "Não foi possível buscar o produto")
	default:
		httpx.JSON(w, http.StatusOK, p)
	}
}
