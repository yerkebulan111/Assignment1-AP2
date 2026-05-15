package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"notification-service/internal/messaging"
	"notification-service/internal/repository"
	"notification-service/internal/usecase"
)

func main() {
	amqpURL := getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")

	idempotencyStore := repository.NewInMemoryIdempotencyStore()
	notificationUC := usecase.NewNotificationUseCase(idempotencyStore)

	consumer, err := messaging.NewConsumer(amqpURL, notificationUC)
	if err != nil {
		log.Fatalf("[main] Failed to create consumer: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		log.Printf("[main] Received signal %s, shutting down...", sig)
		cancel()
	}()

	log.Println("[main] Notification Service started.")
	if err := consumer.Consume(ctx); err != nil {
		log.Printf("[main] Consumer stopped with error: %v", err)
	}

	consumer.Close()
	log.Println("[main] Notification Service stopped.")
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
