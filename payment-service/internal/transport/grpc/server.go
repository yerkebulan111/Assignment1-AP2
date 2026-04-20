package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"payment-service/internal/usecase"

	pb "github.com/yerkebulan111/ap-2_protos-gen/payment"
)

type PaymentGRPCServer struct {
	pb.UnimplementedPaymentServiceServer
	uc usecase.PaymentUseCase
}

func NewPaymentGRPCServer(uc usecase.PaymentUseCase) *PaymentGRPCServer {
	return &PaymentGRPCServer{uc: uc}
}

func (s *PaymentGRPCServer) ProcessPayment(ctx context.Context, req *pb.PaymentRequest) (*pb.PaymentResponse, error) {

	output, err := s.uc.Authorize(usecase.AuthorizeInput{
		OrderID: req.OrderId,
		Amount:  req.Amount,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "authorize failed: %v", err)
	}

	return &pb.PaymentResponse{TransactionId: output.TransactionID, Status: output.Status}, nil
}

func (s *PaymentGRPCServer) ListPayments(ctx context.Context, req *pb.ListPaymentsRequest) (*pb.ListPaymentsResponse, error) {

	payments, err := s.uc.ListByAmountRange(req.MinAmount, req.MaxAmount)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "list payments failed: %v", err)
	}

	pbPayments := make([]*pb.PaymentResponse, 0, len(payments))
	for _, p := range payments {
		pbPayments = append(pbPayments, &pb.PaymentResponse{
			TransactionId: p.TransactionID,
			Status:        p.Status,
		})
	}

	return &pb.ListPaymentsResponse{
		Payments: pbPayments,
	}, nil
}
