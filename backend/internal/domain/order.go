package domain

import (
	"math/big"
	"time"
)

type Side string

const (
	SideBuy  Side = "buy"
	SideSell Side = "sell"
)

type OrderStatus string

const (
	OrderStatusOpen            OrderStatus = "open"
	OrderStatusPartiallyFilled OrderStatus = "partially_filled"
	OrderStatusFilled          OrderStatus = "filled"
	OrderStatusCancelled       OrderStatus = "cancelled"
)

type Order struct {
	ID         string      `json:"id" db:"id"`
	Maker      string      `json:"maker" db:"maker"`
	TokenSell  string      `json:"tokenSell" db:"token_sell"`
	TokenBuy   string      `json:"tokenBuy" db:"token_buy"`
	AmountSell *big.Int    `json:"amountSell" db:"amount_sell"`
	AmountBuy  *big.Int    `json:"amountBuy" db:"amount_buy"`
	Expiry     uint64      `json:"expiry" db:"expiry"`
	Nonce      uint64      `json:"nonce" db:"nonce"`
	Salt       *big.Int    `json:"salt" db:"salt"`
	Signature  string      `json:"signature" db:"signature"`
	Side       Side        `json:"side" db:"side"`
	Status     OrderStatus `json:"status" db:"status"`
	FilledBase *big.Int    `json:"filledBase" db:"filled_base"`
	Pair       string      `json:"pair" db:"pair"`
	CreatedAt  time.Time   `json:"createdAt" db:"created_at"`
	UpdatedAt  time.Time   `json:"updatedAt" db:"updated_at"`
}

// Price returns the price as a float64 (quote/base).
// For a buy order: amountSell/amountBuy (how much quote per base)
// For a sell order: amountBuy/amountSell (how much quote per base)
func (o *Order) Price() float64 {
	if o.Side == SideBuy {
		num := new(big.Float).SetInt(o.AmountSell)
		den := new(big.Float).SetInt(o.AmountBuy)
		price, _ := new(big.Float).Quo(num, den).Float64()
		return price
	}
	num := new(big.Float).SetInt(o.AmountBuy)
	den := new(big.Float).SetInt(o.AmountSell)
	price, _ := new(big.Float).Quo(num, den).Float64()
	return price
}

// RemainingBase returns remaining base token amount to fill.
func (o *Order) RemainingBase() *big.Int {
	if o.Side == SideBuy {
		return new(big.Int).Sub(o.AmountBuy, o.FilledBase)
	}
	return new(big.Int).Sub(o.AmountSell, o.FilledBase)
}

// OrderSubmission is the JSON payload for submitting a new order.
type OrderSubmission struct {
	Maker      string `json:"maker" binding:"required"`
	TokenSell  string `json:"tokenSell" binding:"required"`
	TokenBuy   string `json:"tokenBuy" binding:"required"`
	AmountSell string `json:"amountSell" binding:"required"`
	AmountBuy  string `json:"amountBuy" binding:"required"`
	Expiry     uint64 `json:"expiry" binding:"required"`
	Nonce      uint64 `json:"nonce"`
	Salt       string `json:"salt" binding:"required"`
	Signature  string `json:"signature" binding:"required"`
	Side       Side   `json:"side" binding:"required"`
	Pair       string `json:"pair" binding:"required"`
}
