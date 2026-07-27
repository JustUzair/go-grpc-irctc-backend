package main

import (
	"context"
	"log"
	"net"
	"time"

	userv1 "github.com/JustUzair/irctc-microservice/gen/go/user/v1"
	service "github.com/JustUzair/irctc-microservice/user-service/server/internal"
	"github.com/JustUzair/irctc-microservice/user-service/server/models"

	"github.com/JustUzair/irctc-microservice/utils"
	env "github.com/JustUzair/irctc-microservice/utils/env"
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := utils.NewGormClient(ctx, config.UserDatabaseURL, &utils.PostgresGorm{
		MaxOpenConns: 10,
		MaxIdleConns: 5,
	})
	if err != nil {
		log.Fatalf("connect to user database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("get user database pool: %v", err)
	}
	defer sqlDB.Close()

	if err := db.AutoMigrate(&models.User{}); err != nil {
		log.Fatalf("auto-migrate user schema: %v", err)
	}

	redisClient, err := utils.NewRedisClient(ctx, config.RedisAddress, config.RedisPassword)
	if err != nil {
		log.Fatalf("cannot instantiate redis client")
	}

	lis, err := net.Listen("tcp", ":"+config.UserServicePort)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(logger.UnaryServerLoggerInterceptor))
	userService := &service.UserService{
		DB:          db,
		RedisClient: redisClient,
		Config:      config,
	}

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
