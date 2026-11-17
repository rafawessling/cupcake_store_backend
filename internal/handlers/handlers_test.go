package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"cupcake_store/backend/internal/auth"
	"cupcake_store/backend/internal/db"
	"cupcake_store/backend/internal/models"
)

type fakeStore struct {
	users    map[string]fakeUser
	products map[int]models.Product
	orders   map[int]orderOwner
}

type fakeUser struct {
	models.User
	hash string
}

type orderOwner struct {
	models.Order
	userID int
}

func (f *fakeStore) CreateUser(_ context.Context, nome, email, hash string) (models.User, error) {
	if _, ok := f.users[email]; ok {
		return models.User{}, db.ErrEmailTaken
	}
	u := models.User{ID: len(f.users) + 1, Nome: nome, Email: email}
	f.users[email] = fakeUser{u, hash}
	return u, nil
}

func (f *fakeStore) UserByEmail(_ context.Context, email string) (models.User, string, error) {
	u, ok := f.users[email]
	if !ok {
		return models.User{}, "", db.ErrNotFound
	}
	return u.User, u.hash, nil
}

func (f *fakeStore) ListProducts(context.Context, string) ([]models.Product, error) {
	return []models.Product{f.products[1]}, nil
}

func (f *fakeStore) ProductByID(_ context.Context, id int) (models.Product, error) {
	p, ok := f.products[id]
	if !ok {
		return p, db.ErrNotFound
	}
	return p, nil
}

func (f *fakeStore) CreateOrder(_ context.Context, userID int, itens []models.ItemInput) (models.Order, error) {
	o := models.Order{ID: len(f.orders) + 1, Status: "confirmado"}
	for _, it := range itens {
		p, ok := f.products[it.ProdutoID]
		if !ok {
			return o, db.ProductUnavailableError{ProdutoID: it.ProdutoID}
		}
		sub := p.Preco * models.Money(it.Quantidade)
		o.ValorTotal += sub
		o.Itens = append(o.Itens, models.OrderItem{ProdutoID: p.ID, Nome: p.Nome, Quantidade: it.Quantidade, PrecoUnitario: p.Preco, Subtotal: sub})
	}
	f.orders[o.ID] = orderOwner{o, userID}
	return o, nil
}

func (f *fakeStore) OrderByID(_ context.Context, id, userID int) (models.Order, error) {
	o, ok := f.orders[id]
	if !ok || o.userID != userID {
		return models.Order{}, db.ErrNotFound
	}
	return o.Order, nil
}

func newServer() (http.Handler, *auth.Tokens) {
	store := &fakeStore{
		users:    map[string]fakeUser{},
		products: map[int]models.Product{1: {ID: 1, Nome: "Beijo de Berries", Preco: 1350}},
		orders:   map[int]orderOwner{},
	}
	tk := auth.NewTokens("segredo-de-teste-123")
	return New(store, tk).Routes(), tk
}

