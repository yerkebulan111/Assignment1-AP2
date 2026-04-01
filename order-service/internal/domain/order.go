package domain

import (
	"errors"
	"time"
)

// Order statuses
const (
	StatusPending   = "Pending"
	StatusPaid      = "Paid"
	StatusFailed    = "Failed"
	StatusCancelled = "Cancelled"
)

// Order is the core domain entity. It must not depend on HTTP, JSON, or any framework.
type Order struct {
	ID         string
	CustomerID string
	ItemName   string
	Amount     int64 // Amount in cents (e.g., 1000 = $10.00). Never float64.
	Status     string
	CreatedAt  time.Time
}

// Validate enforces Order invariants.
func (o *Order) Validate() error {
	if o.CustomerID == "" {
		return errors.New("customer_id is required")
	}
	if o.ItemName == "" {
		return errors.New("item_name is required")
	}
	if o.Amount <= 0 {
		return errors.New("amount must be greater than 0")
	}
	return nil
}

// MarkPaid transitions order to Paid status.
func (o *Order) MarkPaid() error {
	if o.Status != StatusPending {
		return errors.New("only pending orders can be marked as paid")
	}
	o.Status = StatusPaid
	return nil
}

// MarkFailed transitions order to Failed status.
func (o *Order) MarkFailed() error {
	if o.Status != StatusPending {
		return errors.New("only pending orders can be marked as failed")
	}
	o.Status = StatusFailed
	return nil
}

// Cancel transitions order to Cancelled status.
func (o *Order) Cancel() error {
	if o.Status == StatusPaid {
		return errors.New("paid orders cannot be cancelled")
	}
	if o.Status != StatusPending {
		return errors.New("only pending orders can be cancelled")
	}
	o.Status = StatusCancelled
	return nil
}
