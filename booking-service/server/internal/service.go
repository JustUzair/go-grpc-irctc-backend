package service

import (
	"context"

	bookingv1 "github.com/JustUzair/go-grpc-irctc-backend/gen/go/booking/v1"
)

type BookingService struct {
	bookingv1.UnimplementedBookingServiceServer
}

func (*BookingService) GetBooking(context.Context, *bookingv1.GetBookingRequest) (*bookingv1.GetBookingResponse, error) {
	return &bookingv1.GetBookingResponse{
		Booking: &bookingv1.Booking{Id: "booking"},
	}, nil
}
