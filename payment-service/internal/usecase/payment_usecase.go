package usecase

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"payment-service/internal/domain"
)

type AuthorizeInput struct {
	OrderID string
	Amount  int64
}

type AuthorizeOutput struct {
	PaymentID     string
	OrderID       string
	TransactionID string
	Amount        int64
	Status        string
}

type GetByOrderIDOutput struct {
	PaymentID     string
	OrderID       string
	TransactionID string
	Amount        int64
	Status        string
	CreatedAt     time.Time
}

type PaymentUseCase interface {
	Authorize(input AuthorizeInput) (*AuthorizeOutput, error)
	GetByOrderID(orderID string) (*GetByOrderIDOutput, error)
	ListByAmountRange(min, max int64) ([]*AuthorizeOutput, error)
}

type paymentUseCase struct {
	repo domain.PaymentRepository
}

func NewPaymentUseCase(repo domain.PaymentRepository) PaymentUseCase {
	return &paymentUseCase{repo: repo}
}

func (uc *paymentUseCase) Authorize(input AuthorizeInput) (*AuthorizeOutput, error) {
	p := &domain.Payment{
		ID:        uuid.NewString(),
		OrderID:   input.OrderID,
		Amount:    input.Amount,
		CreatedAt: time.Now().UTC(),
	}

	if err := p.Validate(); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	if domain.IsDeclined(input.Amount) {
		p.TransactionID = ""
		p.Status = domain.StatusDeclined
	} else {
		p.TransactionID = uuid.NewString()
		p.Status = domain.StatusAuthorized
	}

	if err := uc.repo.Save(p); err != nil {
		return nil, fmt.Errorf("failed to persist payment: %w", err)
	}

	return &AuthorizeOutput{
		PaymentID:     p.ID,
		OrderID:       p.OrderID,
		TransactionID: p.TransactionID,
		Amount:        p.Amount,
		Status:        p.Status,
	}, nil
}

func (uc *paymentUseCase) GetByOrderID(orderID string) (*GetByOrderIDOutput, error) {
	p, err := uc.repo.FindByOrderID(orderID)
	if err != nil {
		return nil, fmt.Errorf("payment not found: %w", err)
	}

	return &GetByOrderIDOutput{
		PaymentID:     p.ID,
		OrderID:       p.OrderID,
		TransactionID: p.TransactionID,
		Amount:        p.Amount,
		Status:        p.Status,
		CreatedAt:     p.CreatedAt,
	}, nil
}

func (uc *paymentUseCase) ListByAmountRange(min, max int64) ([]*AuthorizeOutput, error) {
	if min > 0 && max > 0 && min > max {
		return nil, fmt.Errorf("min_amount cannot be greater than max_amount")
	}

	payments, err := uc.repo.FindByAmountRange(min, max)
	if err != nil {
		return nil, fmt.Errorf("failed to list payments: %w", err)
	}

	result := make([]*AuthorizeOutput, 0, len(payments))
	for _, p := range payments {
		result = append(result, &AuthorizeOutput{
			PaymentID:     p.ID,
			OrderID:       p.OrderID,
			TransactionID: p.TransactionID,
			Amount:        p.Amount,
			Status:        p.Status,
		})
	}

	return result, nil
}
