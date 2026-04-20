package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"order-service/internal/app"
	"order-service/internal/repository"
	grpcclient "order-service/internal/transport/grpc"
	"order-service/internal/usecase"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found")
	}

	cfg := app.Config{
		HTTPPort:        getEnv("HTTP_PORT", "8080"),
		DBHost:          getEnv("DB_HOST", "localhost"),
		DBPort:          getEnv("DB_PORT", "5432"),
		DBUser:          getEnv("DB_USER", "postgres"),
		DBPassword:      getEnv("DB_PASSWORD", "postgres"),
		DBName:          getEnv("DB_NAME", "orders_db"),
		PaymentGRPCAddr: getEnv("PAYMENT_GRPC_ADDR", "localhost:50051"),
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

	orderRepo := repository.NewPostgresOrderRepository(db)
	paymentClient, err := grpcclient.NewPaymentGRPCClient(cfg.PaymentGRPCAddr)
	if err != nil {
		log.Fatalf("failed to connect to payment gRPC: %v", err)
	}

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
