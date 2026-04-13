package usecase

import (
	"errors"
	"fmt"
	"time"

	"order-service/internal/domain"

	"github.com/google/uuid"
)

type OrderUseCase struct {
	repo          domain.OrderRepository
	paymentClient domain.PaymentClient
}

func NewOrderUseCase(repo domain.OrderRepository, paymentClient domain.PaymentClient) *OrderUseCase {
	return &OrderUseCase{
		repo:          repo,
		paymentClient: paymentClient,
	}
}

type CreateOrderInput struct {
	CustomerID     string
	ItemName       string
	Amount         int64
	IdempotencyKey string
}

type CreateOrderOutput struct {
	Order         *domain.Order
	PaymentStatus string
}

func (uc *OrderUseCase) CreateOrder(input CreateOrderInput) (*CreateOrderOutput, error) {
	if input.IdempotencyKey != "" {
		existing, err := uc.repo.FindByIdempotencyKey(input.IdempotencyKey)
		if err == nil && existing != nil {
			return &CreateOrderOutput{Order: existing, PaymentStatus: existing.Status}, nil
		}
		if err != nil && !errors.Is(err, domain.ErrOrderNotFound) {
			return nil, fmt.Errorf("idempotency check failed: %w", err)
		}
	}

	order := &domain.Order{
		ID:         uuid.New().String(),
		CustomerID: input.CustomerID,
		ItemName:   input.ItemName,
		Amount:     input.Amount,
		Status:     domain.StatusPending,
		CreatedAt:  time.Now().UTC(),
	}

	if input.IdempotencyKey != "" {
		order.IdempotencyKey = input.IdempotencyKey
	}

	if err := order.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	if err := uc.repo.Save(order); err != nil {
		return nil, fmt.Errorf("failed to save order: %w", err)
	}

	result, err := uc.paymentClient.Authorize(order.ID, order.Amount)
	if err != nil {
		order.MarkFailed()
		_ = uc.repo.Update(order)
		return nil, fmt.Errorf("payment service unavailable: %w", err)
	}

	if result.Status == "Authorized" {
		order.MarkPaid()
	} else {
		order.MarkFailed()
	}

	if err := uc.repo.Update(order); err != nil {
		return nil, fmt.Errorf("failed to update order status: %w", err)
	}

	return &CreateOrderOutput{
		Order:         order,
		PaymentStatus: result.Status,
	}, nil
}

func (uc *OrderUseCase) GetOrder(id string) (*domain.Order, error) {
	order, err := uc.repo.FindByID(id)
	if err != nil {
		return nil, fmt.Errorf("order not found: %w", err)
	}
	return order, nil
}

func (uc *OrderUseCase) CancelOrder(id string) (*domain.Order, error) {
	order, err := uc.repo.FindByID(id)
	if err != nil {
		return nil, fmt.Errorf("order not found: %w", err)
	}

	if err := order.Cancel(); err != nil {
		return nil, err
	}

	if err := uc.repo.Update(order); err != nil {
		return nil, fmt.Errorf("failed to cancel order: %w", err)
	}

	return order, nil
}

// customer id
func (uc *OrderUseCase) GetOrdersByCustomer(customerID string) ([]*domain.Order, error) {
	if customerID == "" {
		return nil, fmt.Errorf("customer_id is required")
	}
	return uc.repo.FindByCustomerID(customerID)
}
