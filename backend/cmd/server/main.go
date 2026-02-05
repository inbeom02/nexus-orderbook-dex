package main

import (
	"context"
	"log"
	"math/big"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"

	"github.com/nexus-orderbook-dex/backend/internal/blockchain"
	"github.com/nexus-orderbook-dex/backend/internal/config"
	"github.com/nexus-orderbook-dex/backend/internal/handler"
	"github.com/nexus-orderbook-dex/backend/internal/repository/postgres"
	redisRepo "github.com/nexus-orderbook-dex/backend/internal/repository/redis"
	"github.com/nexus-orderbook-dex/backend/internal/service"
)

func main() {
	cfg := config.Load()

	// Database
	db, err := sqlx.Connect("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run migration
	migrationSQL, err := os.ReadFile("migrations/001_init.sql")
	if err != nil {
		log.Printf("Warning: could not read migration file: %v", err)
	} else {
		if _, err := db.Exec(string(migrationSQL)); err != nil {
			log.Printf("Warning: migration may have already been applied: %v", err)
		}
	}

	// Redis
	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.RedisURL,
	})
	defer rdb.Close()
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	// Blockchain client
	privateKey := strings.TrimPrefix(cfg.PrivateKey, "0x")
	chainID, _ := new(big.Int).SetString(cfg.ChainID, 10)

	var bcClient *blockchain.Client
	var settleCh chan blockchain.SettleJob

	if cfg.ContractAddress != "" && cfg.PrivateKey != "" {
		bcClient, err = blockchain.NewClient(cfg.RPCUrl, privateKey, chainID.Int64(), cfg.ContractAddress)
		if err != nil {
			log.Fatalf("Failed to create blockchain client: %v", err)
		}

		// Settlement worker
		settleCh = make(chan blockchain.SettleJob, 100)
		worker, err := blockchain.NewSettlementWorker(bcClient)
		if err != nil {
			log.Fatalf("Failed to create settlement worker: %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go worker.Run(ctx, settleCh)

		// Event indexer
		indexer := blockchain.NewIndexer(bcClient, &noopHandler{})
		go indexer.Start(ctx)
	} else {
		log.Println("Warning: blockchain not configured, settlement disabled")
		settleCh = make(chan blockchain.SettleJob, 100)
		// Drain settlement channel
		go func() {
			for job := range settleCh {
				job.Result <- blockchain.SettleResult{Err: nil, TxHash: "0x_mock"}
			}
		}()
	}

	// Repos
	orderRepo := postgres.NewOrderRepo(db)
	tradeRepo := postgres.NewTradeRepo(db)
	cache := redisRepo.NewOrderbookCache(rdb)

	// Service
	contractAddr := common.HexToAddress(cfg.ContractAddress)
	if chainID == nil {
		chainID = big.NewInt(31337)
	}
	orderSvc := service.NewOrderService(orderRepo, tradeRepo, cache, chainID, contractAddr, settleCh)

	// Load existing open orders
	if err := orderSvc.LoadOpenOrders(context.Background(), "TKA-TKB"); err != nil {
		log.Printf("Warning: failed to load open orders: %v", err)
	}

	// Handlers
	orderH := handler.NewOrderHandler(orderSvc)
	orderbookH := handler.NewOrderbookHandler(orderSvc)
	tradeH := handler.NewTradeHandler(orderSvc)
	wsH := handler.NewWSHandler(cache)

	// Router
	r := gin.Default()

	// CORS
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	api := r.Group("/api")
	{
		api.POST("/orders", orderH.SubmitOrder)
		api.GET("/orders/:address", orderH.GetUserOrders)
		api.DELETE("/orders/:id", orderH.CancelOrder)
		api.GET("/orderbook", orderbookH.GetOrderbook)
		api.GET("/trades", tradeH.GetTrades)
	}

	r.GET("/ws", wsH.Handle)

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Graceful shutdown
	go func() {
		if err := r.Run(":" + cfg.ServerPort); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down...")
}

// noopHandler is a placeholder event handler.
type noopHandler struct{}

func (h *noopHandler) OnDeposit(user, token common.Address, amount *big.Int) {
	log.Printf("Event: Deposit user=%s token=%s amount=%s", user.Hex(), token.Hex(), amount.String())
}

func (h *noopHandler) OnWithdraw(user, token common.Address, amount *big.Int) {
	log.Printf("Event: Withdraw user=%s token=%s amount=%s", user.Hex(), token.Hex(), amount.String())
}

func (h *noopHandler) OnTradeSettled(buyHash, sellHash common.Hash, buyer, seller common.Address, baseAmount, quoteAmount *big.Int) {
	log.Printf("Event: TradeSettled buy=%s sell=%s", buyHash.Hex(), sellHash.Hex())
}
