package orderbook

import (
	"container/heap"
	"math/big"
	"sync"

	"github.com/nexus-orderbook-dex/backend/internal/domain"
)

// MatchResult represents a single match between a buy and sell order.
type MatchResult struct {
	BuyOrder    *domain.Order
	SellOrder   *domain.Order
	FillAmount  *big.Int // base token amount
	QuoteAmount *big.Int // quote token amount
	Price       float64
}

// PriceLevel represents an aggregated price level in the orderbook.
type PriceLevel struct {
	Price  float64  `json:"price"`
	Amount *big.Int `json:"amount"` // total remaining base token
	Count  int      `json:"count"`
}

// Snapshot is a point-in-time view of the orderbook.
type Snapshot struct {
	Bids []PriceLevel `json:"bids"`
	Asks []PriceLevel `json:"asks"`
}

// OrderBook manages buy and sell orders for a single trading pair.
type OrderBook struct {
	mu       sync.RWMutex
	pair     string
	buys     *BuyHeap
	sells    *SellHeap
	orderMap map[string]*OrderEntry // orderID -> entry
}

// NewOrderBook creates a new orderbook for the given pair.
func NewOrderBook(pair string) *OrderBook {
	bh := &BuyHeap{}
	sh := &SellHeap{}
	heap.Init(bh)
	heap.Init(sh)
	return &OrderBook{
		pair:     pair,
		buys:     bh,
		sells:    sh,
		orderMap: make(map[string]*OrderEntry),
	}
}

// AddOrder adds a new order and attempts to match it. Returns match results.
func (ob *OrderBook) AddOrder(order *domain.Order) []MatchResult {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	var matches []MatchResult

	if order.Side == domain.SideBuy {
		matches = ob.matchBuy(order)
	} else {
		matches = ob.matchSell(order)
	}

	// If order still has remaining quantity, add to book
	if order.RemainingBase().Sign() > 0 && order.Status != domain.OrderStatusFilled {
		entry := &OrderEntry{Order: order}
		ob.orderMap[order.ID] = entry
		if order.Side == domain.SideBuy {
			heap.Push(ob.buys, entry)
		} else {
			heap.Push(ob.sells, entry)
		}
	}

	return matches
}

func (ob *OrderBook) matchBuy(buyOrder *domain.Order) []MatchResult {
	var matches []MatchResult
	buyPrice := buyOrder.Price()

	for ob.sells.Len() > 0 && buyOrder.RemainingBase().Sign() > 0 {
		bestSell := (*ob.sells)[0]

		// Check price compatibility: buy price >= sell price
		if buyPrice < bestSell.Order.Price() {
			break
		}

		fillAmount := minBigInt(buyOrder.RemainingBase(), bestSell.Order.RemainingBase())
		if fillAmount.Sign() <= 0 {
			break
		}

		// Calculate quote amount at the sell price (maker price)
		// quoteAmount = fillAmount * sellOrder.amountBuy / sellOrder.amountSell
		quoteAmount := new(big.Int).Mul(fillAmount, bestSell.Order.AmountBuy)
		quoteAmount.Div(quoteAmount, bestSell.Order.AmountSell)

		matches = append(matches, MatchResult{
			BuyOrder:    buyOrder,
			SellOrder:   bestSell.Order,
			FillAmount:  new(big.Int).Set(fillAmount),
			QuoteAmount: quoteAmount,
			Price:       bestSell.Order.Price(),
		})

		// Update filled amounts
		buyOrder.FilledBase = new(big.Int).Add(buyOrder.FilledBase, fillAmount)
		bestSell.Order.FilledBase = new(big.Int).Add(bestSell.Order.FilledBase, fillAmount)

		// Update statuses
		updateOrderStatus(buyOrder)
		updateOrderStatus(bestSell.Order)

		// Remove sell order if fully filled
		if bestSell.Order.Status == domain.OrderStatusFilled {
			heap.Pop(ob.sells)
			delete(ob.orderMap, bestSell.Order.ID)
		}
	}

	return matches
}

