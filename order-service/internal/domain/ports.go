package domain

type OrderRepository interface {
	Save(order *Order) error
	FindByID(id string) (*Order, error)
	Update(order *Order) error
	FindByIdempotencyKey(key string) (*Order, error)
}

type PaymentResult struct {
	TransactionID string
	Status        string
}

type PaymentClient interface {
	Authorize(orderID string, amount int64) (*PaymentResult, error)
}
