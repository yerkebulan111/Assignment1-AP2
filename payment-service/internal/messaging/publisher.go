package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	QueueName = "payment.completed"
)

// PaymentEvent is the message payload sent to the broker.
// It must stay self-contained — no coupling to Order or Notification services.
type PaymentEvent struct {
	EventID       string    `json:"event_id"`
	OrderID       string    `json:"order_id"`
	Amount        int64     `json:"amount"`
	CustomerEmail string    `json:"customer_email"`
	Status        string    `json:"status"`
	OccurredAt    time.Time `json:"occurred_at"`
}

type Publisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

func NewPublisher(amqpURL string) (*Publisher, error) {
	conn, err := dialWithRetry(amqpURL, 10, 3*time.Second)
	if err != nil {
		return nil, fmt.Errorf("publisher: failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("publisher: failed to open channel: %w", err)
	}

	if _, err := ch.QueueDeclare(
		QueueName,
		true,
		false,
		false,
		false,
		amqp.Table{
			"x-dead-letter-exchange":    "payment.dlx",
			"x-dead-letter-routing-key": "payment.completed.dlq",
		},
	); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("publisher: declare queue: %w", err)
	}

	return &Publisher{conn: conn, channel: ch}, nil
}

func (p *Publisher) Publish(ctx context.Context, event PaymentEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("publisher: marshal event: %w", err)
	}

	err = p.channel.PublishWithContext(
		ctx,
		"",
		QueueName,
		true,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			MessageId:    event.EventID,
			Body:         body,
		},
	)
	if err != nil {
		return fmt.Errorf("publisher: publish failed: %w", err)
	}

	log.Printf("[Publisher] Event published: order_id=%s event_id=%s status=%s",
		event.OrderID, event.EventID, event.Status)
	return nil
}

func (p *Publisher) Close() {
	if p.channel != nil {
		_ = p.channel.Close()
	}
	if p.conn != nil {
		_ = p.conn.Close()
	}
}

func dialWithRetry(url string, attempts int, delay time.Duration) (*amqp.Connection, error) {
	var err error
	for i := 0; i < attempts; i++ {
		conn, dialErr := amqp.Dial(url)
		if dialErr == nil {
			return conn, nil
		}
		err = dialErr
		log.Printf("[Publisher] RabbitMQ not ready, retrying in %s (%d/%d): %v",
			delay, i+1, attempts, dialErr)
		time.Sleep(delay)
	}
	return nil, err
}
