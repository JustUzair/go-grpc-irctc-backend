package main

import (
	"context"
	"errors"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	service "github.com/JustUzair/go-grpc-irctc-backend/booking-service/server/internal"
	bookingv1 "github.com/JustUzair/go-grpc-irctc-backend/gen/go/booking/v1"

	"github.com/JustUzair/go-grpc-irctc-backend/utils"
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

	runCtx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()
	startupCtx, cancelStartup := context.WithTimeout(runCtx, 10*time.Second)
	defer cancelStartup()

	db, err := utils.NewGormClient(startupCtx, config.BookingDatabaseURL, &utils.PostgresGorm{
		MaxOpenConns: 10,
		MaxIdleConns: 5,
	})
	if err != nil {
		log.Fatalf("connect to booking database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("get booking database pool: %v", err)
	}
	defer sqlDB.Close()

	redisClient, err := utils.NewRedisClient(startupCtx, config.RedisAddress, config.RedisPassword)
	if err != nil {
		log.Fatalf("connect to Redis: %v", err)
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			log.Printf("close Redis client: %v", err)
		}
	}()

	kafkaClient, err := utils.NewKafkaProducer(config.KafkaBrokers, "booking-service")
	if err != nil {
		log.Fatalf("connect to Kafka: %v", err)
	}
	defer func() {
		if err := kafkaClient.Close(); err != nil {
			log.Printf("close Kafka producer: %v", err)
		}
	}()

	lis, err := net.Listen("tcp", ":"+config.BookingServicePort)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	defer lis.Close()

	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(logger.UnaryServerLoggerInterceptor))
	bookingService := &service.BookingService{}
	healthServer := health.NewServer()

	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus(bookingv1.BookingService_ServiceDesc.ServiceName, healthpb.HealthCheckResponse_SERVING)

	bookingv1.RegisterBookingServiceServer(grpcServer, bookingService)
	healthpb.RegisterHealthServer(grpcServer, healthServer)

	log.Printf("booking service started on port %s", config.BookingServicePort)

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
		healthServer.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
		healthServer.SetServingStatus(bookingv1.BookingService_ServiceDesc.ServiceName, healthpb.HealthCheckResponse_NOT_SERVING)

		gracefulShutdown := make(chan struct{})
		go func() {
			grpcServer.GracefulStop()
			close(gracefulShutdown)
		}()

		shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
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
