package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"order-service/internal/app"
	"order-service/internal/repository"
	transportHttp "order-service/internal/transport/http"
	"order-service/internal/usecase"

	"github.com/joho/godotenv"
	_ "github.com/joho/godotenv"

	_ "github.com/lib/pq"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found")
	}

	cfg := app.Config{
		HTTPPort:          getEnv("HTTP_PORT", "8080"),
		DBHost:            getEnv("DB_HOST", "localhost"),
		DBPort:            getEnv("DB_PORT", "5432"),
		DBUser:            getEnv("DB_USER", "postgres"),
		DBPassword:        getEnv("DB_PASSWORD", "postgres"),
		DBName:            getEnv("DB_NAME", "orders_db"),
		PaymentServiceURL: getEnv("PAYMENT_SERVICE_URL", "http://localhost:8081"),
	}

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}
	log.Println("Connected to PostgreSQL successfully")

	log.Println("connected to PostgreSQL")

	orderRepo := repository.NewPostgresOrderRepository(db)
	httpClient := &http.Client{Timeout: 2 * time.Second}
	paymentClient := transportHttp.NewPaymentHTTPClient(httpClient, cfg.PaymentServiceURL)
	orderUseCase := usecase.NewOrderUseCase(orderRepo, paymentClient)
	server := app.NewServer(cfg, db, orderUseCase)
	if err := server.Run(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}
