package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"order-service/internal/repository"
	grpcclient "order-service/internal/transport/grpc"
	"order-service/internal/usecase"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"

	transporthttp "order-service/internal/transport/http"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	httpPort := getEnv("HTTP_PORT", "8080")
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "postgres")
	dbName := getEnv("DB_NAME", "orders_db")
	paymentGRPCAddr := getEnv("PAYMENT_GRPC_ADDR", "localhost:50051")

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName,
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
	paymentClient, err := grpcclient.NewPaymentGRPCClient(paymentGRPCAddr)
	if err != nil {
		log.Fatalf("failed to connect to payment gRPC: %v", err)
	}

	orderUseCase := usecase.NewOrderUseCase(orderRepo, paymentClient)

	router := gin.Default()
	handler := transporthttp.NewOrderHandler(orderUseCase)
	handler.RegisterRoutes(router)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", httpPort),
		Handler: router,
	}

	go func() {
		log.Printf("Order Service listening on :%s", httpPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Order service HTTP error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("[main] Shutdown signal received...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("[main] Order service shutdown error: %v", err)
	}
	log.Println("[main] Order Service stopped gracefully.")
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}
