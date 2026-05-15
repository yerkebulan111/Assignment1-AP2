package usecase

import (
	"fmt"
	"log"

	"notification-service/internal/domain"
	"notification-service/internal/repository"
)

type NotificationUseCase struct {
	idempotency repository.IdempotencyStore
}

func NewNotificationUseCase(store repository.IdempotencyStore) *NotificationUseCase {
	return &NotificationUseCase{idempotency: store}
}

func (uc *NotificationUseCase) Handle(event domain.PaymentEvent) (bool, error) {
	if event.EventID == "" {
		return false, fmt.Errorf("event_id is required for idempotency check")
	}

	if uc.idempotency.IsProcessed(event.EventID) {
		log.Printf("[Notification] Duplicate event detected, skipping. event_id=%s order_id=%s",
			event.EventID, event.OrderID)
		return true, nil
	}

	log.Printf("[Notification] Sent email to %s for Order #%s. Amount: $%.2f Status: %s",
		event.CustomerEmail,
		event.OrderID,
		float64(event.Amount)/100.0,
		event.Status,
	)

	uc.idempotency.MarkProcessed(event.EventID)
	return false, nil
}
