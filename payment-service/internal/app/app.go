package app

import (
	"database/sql"

	"github.com/gin-gonic/gin"

	"payment-service/internal/messaging"
	"payment-service/internal/repository"
	httpdelivery "payment-service/internal/transport/http"
	"payment-service/internal/usecase"
)

// NewRouter now accepts the publisher so the HTTP handler's use-case can publish events.
func NewRouter(db *sql.DB, publisher *messaging.Publisher) *gin.Engine {
	repo := repository.NewPostgresPaymentRepository(db)
	uc := usecase.NewPaymentUseCase(repo, publisher)
	handler := httpdelivery.NewPaymentHandler(uc)
	router := gin.Default()
	handler.RegisterRoutes(router)
	return router
}
