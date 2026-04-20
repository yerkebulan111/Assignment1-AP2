package grpc

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/yerkebulan111/ap-2_protos-gen/payment"
	"order-service/internal/domain"
)

type PaymentGRPCClient struct {
	client pb.PaymentServiceClient
}

func NewPaymentGRPCClient(addr string) (domain.PaymentClient, error) {
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to dial payment service: %w", err)
	}

	return &PaymentGRPCClient{client: pb.NewPaymentServiceClient(conn)}, nil
}

func (c *PaymentGRPCClient) Authorize(orderID string, amount int64) (*domain.PaymentResult, error) {
	resp, err := c.client.ProcessPayment(context.Background(), &pb.PaymentRequest{
		OrderId: orderID,
		Amount:  amount,
	})
	if err != nil {
		return nil, fmt.Errorf("payment service unavailable: %w", err)
	}

	return &domain.PaymentResult{TransactionID: resp.TransactionId, Status: resp.Status}, nil
}
