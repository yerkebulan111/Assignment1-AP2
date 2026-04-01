package app

import (
	"database/sql"

	"github.com/gin-gonic/gin"

	"payment-service/internal/repository"
	httpdelivery "payment-service/internal/transport/http"
	"payment-service/internal/usecase"
)

func NewRouter(db *sql.DB) *gin.Engine {

	repo := repository.NewPostgresPaymentRepository(db)
	uc := usecase.NewPaymentUseCase(repo)
	handler := httpdelivery.NewPaymentHandler(uc)

	router := gin.Default()
	handler.RegisterRoutes(router)

	return router
}
