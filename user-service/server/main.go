package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	userv1 "github.com/JustUzair/go-grpc-irctc-backend/gen/go/user/v1"

	service "github.com/JustUzair/go-grpc-irctc-backend/user-service/server/internal"
	"github.com/JustUzair/go-grpc-irctc-backend/user-service/server/models"

	"github.com/JustUzair/go-grpc-irctc-backend/utils"
	env "github.com/JustUzair/go-grpc-irctc-backend/utils/env"
	"github.com/JustUzair/go-grpc-irctc-backend/utils/interceptors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	config, err := env.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	runCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	startupCtx, cancelStartup := context.WithTimeout(
		runCtx,
		10*time.Second,
	)
	defer cancelStartup()

	if err := utils.EnsureTopics(
		startupCtx,
		config.KafkaBrokers,
	); err != nil {
		log.Fatalf("initialize Kafka topics: %v", err)
	}

	db, err := utils.NewGormClient(startupCtx, config.UserDatabaseURL, &utils.PostgresGorm{
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

	if err := db.WithContext(startupCtx).AutoMigrate(&models.User{}, &models.AuthProvider{}); err != nil {
		log.Fatalf("auto-migrate user schema: %v", err)
	}

	redisClient, err := utils.NewRedisClient(startupCtx, config.RedisAddress, config.RedisPassword)
	if err != nil {
		log.Fatalf("cannot instantiate redis client: %v\n", err)
	}
	defer redisClient.Close()

	kafkaClient, err := utils.NewKafkaProducer(config.KafkaBrokers, "user-service")
	if err != nil {
		log.Fatalf("cannot instantiate kafka client: %v\n", err)
	}
	defer func() {
		if err := kafkaClient.Close(); err != nil {
			slog.Error("close Kafka producer", "error", err)
		}
	}()
	lis, err := net.Listen("tcp", ":"+config.UserServicePort)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	defer lis.Close()
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			interceptors.UnaryServerLoggerInterceptor,
			interceptors.MetaInterceptor,
			interceptors.UnaryServerAuthInterceptor(
				config,
				userv1.UserService_GetUser_FullMethodName),
		),
	)
	userService := &service.UserService{
		DB:          db,
		RedisClient: redisClient,
		KafkaClient: kafkaClient,
		Config:      config,
	}

	healthServer := health.NewServer()

	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus(userv1.UserService_ServiceDesc.ServiceName, healthpb.HealthCheckResponse_SERVING)

	userv1.RegisterUserServiceServer(grpcServer, userService)
	healthpb.RegisterHealthServer(grpcServer, healthServer)

	log.Printf("user service started on port %s", config.UserServicePort)

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- grpcServer.Serve(lis)
	}()

	select {
	case err := <-serveErr:
		if err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			log.Printf("gRPC server stopped unexpectedly: %v", err)
		}
	case <-runCtx.Done():
		log.Printf("shutdown signal received")

		// Set health service status to not serving
		healthServer.SetServingStatus(
			"",
			healthpb.HealthCheckResponse_NOT_SERVING,
		)
		healthServer.SetServingStatus(
			userv1.UserService_ServiceDesc.ServiceName,
			healthpb.HealthCheckResponse_NOT_SERVING,
		)
		gracefulShutdown := make(chan struct{})
		go func() {
			grpcServer.GracefulStop()
			close(gracefulShutdown)
		}()

		shutdownCtx, cancelShutdown := context.WithTimeout(
			context.Background(),
			10*time.Second,
		)
		defer cancelShutdown()

		select {
		case <-gracefulShutdown:
			log.Printf("gRPC server stopped gracefully")
		case <-shutdownCtx.Done():
			log.Printf("graceful shutdown timed out; forcing stop")
			grpcServer.Stop()

		}

	}
}
