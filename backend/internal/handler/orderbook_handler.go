package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nexus-orderbook-dex/backend/internal/service"
)

type OrderbookHandler struct {
	svc *service.OrderService
}

func NewOrderbookHandler(svc *service.OrderService) *OrderbookHandler {
	return &OrderbookHandler{svc: svc}
}

func (h *OrderbookHandler) GetOrderbook(c *gin.Context) {
	pair := c.DefaultQuery("pair", "TKA-TKB")
	snapshot := h.svc.GetOrderbook(pair)
	c.JSON(http.StatusOK, snapshot)
}
