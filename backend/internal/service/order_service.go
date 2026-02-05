package service

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/nexus-orderbook-dex/backend/internal/blockchain"
	"github.com/nexus-orderbook-dex/backend/internal/domain"
	ob "github.com/nexus-orderbook-dex/backend/internal/orderbook"
	"github.com/nexus-orderbook-dex/backend/internal/repository/postgres"
	redisRepo "github.com/nexus-orderbook-dex/backend/internal/repository/redis"
	"github.com/nexus-orderbook-dex/backend/pkg/eip712"
)

type OrderService struct {
	orderRepo  *postgres.OrderRepo
	tradeRepo  *postgres.TradeRepo
	cache      *redisRepo.OrderbookCache
	orderbooks map[string]*ob.OrderBook
	domain     eip712.DomainSeparator
	settleCh   chan blockchain.SettleJob
}

func NewOrderService(
	orderRepo *postgres.OrderRepo,
	tradeRepo *postgres.TradeRepo,
	cache *redisRepo.OrderbookCache,
	chainID *big.Int,
	contractAddr common.Address,
	settleCh chan blockchain.SettleJob,
) *OrderService {
	return &OrderService{
		orderRepo:  orderRepo,
		tradeRepo:  tradeRepo,
		cache:      cache,
		orderbooks: make(map[string]*ob.OrderBook),
		domain:     eip712.NewDomainSeparator(chainID, contractAddr),
		settleCh:   settleCh,
	}
}

func (s *OrderService) GetOrCreateOrderBook(pair string) *ob.OrderBook {
	if book, ok := s.orderbooks[pair]; ok {
		return book
	}
	book := ob.NewOrderBook(pair)
	s.orderbooks[pair] = book
	return book
}

func (s *OrderService) SubmitOrder(ctx context.Context, sub domain.OrderSubmission) (*domain.Order, []ob.MatchResult, error) {
	amountSell, ok := new(big.Int).SetString(sub.AmountSell, 10)
	if !ok {
		return nil, nil, fmt.Errorf("invalid amountSell")
	}
	amountBuy, ok := new(big.Int).SetString(sub.AmountBuy, 10)
	if !ok {
		return nil, nil, fmt.Errorf("invalid amountBuy")
	}
	salt, ok := new(big.Int).SetString(sub.Salt, 10)
	if !ok {
		return nil, nil, fmt.Errorf("invalid salt")
	}

	// Verify EIP-712 signature
	sigBytes, err := hexToBytes(sub.Signature)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid signature hex: %w", err)
	}

	orderData := eip712.OrderData{
		Maker:      common.HexToAddress(sub.Maker),
		TokenSell:  common.HexToAddress(sub.TokenSell),
		TokenBuy:   common.HexToAddress(sub.TokenBuy),
		AmountSell: amountSell,
		AmountBuy:  amountBuy,
		Expiry:     new(big.Int).SetUint64(sub.Expiry),
		Nonce:      new(big.Int).SetUint64(sub.Nonce),
		Salt:       salt,
	}

	valid, err := eip712.VerifyOrderSignature(s.domain, orderData, sigBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("signature verification failed: %w", err)
	}
	if !valid {
		return nil, nil, fmt.Errorf("invalid signature: signer mismatch")
	}

	order := &domain.Order{
		Maker:      sub.Maker,
		TokenSell:  sub.TokenSell,
		TokenBuy:   sub.TokenBuy,
		AmountSell: amountSell,
		AmountBuy:  amountBuy,
		Expiry:     sub.Expiry,
		Nonce:      sub.Nonce,
		Salt:       salt,
		Signature:  sub.Signature,
		Side:       sub.Side,
		Status:     domain.OrderStatusOpen,
		FilledBase: big.NewInt(0),
		Pair:       sub.Pair,
	}

	// Persist order
	if err := s.orderRepo.Create(ctx, order); err != nil {
		return nil, nil, fmt.Errorf("failed to persist order: %w", err)
	}

	// Add to in-memory orderbook and match
	book := s.GetOrCreateOrderBook(order.Pair)
	matches := book.AddOrder(order)

	// Process matches
	for _, match := range matches {
		trade := &domain.Trade{
			BuyOrderID:  match.BuyOrder.ID,
			SellOrderID: match.SellOrder.ID,
			Buyer:       match.BuyOrder.Maker,
			Seller:      match.SellOrder.Maker,
			Pair:        order.Pair,
			BaseAmount:  match.FillAmount,
			QuoteAmount: match.QuoteAmount,
			Price:       match.Price,
		}

		if err := s.tradeRepo.Create(ctx, trade); err != nil {
			log.Printf("Failed to persist trade: %v", err)
			continue
		}

		// Update order statuses in DB
		s.orderRepo.UpdateStatus(ctx, match.BuyOrder.ID, match.BuyOrder.Status, match.BuyOrder.FilledBase.String())
		s.orderRepo.UpdateStatus(ctx, match.SellOrder.ID, match.SellOrder.Status, match.SellOrder.FilledBase.String())

		// Submit to settlement worker
		resultCh := make(chan blockchain.SettleResult, 1)
		s.settleCh <- blockchain.SettleJob{
			Match:   match,
			TradeID: trade.ID,
			Result:  resultCh,
		}

		// Handle settlement result async
		go func(tradeID string, ch chan blockchain.SettleResult) {
			result := <-ch
			if result.Err != nil {
				log.Printf("Settlement failed for trade %s: %v", tradeID, result.Err)
				return
			}
			if err := s.tradeRepo.MarkSettled(context.Background(), tradeID, result.TxHash); err != nil {
				log.Printf("Failed to mark trade %s settled: %v", tradeID, err)
			}
			log.Printf("Trade %s settled: tx %s", tradeID, result.TxHash)
		}(trade.ID, resultCh)
	}

	// Update cache
	s.updateCache(ctx, order.Pair, book)

	return order, matches, nil
}

