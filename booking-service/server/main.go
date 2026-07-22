package main

import (
	"log"
	"net"

	"github.com/JustUzair/irctc-microservice/booking-service/server/services"
	"github.com/JustUzair/irctc-microservice/env"
	bookingv1 "github.com/JustUzair/irctc-microservice/gen/go/booking/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	config, err := env.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	lis, err := net.Listen("tcp", ":"+config.BookingServicePort)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	bookingService := &services.BookingService{}
	healthServer := health.NewServer()

	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus(bookingv1.BookingService_ServiceDesc.ServiceName, healthpb.HealthCheckResponse_SERVING)

	bookingv1.RegisterBookingServiceServer(grpcServer, bookingService)
	healthpb.RegisterHealthServer(grpcServer, healthServer)

	log.Printf("gRPC server serving on :%s", config.BookingServicePort)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Error occurred on gRPC server startup: %v", err)
	}
}
