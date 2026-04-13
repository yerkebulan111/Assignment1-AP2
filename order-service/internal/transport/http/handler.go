package http

import (
	"errors"
	"log"
	"net/http"

	"order-service/internal/domain"
	"order-service/internal/usecase"

	"github.com/gin-gonic/gin"
)

type OrderHandler struct {
	uc *usecase.OrderUseCase
}

func NewOrderHandler(uc *usecase.OrderUseCase) *OrderHandler {
	return &OrderHandler{uc: uc}
}

func (h *OrderHandler) RegisterRoutes(router *gin.Engine) {
	router.POST("/orders", h.CreateOrder)
	router.GET("/orders/:id", h.GetOrder)
	router.PATCH("/orders/:id/cancel", h.CancelOrder)
	router.GET("/orders", h.GetOrdersByCustomer)
}

type createOrderRequest struct {
	CustomerID string `json:"customer_id" binding:"required"`
	ItemName   string `json:"item_name"   binding:"required"`
	Amount     int64  `json:"amount"      binding:"required"`
}

type orderResponse struct {
	ID         string `json:"id"`
	CustomerID string `json:"customer_id"`
	ItemName   string `json:"item_name"`
	Amount     int64  `json:"amount"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at"`
}

type createOrderResponse struct {
	Order         orderResponse `json:"order"`
	PaymentStatus string        `json:"payment_status"`
}

// POST /orders
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var req createOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	idempotencyKey := c.GetHeader("Idempotency-Key")

	input := usecase.CreateOrderInput{
		CustomerID:     req.CustomerID,
		ItemName:       req.ItemName,
		Amount:         req.Amount,
		IdempotencyKey: idempotencyKey,
	}

	output, err := h.uc.CreateOrder(input)
	if err != nil {
		log.Printf("ERROR IN CREATE ORDER: %v", err)
		if isPaymentUnavailable(err) {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "payment service unavailable"})
			return
		}

		if isDomainValidationError(err) {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusCreated, createOrderResponse{
		Order:         toOrderResponse(output.Order),
		PaymentStatus: output.PaymentStatus,
	})
}

// GET /orders/:id
func (h *OrderHandler) GetOrder(c *gin.Context) {
	id := c.Param("id")

	order, err := h.uc.GetOrder(id)
	if err != nil {
		if errors.Is(err, domain.ErrOrderNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, toOrderResponse(order))
}

// PATCH /orders/:id/cancel
func (h *OrderHandler) CancelOrder(c *gin.Context) {
	id := c.Param("id")

	order, err := h.uc.CancelOrder(id)
	if err != nil {
		if errors.Is(err, domain.ErrOrderNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
			return
		}
		if errors.Is(err, domain.ErrCannotCancel) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, domain.ErrAlreadyCancelled) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, toOrderResponse(order))
}

func toOrderResponse(o *domain.Order) orderResponse {
	return orderResponse{
		ID:         o.ID,
		CustomerID: o.CustomerID,
		ItemName:   o.ItemName,
		Amount:     o.Amount,
		Status:     o.Status,
		CreatedAt:  o.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func isPaymentUnavailable(err error) bool {
	return containsString(err.Error(), "payment service unavailable")
}

func isDomainValidationError(err error) bool {
	return errors.Is(err, domain.ErrInvalidAmount) ||
		errors.Is(err, domain.ErrInvalidCustomer) ||
		errors.Is(err, domain.ErrInvalidItemName)
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// GET /orders?customer_id=some_user
func (h *OrderHandler) GetOrdersByCustomer(c *gin.Context) {
	customerID := c.Query("customer_id")
	if customerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "customer_id query parameter is required"})
		return
	}

	orders, err := h.uc.GetOrdersByCustomer(customerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	response := make([]orderResponse, 0, len(orders))
	for _, o := range orders {
		response = append(response, toOrderResponse(o))
	}
	c.JSON(http.StatusOK, response)
}
