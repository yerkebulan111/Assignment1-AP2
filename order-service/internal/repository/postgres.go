package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"order-service/internal/domain"
)

type postgresOrderRepository struct {
	db *sql.DB
}

func NewPostgresOrderRepository(db *sql.DB) domain.OrderRepository {
	return &postgresOrderRepository{db: db}
}

func (r *postgresOrderRepository) Save(order *domain.Order) error {
	query := `
		INSERT INTO orders (id, customer_id, item_name, amount, status, created_at, idempotency_key)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.db.Exec(query,
		order.ID,
		order.CustomerID,
		order.ItemName,
		order.Amount,
		order.Status,
		order.CreatedAt,
		order.IdempotencyKey,
	)
	if err != nil {
		return fmt.Errorf("postgresOrderRepository.Save: %w", err)
	}
	return nil
}

func (r *postgresOrderRepository) FindByID(id string) (*domain.Order, error) {
	query := `
		SELECT id, customer_id, item_name, amount, status, created_at
		FROM orders WHERE id = $1
	`
	row := r.db.QueryRow(query, id)

	order := &domain.Order{}
	var createdAt time.Time

	err := row.Scan(
		&order.ID,
		&order.CustomerID,
		&order.ItemName,
		&order.Amount,
		&order.Status,
		&createdAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrOrderNotFound
		}
		return nil, fmt.Errorf("postgresOrderRepository.FindByID: %w", err)
	}

	order.CreatedAt = createdAt
	return order, nil
}

func (r *postgresOrderRepository) Update(order *domain.Order) error {
	query := `UPDATE orders SET status = $1 WHERE id = $2`
	result, err := r.db.Exec(query, order.Status, order.ID)
	if err != nil {
		return fmt.Errorf("postgresOrderRepository.Update: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("postgresOrderRepository.Update rows affected: %w", err)
	}
	if rows == 0 {
		return domain.ErrOrderNotFound
	}
	return nil
}

func (r *postgresOrderRepository) FindByIdempotencyKey(key string) (*domain.Order, error) {
	if key == "" {
		return nil, errors.New("empty idempotency key")
	}
	query := `
		SELECT id, customer_id, item_name, amount, status, created_at
		FROM orders WHERE idempotency_key = $1
	`
	row := r.db.QueryRow(query, key)

	order := &domain.Order{}
	var createdAt time.Time

	err := row.Scan(
		&order.ID,
		&order.CustomerID,
		&order.ItemName,
		&order.Amount,
		&order.Status,
		&createdAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrOrderNotFound
		}
		return nil, fmt.Errorf("postgresOrderRepository.FindByIdempotencyKey: %w", err)
	}
	order.CreatedAt = createdAt
	return order, nil
}
