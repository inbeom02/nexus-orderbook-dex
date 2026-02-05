package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/nexus-orderbook-dex/backend/internal/domain"
)

type TradeRepo struct {
	db *sqlx.DB
}

func NewTradeRepo(db *sqlx.DB) *TradeRepo {
	return &TradeRepo{db: db}
}

type tradeRow struct {
	ID             string    `db:"id"`
	BuyOrderID     string    `db:"buy_order_id"`
	SellOrderID    string    `db:"sell_order_id"`
	Buyer          string    `db:"buyer"`
	Seller         string    `db:"seller"`
	Pair           string    `db:"pair"`
	BaseAmount     string    `db:"base_amount"`
	QuoteAmount    string    `db:"quote_amount"`
	Price          float64   `db:"price"`
	TxHash         string    `db:"tx_hash"`
	SettledOnChain bool      `db:"settled_on_chain"`
	CreatedAt      time.Time `db:"created_at"`
}

func (r *TradeRepo) Create(ctx context.Context, trade *domain.Trade) error {
	if trade.ID == "" {
		trade.ID = uuid.New().String()
	}
	trade.CreatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO trades (id, buy_order_id, sell_order_id, buyer, seller, pair, base_amount, quote_amount, price, tx_hash, settled_on_chain, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		trade.ID, trade.BuyOrderID, trade.SellOrderID,
		trade.Buyer, trade.Seller, trade.Pair,
		trade.BaseAmount.String(), trade.QuoteAmount.String(),
		trade.Price, trade.TxHash, trade.SettledOnChain, trade.CreatedAt,
	)
	return err
}

func (r *TradeRepo) GetByPair(ctx context.Context, pair string, limit int) ([]*domain.Trade, error) {
	var rows []tradeRow
	err := r.db.SelectContext(ctx, &rows,
		`SELECT * FROM trades WHERE pair = $1 ORDER BY created_at DESC LIMIT $2`, pair, limit)
	if err != nil {
		return nil, err
	}
	return rowsToTrades(rows)
}

func (r *TradeRepo) MarkSettled(ctx context.Context, id string, txHash string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE trades SET settled_on_chain = TRUE, tx_hash = $1 WHERE id = $2`, txHash, id)
	return err
}

func (r *TradeRepo) GetUnsettled(ctx context.Context) ([]*domain.Trade, error) {
	var rows []tradeRow
	err := r.db.SelectContext(ctx, &rows,
		`SELECT * FROM trades WHERE settled_on_chain = FALSE ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	return rowsToTrades(rows)
}

func rowToTrade(row tradeRow) (*domain.Trade, error) {
	baseAmount, ok := parseBigInt(row.BaseAmount)
	if !ok {
		return nil, fmt.Errorf("invalid base_amount: %s", row.BaseAmount)
	}
	quoteAmount, ok := parseBigInt(row.QuoteAmount)
	if !ok {
		return nil, fmt.Errorf("invalid quote_amount: %s", row.QuoteAmount)
	}

	return &domain.Trade{
		ID:             row.ID,
		BuyOrderID:     row.BuyOrderID,
		SellOrderID:    row.SellOrderID,
		Buyer:          row.Buyer,
		Seller:         row.Seller,
		Pair:           row.Pair,
		BaseAmount:     baseAmount,
		QuoteAmount:    quoteAmount,
		Price:          row.Price,
		TxHash:         row.TxHash,
		SettledOnChain: row.SettledOnChain,
		CreatedAt:      row.CreatedAt,
	}, nil
}

func rowsToTrades(rows []tradeRow) ([]*domain.Trade, error) {
	trades := make([]*domain.Trade, 0, len(rows))
	for _, row := range rows {
		t, err := rowToTrade(row)
		if err != nil {
			return nil, err
		}
		trades = append(trades, t)
	}
	return trades, nil
}