func (s *OrderService) CancelOrder(ctx context.Context, orderID string) error {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("order not found: %w", err)
	}

	book := s.GetOrCreateOrderBook(order.Pair)
	if _, ok := book.CancelOrder(orderID); !ok {
		// Order might already be filled or not in the book
	}

	if err := s.orderRepo.UpdateStatus(ctx, orderID, domain.OrderStatusCancelled, order.FilledBase.String()); err != nil {
		return fmt.Errorf("failed to update order: %w", err)
	}

	s.updateCache(ctx, order.Pair, book)
	return nil
}

func (s *OrderService) GetOrderbook(pair string) ob.Snapshot {
	book := s.GetOrCreateOrderBook(pair)
	return book.GetSnapshot()
}

func (s *OrderService) GetOrdersByMaker(ctx context.Context, maker string) ([]*domain.Order, error) {
	return s.orderRepo.GetByMaker(ctx, maker)
}

func (s *OrderService) GetTrades(ctx context.Context, pair string, limit int) ([]*domain.Trade, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.tradeRepo.GetByPair(ctx, pair, limit)
}

// LoadOpenOrders rebuilds the in-memory orderbook from the database on startup.
func (s *OrderService) LoadOpenOrders(ctx context.Context, pair string) error {
	orders, err := s.orderRepo.GetOpenByPair(ctx, pair)
	if err != nil {
		return err
	}
	book := s.GetOrCreateOrderBook(pair)
	for _, order := range orders {
		book.AddOrder(order)
	}
	log.Printf("Loaded %d open orders for %s", len(orders), pair)
	return nil
}

func (s *OrderService) updateCache(ctx context.Context, pair string, book *ob.OrderBook) {
	snapshot := book.GetSnapshot()

	bids := make([]redisRepo.PriceLevelData, len(snapshot.Bids))
	for i, b := range snapshot.Bids {
		bids[i] = redisRepo.PriceLevelData{Price: b.Price, Amount: b.Amount.String(), Count: b.Count}
	}
	asks := make([]redisRepo.PriceLevelData, len(snapshot.Asks))
	for i, a := range snapshot.Asks {
		asks[i] = redisRepo.PriceLevelData{Price: a.Price, Amount: a.Amount.String(), Count: a.Count}
	}

	if err := s.cache.SetSnapshot(ctx, pair, bids, asks); err != nil {
		log.Printf("Failed to update cache: %v", err)
	}

	// Publish update for WebSocket subscribers
	s.cache.PublishUpdate(ctx, pair, map[string]interface{}{
		"type": "orderbook",
		"bids": bids,
		"asks": asks,
	})
}

func hexToBytes(h string) ([]byte, error) {
	h = strings.TrimPrefix(h, "0x")
	return hex.DecodeString(h)
}
