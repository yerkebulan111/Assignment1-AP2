package main

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"

	_ "github.com/lib/pq"
	"google.golang.org/grpc"

	"github.com/joho/godotenv"
	_ "github.com/joho/godotenv"
	"payment-service/internal/app"
	"payment-service/internal/repository"
	grpcdelivery "payment-service/internal/transport/grpc"
	"payment-service/internal/usecase"

	pb "github.com/yerkebulan111/ap-2_protos-gen/payment"
)

func main() {
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "password")
	dbName := getEnv("DB_NAME", "payment_db")
	serverPort := getEnv("SERVER_PORT", "8081")
	grpcPort := getEnv("GRPC_PORT", "50051")

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


	repo := repository.NewPostgresPaymentRepository(db)
	uc := usecase.NewPaymentUseCase(repo)

	go func() {
		router := app.NewRouter(db)
		addr := fmt.Sprintf(":%s", serverPort)
		log.Printf("Payment HTTP listening on %s", addr)
		if err := router.Run(addr); err != nil {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", grpcPort))
	if err != nil {
		log.Fatalf("failed to listen on gRPC port: %v", err)
	}

	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(grpcdelivery.LoggingInterceptor))

	pb.RegisterPaymentServiceServer(grpcServer, grpcdelivery.NewPaymentGRPCServer(uc))

	log.Printf("Payment gRPC listening on :%s", grpcPort)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("gRPC server error: %v", err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