func (ob *OrderBook) matchSell(sellOrder *domain.Order) []MatchResult {
	var matches []MatchResult
	sellPrice := sellOrder.Price()

	for ob.buys.Len() > 0 && sellOrder.RemainingBase().Sign() > 0 {
		bestBuy := (*ob.buys)[0]

		// Check price compatibility: buy price >= sell price
		if bestBuy.Order.Price() < sellPrice {
			break
		}

		fillAmount := minBigInt(sellOrder.RemainingBase(), bestBuy.Order.RemainingBase())
		if fillAmount.Sign() <= 0 {
			break
		}

		// Calculate quote amount at the buy price (maker price)
		// quoteAmount = fillAmount * sellOrder.amountBuy / sellOrder.amountSell
		quoteAmount := new(big.Int).Mul(fillAmount, sellOrder.AmountBuy)
		quoteAmount.Div(quoteAmount, sellOrder.AmountSell)

		matches = append(matches, MatchResult{
			BuyOrder:    bestBuy.Order,
			SellOrder:   sellOrder,
			FillAmount:  new(big.Int).Set(fillAmount),
			QuoteAmount: quoteAmount,
			Price:       bestBuy.Order.Price(),
		})

		sellOrder.FilledBase = new(big.Int).Add(sellOrder.FilledBase, fillAmount)
		bestBuy.Order.FilledBase = new(big.Int).Add(bestBuy.Order.FilledBase, fillAmount)

		updateOrderStatus(sellOrder)
		updateOrderStatus(bestBuy.Order)

		if bestBuy.Order.Status == domain.OrderStatusFilled {
			heap.Pop(ob.buys)
			delete(ob.orderMap, bestBuy.Order.ID)
		}
	}

	return matches
}

// CancelOrder removes an order from the book.
func (ob *OrderBook) CancelOrder(orderID string) (*domain.Order, bool) {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	entry, ok := ob.orderMap[orderID]
	if !ok {
		return nil, false
	}

	entry.Order.Status = domain.OrderStatusCancelled
	delete(ob.orderMap, orderID)

	// Remove from heap by setting to worst priority and popping
	if entry.Order.Side == domain.SideBuy {
		heap.Remove(ob.buys, entry.Index)
	} else {
		heap.Remove(ob.sells, entry.Index)
	}

	return entry.Order, true
}

// GetSnapshot returns the current orderbook state aggregated by price level.
func (ob *OrderBook) GetSnapshot() Snapshot {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	bids := aggregateLevels(ob.buys)
	asks := aggregateSellLevels(ob.sells)

	return Snapshot{Bids: bids, Asks: asks}
}

func aggregateLevels(h *BuyHeap) []PriceLevel {
	levels := make(map[float64]*PriceLevel)
	var prices []float64
	for _, entry := range *h {
		p := entry.Order.Price()
		if lvl, ok := levels[p]; ok {
			lvl.Amount = new(big.Int).Add(lvl.Amount, entry.Order.RemainingBase())
			lvl.Count++
		} else {
			levels[p] = &PriceLevel{
				Price:  p,
				Amount: new(big.Int).Set(entry.Order.RemainingBase()),
				Count:  1,
			}
			prices = append(prices, p)
		}
	}
	// Sort descending (best bid first)
	sortDesc(prices)
	result := make([]PriceLevel, 0, len(prices))
	for _, p := range prices {
		result = append(result, *levels[p])
	}
	return result
}

func aggregateSellLevels(h *SellHeap) []PriceLevel {
	levels := make(map[float64]*PriceLevel)
	var prices []float64
	for _, entry := range *h {
		p := entry.Order.Price()
		if lvl, ok := levels[p]; ok {
			lvl.Amount = new(big.Int).Add(lvl.Amount, entry.Order.RemainingBase())
			lvl.Count++
		} else {
			levels[p] = &PriceLevel{
				Price:  p,
				Amount: new(big.Int).Set(entry.Order.RemainingBase()),
				Count:  1,
			}
			prices = append(prices, p)
		}
	}
	// Sort ascending (best ask first)
	sortAsc(prices)
	result := make([]PriceLevel, 0, len(prices))
	for _, p := range prices {
		result = append(result, *levels[p])
	}
	return result
}

func sortDesc(s []float64) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] > s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

func sortAsc(s []float64) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

func updateOrderStatus(o *domain.Order) {
	if o.RemainingBase().Sign() <= 0 {
		o.Status = domain.OrderStatusFilled
	} else if o.FilledBase.Sign() > 0 {
		o.Status = domain.OrderStatusPartiallyFilled
	}
}

func minBigInt(a, b *big.Int) *big.Int {
	if a.Cmp(b) < 0 {
		return new(big.Int).Set(a)
	}
	return new(big.Int).Set(b)
}
