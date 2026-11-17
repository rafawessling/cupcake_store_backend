package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"cupcake_store/backend/internal/auth"
	"cupcake_store/backend/internal/db"
	"cupcake_store/backend/internal/httpx"
	"cupcake_store/backend/internal/models"
)

const (
	maxQuantity = 99
	maxItems    = 100
	errOrder    = "Não foi possível concluir o pedido"
)

type orderRequest struct {
	Itens []models.ItemInput `json:"itens"`
}

func (o orderRequest) validate() string {
	if len(o.Itens) == 0 {
		return "O pedido precisa ter ao menos um item"
	}
	if len(o.Itens) > maxItems {
		return "Pedido com itens demais"
	}
	seen := make(map[int]bool, len(o.Itens))
	for _, it := range o.Itens {
		switch {
		case it.ProdutoID <= 0:
			return "Produto inválido"
		case it.Quantidade < 1 || it.Quantidade > maxQuantity:
			return fmt.Sprintf("Quantidade deve estar entre 1 e %d", maxQuantity)
		case seen[it.ProdutoID]:
			return "Produto repetido no pedido"
		}
		seen[it.ProdutoID] = true
	}
	return ""
}

func (h *Handler) createOrder(w http.ResponseWriter, r *http.Request) {
	var req orderRequest
	if !httpx.Decode(w, r, &req) {
		return
	}
	if msg := req.validate(); msg != "" {
		httpx.Error(w, http.StatusBadRequest, msg)
		return
	}
	order, err := h.store.CreateOrder(r.Context(), auth.UserID(r.Context()), req.Itens)
	var unavailable db.ProductUnavailableError
	switch {
	case errors.As(err, &unavailable):
		httpx.Error(w, http.StatusBadRequest,
			fmt.Sprintf("Produto %d não existe ou está indisponível", unavailable.ProdutoID))
	case err != nil:
		httpx.Internal(w, err, errOrder)
	default:
		httpx.JSON(w, http.StatusCreated, order)
	}
}

func (h *Handler) getOrder(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	order, err := h.store.OrderByID(r.Context(), id, auth.UserID(r.Context()))
	switch {
	case errors.Is(err, db.ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "Pedido não encontrado")
	case err != nil:
		httpx.Internal(w, err, "Não foi possível buscar o pedido")
	default:
		httpx.JSON(w, http.StatusOK, order)
	}
}
