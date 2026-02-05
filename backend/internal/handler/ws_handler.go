package handler

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	redisRepo "github.com/nexus-orderbook-dex/backend/internal/repository/redis"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type WSHandler struct {
	cache *redisRepo.OrderbookCache
}

func NewWSHandler(cache *redisRepo.OrderbookCache) *WSHandler {
	return &WSHandler{cache: cache}
}

func (h *WSHandler) Handle(c *gin.Context) {
	pair := c.DefaultQuery("pair", "TKA-TKB")

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	// Send initial snapshot
	bids, asks, err := h.cache.GetSnapshot(c.Request.Context(), pair)
	if err == nil {
		conn.WriteJSON(map[string]interface{}{
			"type": "snapshot",
			"bids": bids,
			"asks": asks,
		})
	}

	// Subscribe to Redis pub/sub for updates
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	sub := h.cache.Subscribe(ctx, pair)
	defer sub.Close()
	ch := sub.Channel()

	// Read pump (detect disconnection)
	var once sync.Once
	go func() {
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				once.Do(cancel)
				return
			}
		}
	}()

	// Ping ticker
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			if err := conn.WriteMessage(websocket.TextMessage, []byte(msg.Payload)); err != nil {
				return
			}
		case <-ticker.C:
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
