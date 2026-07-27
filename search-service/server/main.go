package main

import (
	"log"
	"net"

	searchv1 "github.com/JustUzair/go-grpc-irctc-backend/gen/go/search/v1"
	service "github.com/JustUzair/go-grpc-irctc-backend/search-service/server/internal"
	env "github.com/JustUzair/go-grpc-irctc-backend/utils/env"
	logger "github.com/JustUzair/go-grpc-irctc-backend/utils/interceptors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	config, err := env.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	lis, err := net.Listen("tcp", ":"+config.SearchServicePort)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(logger.UnaryServerLoggerInterceptor))
	searchService := &service.SearchService{}
	healthServer := health.NewServer()

	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus(searchv1.SearchService_ServiceDesc.ServiceName, healthpb.HealthCheckResponse_SERVING)

	searchv1.RegisterSearchServiceServer(grpcServer, searchService)
	healthpb.RegisterHealthServer(grpcServer, healthServer)

	log.Printf("search service started on port %s", config.SearchServicePort)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Error occurred on gRPC server startup: %v", err)
	}
}
