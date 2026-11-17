package models

import "fmt"

// Money guarda centavos para evitar erros de ponto flutuante e serializa como 13.50.
type Money int64

func (m Money) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%d.%02d", m/100, m%100)), nil
}

type User struct {
	ID    int    `json:"id"`
	Nome  string `json:"nome"`
	Email string `json:"email"`
}

type Product struct {
	ID        int    `json:"id"`
	Nome      string `json:"nome"`
	Descricao string `json:"descricao"`
	Preco     Money  `json:"preco"`
	ImagemURL string `json:"imagem_url"`
}

type ItemInput struct {
	ProdutoID  int `json:"produto_id"`
	Quantidade int `json:"quantidade"`
}

type OrderItem struct {
	ProdutoID     int    `json:"produto_id"`
	Nome          string `json:"nome"`
	Quantidade    int    `json:"quantidade"`
	PrecoUnitario Money  `json:"preco_unitario"`
	Subtotal      Money  `json:"subtotal"`
}

type Order struct {
	ID         int         `json:"id"`
	Status     string      `json:"status"`
	ValorTotal Money       `json:"valor_total"`
	CriadoEm   string      `json:"criado_em"`
	Itens      []OrderItem `json:"itens"`
}