func do(h http.Handler, method, path, token, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestRegisterValidation(t *testing.T) {
	h, _ := newServer()
	cases := map[string]string{
		"nome vazio":     `{"nome":"","email":"a@b.com","senha":"123456","confirmacao_senha":"123456"}`,
		"email inválido": `{"nome":"Ana","email":"abc","senha":"123456","confirmacao_senha":"123456"}`,
		"senha curta":    `{"nome":"Ana","email":"a@b.com","senha":"123","confirmacao_senha":"123"}`,
		"confirmação":    `{"nome":"Ana","email":"a@b.com","senha":"123456","confirmacao_senha":"654321"}`,
		"json inválido":  `{`,
	}
	for name, body := range cases {
		if rec := do(h, "POST", "/api/usuarios", "", body); rec.Code != 400 {
			t.Errorf("%s: status %d", name, rec.Code)
		}
	}
}

func TestRegisterLoginFlow(t *testing.T) {
	h, _ := newServer()
	body := `{"nome":"Ana","email":"Ana@Mail.com","senha":"123456","confirmacao_senha":"123456"}`
	rec := do(h, "POST", "/api/usuarios", "", body)
	if rec.Code != 201 || strings.Contains(rec.Body.String(), "senha") {
		t.Fatalf("cadastro: %d %s", rec.Code, rec.Body)
	}
	if rec := do(h, "POST", "/api/usuarios", "", strings.Replace(body, "Ana@Mail.com", "ana@mail.COM", 1)); rec.Code != 409 {
		t.Errorf("duplicado: %d", rec.Code)
	}

	wrongPass := do(h, "POST", "/api/login", "", `{"email":"ana@mail.com","senha":"errada"}`)
	noUser := do(h, "POST", "/api/login", "", `{"email":"x@mail.com","senha":"123456"}`)
	if wrongPass.Code != 401 || wrongPass.Body.String() != noUser.Body.String() {
		t.Errorf("login inválido deve ser idêntico: %d %s / %d %s", wrongPass.Code, wrongPass.Body, noUser.Code, noUser.Body)
	}

	rec = do(h, "POST", "/api/login", "", `{"email":"ANA@mail.com","senha":"123456"}`)
	var res struct {
		Token   string
		Usuario models.User
	}
	json.Unmarshal(rec.Body.Bytes(), &res)
	if rec.Code != 200 || res.Token == "" || res.Usuario.Email != "ana@mail.com" {
		t.Fatalf("login: %d %s", rec.Code, rec.Body)
	}
}

func TestProducts(t *testing.T) {
	h, _ := newServer()
	if rec := do(h, "GET", "/api/produtos/1", "", ""); rec.Code != 200 || !strings.Contains(rec.Body.String(), `"preco":13.50`) {
		t.Errorf("detalhe: %d %s", rec.Code, rec.Body)
	}
	if rec := do(h, "GET", "/api/produtos/9", "", ""); rec.Code != 404 {
		t.Errorf("inexistente: %d", rec.Code)
	}
	if rec := do(h, "GET", "/api/produtos/abc", "", ""); rec.Code != 400 {
		t.Errorf("id inválido: %d", rec.Code)
	}
}

func TestOrders(t *testing.T) {
	h, tk := newServer()
	owner, _ := tk.Issue(1)
	other, _ := tk.Issue(2)

	if rec := do(h, "POST", "/api/pedidos", "", `{"itens":[{"produto_id":1,"quantidade":1}]}`); rec.Code != 401 {
		t.Errorf("sem token: %d", rec.Code)
	}
	if rec := do(h, "POST", "/api/pedidos", "lixo", `{"itens":[{"produto_id":1,"quantidade":1}]}`); rec.Code != 401 {
		t.Errorf("token inválido: %d", rec.Code)
	}

	invalid := []string{
		`{"itens":[]}`,
		`{"itens":[{"produto_id":1,"quantidade":0}]}`,
		`{"itens":[{"produto_id":1,"quantidade":100}]}`,
		`{"itens":[{"produto_id":1,"quantidade":1},{"produto_id":1,"quantidade":2}]}`,
		`{"itens":[{"produto_id":99,"quantidade":1}]}`,
	}
	for _, b := range invalid {
		if rec := do(h, "POST", "/api/pedidos", owner, b); rec.Code != 400 {
			t.Errorf("%s: %d", b, rec.Code)
		}
	}

	// valores enviados pelo cliente são ignorados
	rec := do(h, "POST", "/api/pedidos", owner, `{"itens":[{"produto_id":1,"quantidade":3,"preco_unitario":0.01}],"valor_total":1}`)
	if rec.Code != 201 || !strings.Contains(rec.Body.String(), `"valor_total":40.50`) {
		t.Fatalf("pedido: %d %s", rec.Code, rec.Body)
	}
	if rec := do(h, "GET", "/api/pedidos/1", owner, ""); rec.Code != 200 {
		t.Errorf("dono: %d", rec.Code)
	}
	if rec := do(h, "GET", "/api/pedidos/1", other, ""); rec.Code != 404 {
		t.Errorf("outro usuário: %d", rec.Code)
	}
	if rec := do(h, "GET", "/api/pedidos/77", owner, ""); rec.Code != 404 {
		t.Errorf("inexistente: %d", rec.Code)
	}
}

func TestHealth(t *testing.T) {
	h, _ := newServer()
	if rec := do(h, "GET", "/api/saude", "", ""); rec.Code != 200 || strings.TrimSpace(rec.Body.String()) != `{"status":"ok"}` {
		t.Errorf("saúde: %d %s", rec.Code, rec.Body)
	}
}
