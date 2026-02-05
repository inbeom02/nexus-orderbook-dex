package domain

import (
	"math/big"
	"time"
)

type Trade struct {
	ID            string   `json:"id" db:"id"`
	BuyOrderID    string   `json:"buyOrderId" db:"buy_order_id"`
	SellOrderID   string   `json:"sellOrderId" db:"sell_order_id"`
	Buyer         string   `json:"buyer" db:"buyer"`
	Seller        string   `json:"seller" db:"seller"`
	Pair          string   `json:"pair" db:"pair"`
	BaseAmount    *big.Int `json:"baseAmount" db:"base_amount"`
	QuoteAmount   *big.Int `json:"quoteAmount" db:"quote_amount"`
	Price         float64  `json:"price" db:"price"`
	TxHash        string   `json:"txHash" db:"tx_hash"`
	SettledOnChain bool    `json:"settledOnChain" db:"settled_on_chain"`
	CreatedAt     time.Time `json:"createdAt" db:"created_at"`
}
