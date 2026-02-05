package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nexus-orderbook-dex/backend/internal/domain"
	"github.com/nexus-orderbook-dex/backend/internal/service"
)

type OrderHandler struct {
	svc *service.OrderService
}

func NewOrderHandler(svc *service.OrderService) *OrderHandler {
	return &OrderHandler{svc: svc}
}

func (h *OrderHandler) SubmitOrder(c *gin.Context) {
	var sub domain.OrderSubmission
	if err := c.ShouldBindJSON(&sub); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order, matches, err := h.svc.SubmitOrder(c.Request.Context(), sub)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"order":   order,
		"matches": len(matches),
	})
}

func (h *OrderHandler) GetUserOrders(c *gin.Context) {
	address := c.Param("address")
	orders, err := h.svc.GetOrdersByMaker(c.Request.Context(), address)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, orders)
}

func (h *OrderHandler) CancelOrder(c *gin.Context) {
	id := c.Param("id")
	if err := h.svc.CancelOrder(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "cancelled"})
}
