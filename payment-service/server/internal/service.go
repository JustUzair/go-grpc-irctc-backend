package service

import (
	"context"

	paymentv1 "github.com/JustUzair/irctc-microservice/gen/go/payment/v1"
)

type PaymentService struct {
	paymentv1.UnimplementedPaymentServiceServer
}

func (*PaymentService) GetPayment(context.Context, *paymentv1.GetPaymentRequest) (*paymentv1.GetPaymentResponse, error) {
	return &paymentv1.GetPaymentResponse{
		Payment: &paymentv1.Payment{Id: "payment"},
	}, nil
}
