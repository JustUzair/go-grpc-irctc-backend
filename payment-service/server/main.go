package main

import (
	"log"
	"net"

	"github.com/JustUzair/irctc-microservice/env"
	paymentv1 "github.com/JustUzair/irctc-microservice/gen/go/payment/v1"
	service "github.com/JustUzair/irctc-microservice/payment-service/server/internal"
	logger "github.com/JustUzair/irctc-microservice/utils/interceptors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	config, err := env.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	lis, err := net.Listen("tcp", ":"+config.PaymentServicePort)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(logger.UnaryServerLoggerInterceptor))
	paymentService := &service.PaymentService{}
	healthServer := health.NewServer()

	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus(paymentv1.PaymentService_ServiceDesc.ServiceName, healthpb.HealthCheckResponse_SERVING)

	paymentv1.RegisterPaymentServiceServer(grpcServer, paymentService)
	healthpb.RegisterHealthServer(grpcServer, healthServer)

	log.Printf("payment service started on port %s", config.PaymentServicePort)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Error occurred on gRPC server startup: %v", err)
	}
}
