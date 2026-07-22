package main

import (
	"log"
	"net"

	"github.com/JustUzair/irctc-microservice/env"
	userv1 "github.com/JustUzair/irctc-microservice/gen/go/user/v1"
	"github.com/JustUzair/irctc-microservice/user-service/server/services"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	config, err := env.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	lis, err := net.Listen("tcp", ":"+config.UserServicePort)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	userService := &services.UserService{}

	healthServer := health.NewServer()

	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus(userv1.UserService_ServiceDesc.ServiceName, healthpb.HealthCheckResponse_SERVING)

	userv1.RegisterUserServiceServer(grpcServer, userService)
	healthpb.RegisterHealthServer(grpcServer, healthServer)

	log.Printf("user service started on port %s", config.UserServicePort)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Error occurred on gRPC server startup: %v", err)
	}
}
