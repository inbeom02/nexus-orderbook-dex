package orderbook

import (
	"math/big"
	"testing"
	"time"

	"github.com/nexus-orderbook-dex/backend/internal/domain"
)

func makeOrder(id string, side domain.Side, amountSell, amountBuy int64) *domain.Order {
	return &domain.Order{
		ID:         id,
		Maker:      "0x1234",
		TokenSell:  "0xTKB",
		TokenBuy:   "0xTKA",
		AmountSell: big.NewInt(amountSell),
		AmountBuy:  big.NewInt(amountBuy),
		Side:       side,
		Status:     domain.OrderStatusOpen,
		FilledBase: big.NewInt(0),
		Pair:       "TKA-TKB",
		CreatedAt:  time.Now(),
	}
}

func TestAddBuyOrder_NoMatch(t *testing.T) {
	ob := NewOrderBook("TKA-TKB")
	// Buy 100 TKA for 200 TKB (price = 2)
	order := makeOrder("buy-1", domain.SideBuy, 200, 100)
	matches := ob.AddOrder(order)

	if len(matches) != 0 {
		t.Fatalf("expected 0 matches, got %d", len(matches))
	}

	snap := ob.GetSnapshot()
	if len(snap.Bids) != 1 {
		t.Fatalf("expected 1 bid level, got %d", len(snap.Bids))
	}
	if snap.Bids[0].Price != 2.0 {
		t.Fatalf("expected price 2.0, got %f", snap.Bids[0].Price)
	}
}

func TestAddSellOrder_NoMatch(t *testing.T) {
	ob := NewOrderBook("TKA-TKB")
	// Sell 100 TKA for 300 TKB (price = 3)
	order := makeOrder("sell-1", domain.SideSell, 100, 300)
	matches := ob.AddOrder(order)

	if len(matches) != 0 {
		t.Fatalf("expected 0 matches, got %d", len(matches))
	}

	snap := ob.GetSnapshot()
	if len(snap.Asks) != 1 {
		t.Fatalf("expected 1 ask level, got %d", len(snap.Asks))
	}
}

func TestMatchBuyAndSell(t *testing.T) {
	ob := NewOrderBook("TKA-TKB")

	// Sell 100 TKA for 200 TKB (price = 2)
	sell := &domain.Order{
		ID:         "sell-1",
		Maker:      "0xSeller",
		TokenSell:  "0xTKA",
		TokenBuy:   "0xTKB",
		AmountSell: big.NewInt(100),
		AmountBuy:  big.NewInt(200),
		Side:       domain.SideSell,
		Status:     domain.OrderStatusOpen,
		FilledBase: big.NewInt(0),
		Pair:       "TKA-TKB",
		CreatedAt:  time.Now(),
	}
	ob.AddOrder(sell)

	// Buy 100 TKA for 250 TKB (price = 2.5, willing to pay more)
	buy := &domain.Order{
		ID:         "buy-1",
		Maker:      "0xBuyer",
		TokenSell:  "0xTKB",
		TokenBuy:   "0xTKA",
		AmountSell: big.NewInt(250),
		AmountBuy:  big.NewInt(100),
		Side:       domain.SideBuy,
		Status:     domain.OrderStatusOpen,
		FilledBase: big.NewInt(0),
		Pair:       "TKA-TKB",
		CreatedAt:  time.Now(),
	}
	matches := ob.AddOrder(buy)

	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}

	m := matches[0]
	if m.FillAmount.Int64() != 100 {
		t.Fatalf("expected fill 100, got %d", m.FillAmount.Int64())
	}

	// Both orders should be fully filled
	if sell.Status != domain.OrderStatusFilled {
		t.Fatalf("sell should be filled, got %s", sell.Status)
	}
	if buy.Status != domain.OrderStatusFilled {
		t.Fatalf("buy should be filled, got %s", buy.Status)
	}

	// Orderbook should be empty
	snap := ob.GetSnapshot()
	if len(snap.Bids) != 0 || len(snap.Asks) != 0 {
		t.Fatalf("expected empty book, got %d bids, %d asks", len(snap.Bids), len(snap.Asks))
	}
}

