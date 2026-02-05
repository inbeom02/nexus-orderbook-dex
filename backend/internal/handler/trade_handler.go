package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/nexus-orderbook-dex/backend/internal/service"
)

type TradeHandler struct {
	svc *service.OrderService
}

func NewTradeHandler(svc *service.OrderService) *TradeHandler {
	return &TradeHandler{svc: svc}
}

func (h *TradeHandler) GetTrades(c *gin.Context) {
	pair := c.DefaultQuery("pair", "TKA-TKB")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	trades, err := h.svc.GetTrades(c.Request.Context(), pair, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, trades)
}
