package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"google.golang.org/grpc"

	"github.com/joho/godotenv"

	"payment-service/internal/app"
	"payment-service/internal/messaging"
	"payment-service/internal/repository"
	grpcdelivery "payment-service/internal/transport/grpc"
	"payment-service/internal/usecase"

	pb "github.com/yerkebulan111/ap-2_protos-gen/payment"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "postgres")
	dbName := getEnv("DB_NAME", "payment_db")
	serverPort := getEnv("SERVER_PORT", "8081")
	grpcPort := getEnv("GRPC_PORT", "50051")
	amqpURL := getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/")

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("failed to open database connection: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}
	log.Println("Connected to PostgreSQL successfully")

	// RabbitMQ publisher
	publisher, err := messaging.NewPublisher(amqpURL)
	if err != nil {
		log.Fatalf("failed to create RabbitMQ publisher: %v", err)
	}
	defer publisher.Close()

	repo := repository.NewPostgresPaymentRepository(db)
	uc := usecase.NewPaymentUseCase(repo, publisher)

	// --- HTTP server ---
	httpRouter := app.NewRouter(db, publisher)
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%s", serverPort),
		Handler: httpRouter,
	}

	go func() {
		log.Printf("Payment HTTP listening on :%s", serverPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// --- gRPC server ---
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", grpcPort))
	if err != nil {
		log.Fatalf("failed to listen on gRPC port: %v", err)
	}
	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(grpcdelivery.LoggingInterceptor))
	pb.RegisterPaymentServiceServer(grpcServer, grpcdelivery.NewPaymentGRPCServer(uc))

	go func() {
		log.Printf("Payment gRPC listening on :%s", grpcPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("gRPC server error: %v", err)
		}
	}()

	// --- Graceful shutdown ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("[main] Shutdown signal received...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("[main] HTTP shutdown error: %v", err)
	}
	grpcServer.GracefulStop()
	log.Println("[main] Payment Service stopped gracefully.")
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
