package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"payment-service/internal/usecase"
)

type PaymentHandler struct {
	uc usecase.PaymentUseCase
}

func NewPaymentHandler(uc usecase.PaymentUseCase) *PaymentHandler {
	return &PaymentHandler{uc: uc}
}

func (h *PaymentHandler) RegisterRoutes(router *gin.Engine) {
	router.POST("/payments", h.Authorize)
	router.GET("/payments/:order_id", h.GetByOrderID)
}

type authorizeRequest struct {
	OrderID string `json:"order_id" binding:"required"`
	Amount  int64  `json:"amount"   binding:"required,gt=0"`
}

func (h *PaymentHandler) Authorize(c *gin.Context) {
	var req authorizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	output, err := h.uc.Authorize(usecase.AuthorizeInput{
		OrderID: req.OrderID,
		Amount:  req.Amount,
	})
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"payment_id":     output.PaymentID,
		"order_id":       output.OrderID,
		"transaction_id": output.TransactionID,
		"amount":         output.Amount,
		"status":         output.Status,
	})
}

func (h *PaymentHandler) GetByOrderID(c *gin.Context) {
	orderID := c.Param("order_id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "order_id is required"})
		return
	}

	output, err := h.uc.GetByOrderID(orderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"payment_id":     output.PaymentID,
		"order_id":       output.OrderID,
		"transaction_id": output.TransactionID,
		"amount":         output.Amount,
		"status":         output.Status,
		"created_at":     output.CreatedAt,
	})
}
