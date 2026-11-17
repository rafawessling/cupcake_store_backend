package db

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/lib/pq"

	"cupcake_store/backend/internal/models"
)

var (
	ErrNotFound   = errors.New("não encontrado")
	ErrEmailTaken = errors.New("e-mail já cadastrado")
)

// ProductUnavailableError indica produto inexistente ou inativo em um pedido.
type ProductUnavailableError struct{ ProdutoID int }

func (e ProductUnavailableError) Error() string { return "produto indisponível" }

type Store interface {
	CreateUser(ctx context.Context, nome, email, senhaHash string) (models.User, error)
	UserByEmail(ctx context.Context, email string) (models.User, string, error)
	ListProducts(ctx context.Context, busca string) ([]models.Product, error)
	ProductByID(ctx context.Context, id int) (models.Product, error)
	CreateOrder(ctx context.Context, userID int, itens []models.ItemInput) (models.Order, error)
	OrderByID(ctx context.Context, id, userID int) (models.Order, error)
}

type PostgresStore struct{ db *sql.DB }

func NewStore(conn *sql.DB) *PostgresStore { return &PostgresStore{db: conn} }

const productCols = `id, nome, COALESCE(descricao, ''), (preco*100)::bigint, COALESCE(imagem_url, '')`

type scanner interface{ Scan(...any) error }

func scanProduct(s scanner) (p models.Product, err error) {
	err = s.Scan(&p.ID, &p.Nome, &p.Descricao, &p.Preco, &p.ImagemURL)
	return
}

func (s *PostgresStore) CreateUser(ctx context.Context, nome, email, senhaHash string) (models.User, error) {
	u := models.User{Nome: nome, Email: email}
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO usuarios (nome, email, senha_hash) VALUES ($1, $2, $3) RETURNING id`,
		nome, email, senhaHash).Scan(&u.ID)
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23505" {
		return u, ErrEmailTaken
	}
	return u, err
}

func (s *PostgresStore) UserByEmail(ctx context.Context, email string) (models.User, string, error) {
	var u models.User
	var hash string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, nome, email, senha_hash FROM usuarios WHERE email = $1`, email).
		Scan(&u.ID, &u.Nome, &u.Email, &hash)
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrNotFound
	}
	return u, hash, err
}

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func (s *PostgresStore) ListProducts(ctx context.Context, busca string) ([]models.Product, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+productCols+` FROM produtos
		 WHERE ativo AND nome ILIKE '%' || $1 || '%' ORDER BY id`,
		likeEscaper.Replace(busca))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := []models.Product{}
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, rows.Err()
}

func (s *PostgresStore) ProductByID(ctx context.Context, id int) (models.Product, error) {
	p, err := scanProduct(s.db.QueryRowContext(ctx,
		`SELECT `+productCols+` FROM produtos WHERE id = $1 AND ativo`, id))
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrNotFound
	}
	return p, err
}

// withTx executa fn em uma transação, com rollback em qualquer erro.
func (s *PostgresStore) withTx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

// CreateOrder busca os preços no banco e calcula subtotais e total; nada vindo do cliente é confiado.
func (s *PostgresStore) CreateOrder(ctx context.Context, userID int, itens []models.ItemInput) (models.Order, error) {
	order := models.Order{Status: "confirmado", Itens: make([]models.OrderItem, 0, len(itens))}

	err := s.withTx(ctx, func(tx *sql.Tx) error {
		ids := make([]int64, len(itens))
		for i, it := range itens {
			ids[i] = int64(it.ProdutoID)
		}
		rows, err := tx.QueryContext(ctx,
			`SELECT id, nome, (preco*100)::bigint FROM produtos WHERE ativo AND id = ANY($1)`, pq.Array(ids))
		if err != nil {
			return err
		}
		defer rows.Close()
		found := make(map[int]models.OrderItem, len(itens))
		for rows.Next() {
			var oi models.OrderItem
			if err := rows.Scan(&oi.ProdutoID, &oi.Nome, &oi.PrecoUnitario); err != nil {
				return err
			}
			found[oi.ProdutoID] = oi
		}
		if err := rows.Err(); err != nil {
			return err
		}
		rows.Close()

		for _, it := range itens {
			oi, ok := found[it.ProdutoID]
			if !ok {
				return ProductUnavailableError{ProdutoID: it.ProdutoID}
			}
			oi.Quantidade = it.Quantidade
			oi.Subtotal = oi.PrecoUnitario * models.Money(it.Quantidade)
			order.ValorTotal += oi.Subtotal
			order.Itens = append(order.Itens, oi)
		}

		var criado time.Time
		if err := tx.QueryRowContext(ctx,
			`INSERT INTO pedidos (usuario_id, valor_total, status) VALUES ($1, $2::bigint / 100.0, $3)
			 RETURNING id, criado_em`,
			userID, int64(order.ValorTotal), order.Status).Scan(&order.ID, &criado); err != nil {
			return err
		}
		order.CriadoEm = formatTime(criado)

		for _, oi := range order.Itens {
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO itens_pedido (pedido_id, produto_id, quantidade, preco_unitario, subtotal)
				 VALUES ($1, $2, $3, $4::bigint / 100.0, $5::bigint / 100.0)`,
				order.ID, oi.ProdutoID, oi.Quantidade, int64(oi.PrecoUnitario), int64(oi.Subtotal)); err != nil {
				return err
			}
		}
		return nil
	})
	return order, err
}

// OrderByID só devolve o pedido se pertencer a userID.
func (s *PostgresStore) OrderByID(ctx context.Context, id, userID int) (models.Order, error) {
	var o models.Order
	var criado time.Time
	err := s.db.QueryRowContext(ctx,
		`SELECT id, status, (valor_total*100)::bigint, criado_em FROM pedidos WHERE id = $1 AND usuario_id = $2`,
		id, userID).Scan(&o.ID, &o.Status, &o.ValorTotal, &criado)
	if errors.Is(err, sql.ErrNoRows) {
		return o, ErrNotFound
	}
	if err != nil {
		return o, err
	}
	o.CriadoEm = formatTime(criado)

	rows, err := s.db.QueryContext(ctx,
		`SELECT i.produto_id, p.nome, i.quantidade, (i.preco_unitario*100)::bigint, (i.subtotal*100)::bigint
		 FROM itens_pedido i JOIN produtos p ON p.id = i.produto_id
		 WHERE i.pedido_id = $1 ORDER BY i.id`, id)
	if err != nil {
		return o, err
	}
	defer rows.Close()
	o.Itens = []models.OrderItem{}
	for rows.Next() {
		var oi models.OrderItem
		if err := rows.Scan(&oi.ProdutoID, &oi.Nome, &oi.Quantidade, &oi.PrecoUnitario, &oi.Subtotal); err != nil {
			return o, err
		}
		o.Itens = append(o.Itens, oi)
	}
	return o, rows.Err()
}

// criado_em é gravado em UTC (ver schema.sql).
func formatTime(t time.Time) string { return t.UTC().Format(time.RFC3339) }
