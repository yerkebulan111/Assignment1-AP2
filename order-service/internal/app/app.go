package app

import (
	"database/sql"
	"fmt"
	"log"

	transporthttp "order-service/internal/transport/http"
	"order-service/internal/usecase"

	"github.com/gin-gonic/gin"
)

type Config struct {
	HTTPPort          string
	DBHost            string
	DBPort            string
	DBUser            string
	DBPassword        string
	DBName            string
	PaymentServiceURL string
}

type Server struct {
	cfg    Config
	router *gin.Engine
	db     *sql.DB
}

func NewServer(cfg Config, db *sql.DB, uc *usecase.OrderUseCase) *Server {
	router := gin.Default()

	handler := transporthttp.NewOrderHandler(uc)
	handler.RegisterRoutes(router)

	return &Server{
		cfg:    cfg,
		router: router,
		db:     db,
	}
}

func (s *Server) Run() error {
	addr := fmt.Sprintf(":%s", s.cfg.HTTPPort)
	log.Printf("Order Service listening on %s", addr)
	return s.router.Run(addr)
}
