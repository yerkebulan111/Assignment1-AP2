package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/lib/pq"

	"payment-service/internal/domain"
)

type postgresPaymentRepository struct {
	db *sql.DB
}

func NewPostgresPaymentRepository(db *sql.DB) domain.PaymentRepository {
	return &postgresPaymentRepository{db: db}
}

func (r *postgresPaymentRepository) Save(p *domain.Payment) error {
	query := `
		INSERT INTO payments (id, order_id, transaction_id, amount, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.Exec(query,
		p.ID,
		p.OrderID,
		p.TransactionID,
		p.Amount,
		p.Status,
		p.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("repository save error: %w", err)
	}
	return nil
}

func (r *postgresPaymentRepository) FindByOrderID(orderID string) (*domain.Payment, error) {
	query := `
		SELECT id, order_id, transaction_id, amount, status, created_at
		FROM payments
		WHERE order_id = $1
		LIMIT 1
	`

	row := r.db.QueryRow(query, orderID)

	var p domain.Payment
	var transactionID sql.NullString
	var createdAt time.Time

	err := row.Scan(
		&p.ID,
		&p.OrderID,
		&transactionID,
		&p.Amount,
		&p.Status,
		&createdAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("no payment found for order_id %s", orderID)
		}
		return nil, fmt.Errorf("repository query error: %w", err)
	}

	p.TransactionID = transactionID.String
	p.CreatedAt = createdAt
	return &p, nil
}
