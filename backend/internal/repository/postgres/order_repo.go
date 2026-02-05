package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/nexus-orderbook-dex/backend/internal/domain"
)

type OrderRepo struct {
	db *sqlx.DB
}

func NewOrderRepo(db *sqlx.DB) *OrderRepo {
	return &OrderRepo{db: db}
}

type orderRow struct {
	ID         string    `db:"id"`
	Maker      string    `db:"maker"`
	TokenSell  string    `db:"token_sell"`
	TokenBuy   string    `db:"token_buy"`
	AmountSell string    `db:"amount_sell"`
	AmountBuy  string    `db:"amount_buy"`
	Expiry     int64     `db:"expiry"`
	Nonce      int64     `db:"nonce"`
	Salt       string    `db:"salt"`
	Signature  string    `db:"signature"`
	Side       string    `db:"side"`
	Status     string    `db:"status"`
	FilledBase string    `db:"filled_base"`
	Pair       string    `db:"pair"`
	CreatedAt  time.Time `db:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"`
}

func (r *OrderRepo) Create(ctx context.Context, order *domain.Order) error {
	if order.ID == "" {
		order.ID = uuid.New().String()
	}
	now := time.Now()
	order.CreatedAt = now
	order.UpdatedAt = now

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO orders (id, maker, token_sell, token_buy, amount_sell, amount_buy, expiry, nonce, salt, signature, side, status, filled_base, pair, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`,
		order.ID, order.Maker, order.TokenSell, order.TokenBuy,
		order.AmountSell.String(), order.AmountBuy.String(),
		order.Expiry, order.Nonce, order.Salt.String(),
		order.Signature, string(order.Side), string(order.Status),
		order.FilledBase.String(), order.Pair, order.CreatedAt, order.UpdatedAt,
	)
	return err
}

func (r *OrderRepo) GetByID(ctx context.Context, id string) (*domain.Order, error) {
	var row orderRow
	err := r.db.GetContext(ctx, &row, `SELECT * FROM orders WHERE id = $1`, id)
	if err != nil {
		return nil, err
	}
	return rowToOrder(row)
}

func (r *OrderRepo) GetByMaker(ctx context.Context, maker string) ([]*domain.Order, error) {
	var rows []orderRow
	err := r.db.SelectContext(ctx, &rows, `SELECT * FROM orders WHERE maker = $1 ORDER BY created_at DESC`, maker)
	if err != nil {
		return nil, err
	}
	return rowsToOrders(rows)
}

func (r *OrderRepo) GetOpenByPair(ctx context.Context, pair string) ([]*domain.Order, error) {
	var rows []orderRow
	err := r.db.SelectContext(ctx, &rows,
		`SELECT * FROM orders WHERE pair = $1 AND status IN ('open', 'partially_filled') ORDER BY created_at ASC`, pair)
	if err != nil {
		return nil, err
	}
	return rowsToOrders(rows)
}

func (r *OrderRepo) UpdateStatus(ctx context.Context, id string, status domain.OrderStatus, filledBase string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE orders SET status = $1, filled_base = $2, updated_at = NOW() WHERE id = $3`,
		string(status), filledBase, id)
	return err
}

func rowToOrder(row orderRow) (*domain.Order, error) {
	amountSell, ok := parseBigInt(row.AmountSell)
	if !ok {
		return nil, fmt.Errorf("invalid amount_sell: %s", row.AmountSell)
	}
	amountBuy, ok := parseBigInt(row.AmountBuy)
	if !ok {
		return nil, fmt.Errorf("invalid amount_buy: %s", row.AmountBuy)
	}
	salt, ok := parseBigInt(row.Salt)
	if !ok {
		return nil, fmt.Errorf("invalid salt: %s", row.Salt)
	}
	filledBase, ok := parseBigInt(row.FilledBase)
	if !ok {
		return nil, fmt.Errorf("invalid filled_base: %s", row.FilledBase)
	}

	return &domain.Order{
		ID:         row.ID,
		Maker:      row.Maker,
		TokenSell:  row.TokenSell,
		TokenBuy:   row.TokenBuy,
		AmountSell: amountSell,
		AmountBuy:  amountBuy,
		Expiry:     uint64(row.Expiry),
		Nonce:      uint64(row.Nonce),
		Salt:       salt,
		Signature:  row.Signature,
		Side:       domain.Side(row.Side),
		Status:     domain.OrderStatus(row.Status),
		FilledBase: filledBase,
		Pair:       row.Pair,
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
	}, nil
}

func rowsToOrders(rows []orderRow) ([]*domain.Order, error) {
	orders := make([]*domain.Order, 0, len(rows))
	for _, row := range rows {
		o, err := rowToOrder(row)
		if err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, nil
}
