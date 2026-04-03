package domain

import (
	"errors"
	"time"
)

const (
	StatusPending   = "Pending"
	StatusPaid      = "Paid"
	StatusFailed    = "Failed"
	StatusCancelled = "Cancelled"
)

type Order struct {
	ID             string
	CustomerID     string
	ItemName       string
	Amount         int64
	Status         string
	CreatedAt      time.Time
	IdempotencyKey string
}

var (
	ErrInvalidAmount    = errors.New("amount must be greater than 0")
	ErrInvalidCustomer  = errors.New("customer_id must not be empty")
	ErrInvalidItemName  = errors.New("item_name must not be empty")
	ErrOrderNotFound    = errors.New("order not found")
	ErrCannotCancel     = errors.New("only Pending orders can be cancelled")
	ErrAlreadyCancelled = errors.New("order is already cancelled")
)

func (o *Order) Validate() error {
	if o.Amount <= 0 {
		return ErrInvalidAmount
	}
	if o.CustomerID == "" {
		return ErrInvalidCustomer
	}
	if o.ItemName == "" {
		return ErrInvalidItemName
	}
	return nil
}

func (o *Order) Cancel() error {
	if o.Status == StatusPaid {
		return ErrCannotCancel
	}
	if o.Status == StatusCancelled {
		return ErrAlreadyCancelled
	}
	if o.Status != StatusPending {
		return ErrCannotCancel
	}
	o.Status = StatusCancelled
	return nil
}

func (o *Order) MarkPaid() {
	o.Status = StatusPaid
}

func (o *Order) MarkFailed() {
	o.Status = StatusFailed
}
