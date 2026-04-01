package domain

import "context"

type OrderRepository interface {
	Save(ctx context.Context, order *Order) error
	FindByID(ctx context.Context, id string) (*Order, error)
	Update(ctx context.Context, order *Order) error
}

type PaymentResponse struct {
	TransactionID string
	Status        string
}

type PaymentClient interface {
	AuthorizePayment(ctx context.Context, orderID string, amount int64) (*PaymentResponse, error)
}