func TestPartialMatch(t *testing.T) {
	ob := NewOrderBook("TKA-TKB")

	// Sell 50 TKA for 100 TKB (price = 2)
	sell := &domain.Order{
		ID:         "sell-1",
		Maker:      "0xSeller",
		TokenSell:  "0xTKA",
		TokenBuy:   "0xTKB",
		AmountSell: big.NewInt(50),
		AmountBuy:  big.NewInt(100),
		Side:       domain.SideSell,
		Status:     domain.OrderStatusOpen,
		FilledBase: big.NewInt(0),
		Pair:       "TKA-TKB",
		CreatedAt:  time.Now(),
	}
	ob.AddOrder(sell)

	// Buy 100 TKA for 200 TKB (price = 2)
	buy := &domain.Order{
		ID:         "buy-1",
		Maker:      "0xBuyer",
		TokenSell:  "0xTKB",
		TokenBuy:   "0xTKA",
		AmountSell: big.NewInt(200),
		AmountBuy:  big.NewInt(100),
		Side:       domain.SideBuy,
		Status:     domain.OrderStatusOpen,
		FilledBase: big.NewInt(0),
		Pair:       "TKA-TKB",
		CreatedAt:  time.Now(),
	}
	matches := ob.AddOrder(buy)

	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}

	if matches[0].FillAmount.Int64() != 50 {
		t.Fatalf("expected fill 50, got %d", matches[0].FillAmount.Int64())
	}

	// Sell fully filled, buy partially filled and resting
	if sell.Status != domain.OrderStatusFilled {
		t.Fatalf("sell should be filled, got %s", sell.Status)
	}
	if buy.Status != domain.OrderStatusPartiallyFilled {
		t.Fatalf("buy should be partially_filled, got %s", buy.Status)
	}

	snap := ob.GetSnapshot()
	if len(snap.Bids) != 1 {
		t.Fatalf("expected 1 bid resting, got %d", len(snap.Bids))
	}
}

func TestPriceIncompatible(t *testing.T) {
	ob := NewOrderBook("TKA-TKB")

	// Sell 100 TKA for 300 TKB (price = 3)
	sell := &domain.Order{
		ID:         "sell-1",
		Maker:      "0xSeller",
		TokenSell:  "0xTKA",
		TokenBuy:   "0xTKB",
		AmountSell: big.NewInt(100),
		AmountBuy:  big.NewInt(300),
		Side:       domain.SideSell,
		Status:     domain.OrderStatusOpen,
		FilledBase: big.NewInt(0),
		Pair:       "TKA-TKB",
		CreatedAt:  time.Now(),
	}
	ob.AddOrder(sell)

	// Buy 100 TKA for 200 TKB (price = 2, below seller's 3)
	buy := &domain.Order{
		ID:         "buy-1",
		Maker:      "0xBuyer",
		TokenSell:  "0xTKB",
		TokenBuy:   "0xTKA",
		AmountSell: big.NewInt(200),
		AmountBuy:  big.NewInt(100),
		Side:       domain.SideBuy,
		Status:     domain.OrderStatusOpen,
		FilledBase: big.NewInt(0),
		Pair:       "TKA-TKB",
		CreatedAt:  time.Now(),
	}
	matches := ob.AddOrder(buy)

	if len(matches) != 0 {
		t.Fatalf("expected 0 matches (price incompatible), got %d", len(matches))
	}

	snap := ob.GetSnapshot()
	if len(snap.Bids) != 1 || len(snap.Asks) != 1 {
		t.Fatalf("expected 1 bid and 1 ask, got %d bids, %d asks", len(snap.Bids), len(snap.Asks))
	}
}

func TestCancelOrder(t *testing.T) {
	ob := NewOrderBook("TKA-TKB")

	order := makeOrder("buy-1", domain.SideBuy, 200, 100)
	ob.AddOrder(order)

	cancelled, ok := ob.CancelOrder("buy-1")
	if !ok {
		t.Fatal("cancel should succeed")
	}
	if cancelled.Status != domain.OrderStatusCancelled {
		t.Fatalf("expected cancelled, got %s", cancelled.Status)
	}

	snap := ob.GetSnapshot()
	if len(snap.Bids) != 0 {
		t.Fatalf("expected 0 bids after cancel, got %d", len(snap.Bids))
	}
}
