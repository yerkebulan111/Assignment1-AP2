package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/yerkebulan111/ap-2_protos-gen/payment"
	"payment-service/internal/usecase"
)

type PaymentGRPCServer struct {
	pb.UnimplementedPaymentServiceServer
	uc usecase.PaymentUseCase
}

func NewPaymentGRPCServer(uc usecase.PaymentUseCase) *PaymentGRPCServer {
	return &PaymentGRPCServer{uc: uc}
}

func (s *PaymentGRPCServer) ProcessPayment(
	ctx context.Context,
	req *pb.PaymentRequest,
) (*pb.PaymentResponse, error) {

	output, err := s.uc.Authorize(usecase.AuthorizeInput{
		OrderID: req.OrderId,
		Amount:  req.Amount,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "authorize failed: %v", err)
	}

	return &pb.PaymentResponse{
		TransactionId: output.TransactionID,
		Status:        output.Status,
	}, nil
}
