package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"notification-service/internal/domain"
	"notification-service/internal/usecase"
)

const (
	QueueName    = "payment.completed"
	DLXName      = "payment.dlx"           // Dead Letter Exchange
	DLQName      = "payment.completed.dlq" // Dead Letter Queue
	MaxRetries   = 3
	ExchangeName = ""
)

type Consumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	uc      *usecase.NotificationUseCase
}

func NewConsumer(amqpURL string, uc *usecase.NotificationUseCase) (*Consumer, error) {
	conn, err := dialWithRetry(amqpURL, 10, 3*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	if err := declareInfrastructure(ch); err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}

	return &Consumer{conn: conn, channel: ch, uc: uc}, nil
}

func declareInfrastructure(ch *amqp.Channel) error {
	if err := ch.ExchangeDeclare(
		DLXName, "direct", true, false, false, false, nil,
	); err != nil {
		return fmt.Errorf("declare DLX: %w", err)
	}

	if _, err := ch.QueueDeclare(
		DLQName, true, false, false, false, nil,
	); err != nil {
		return fmt.Errorf("declare DLQ: %w", err)
	}

	if err := ch.QueueBind(DLQName, DLQName, DLXName, false, nil); err != nil {
		return fmt.Errorf("bind DLQ: %w", err)
	}

	args := amqp.Table{
		"x-dead-letter-exchange":    DLXName,
		"x-dead-letter-routing-key": DLQName,
	}
	if _, err := ch.QueueDeclare(
		QueueName, true, false, false, false, args,
	); err != nil {
		return fmt.Errorf("declare main queue: %w", err)
	}

	if err := ch.Qos(1, 0, false); err != nil {
		return fmt.Errorf("set QoS: %w", err)
	}

	return nil
}

func (c *Consumer) Consume(ctx context.Context) error {
	msgs, err := c.channel.Consume(
		QueueName,
		"notification-consumer",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	log.Printf("[Consumer] Listening on queue: %s", QueueName)

	for {
		select {
		case <-ctx.Done():
			log.Println("[Consumer] Context cancelled, shutting down.")
			return nil

		case msg, ok := <-msgs:
			if !ok {
				log.Println("[Consumer] Channel closed.")
				return fmt.Errorf("channel closed unexpectedly")
			}
			c.handleMessage(msg)
		}
	}
}

func (c *Consumer) handleMessage(msg amqp.Delivery) {
	retryCount := getRetryCount(msg)

	var event domain.PaymentEvent
	if err := json.Unmarshal(msg.Body, &event); err != nil {
		log.Printf("[Consumer] Failed to unmarshal message: %v. Sending to DLQ.", err)
		_ = msg.Reject(false)
		return
	}

	duplicate, err := c.uc.Handle(event)
	if err != nil {
		log.Printf("[Consumer] Error handling event %s: %v (retry %d/%d)",
			event.EventID, err, retryCount+1, MaxRetries)

		if retryCount >= MaxRetries-1 {
			log.Printf("[Consumer] Max retries reached for event %s. Moving to DLQ.", event.EventID)
			_ = msg.Reject(false)
			return
		}

		_ = msg.Nack(false, true)
		return
	}

	if duplicate {
		_ = msg.Ack(false)
		return
	}

	_ = msg.Ack(false)
}

func getRetryCount(msg amqp.Delivery) int64 {
	deaths, ok := msg.Headers["x-death"]
	if !ok {
		return 0
	}
	table, ok := deaths.([]interface{})
	if !ok || len(table) == 0 {
		return 0
	}
	entry, ok := table[0].(amqp.Table)
	if !ok {
		return 0
	}
	count, _ := entry["count"].(int64)
	return count
}

func (c *Consumer) Close() {
	if c.channel != nil {
		_ = c.channel.Close()
	}
	if c.conn != nil {
		_ = c.conn.Close()
	}
	log.Println("[Consumer] RabbitMQ connection closed.")
}

func dialWithRetry(url string, attempts int, delay time.Duration) (*amqp.Connection, error) {
	var err error
	for i := 0; i < attempts; i++ {
		conn, dialErr := amqp.Dial(url)
		if dialErr == nil {
			return conn, nil
		}
		err = dialErr
		log.Printf("[Consumer] RabbitMQ not ready, retrying in %s (%d/%d): %v",
			delay, i+1, attempts, dialErr)
		time.Sleep(delay)
	}
	return nil, err
}
