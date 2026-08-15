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

	notification_consumer "github.com/JustUzair/go-grpc-irctc-backend/notification-service/server/internal/kafka/consumer"
	"github.com/JustUzair/go-grpc-irctc-backend/utils"
	"github.com/JustUzair/go-grpc-irctc-backend/utils/constants"
	env "github.com/JustUzair/go-grpc-irctc-backend/utils/env"
	logger "github.com/JustUzair/go-grpc-irctc-backend/utils/interceptors"
	"github.com/JustUzair/go-grpc-irctc-backend/utils/mailer"
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

	// Notification-specific dependency.
	mailService, err := mailer.New(config)
	if err != nil {
		log.Fatalln("Error initializing mailer for notification-service")
	}

	consumerCtx, cancelConsumer := context.WithCancel(runCtx)
	defer cancelConsumer()

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

	consumer, err := utils.NewKafkaConsumer(config.KafkaBrokers, "notification-service", []string{
		constants.TOPIC.OTP_EMAIL,
		constants.TOPIC.WELCOME_EMAIL,
		constants.TOPIC.BOOKING_EMAIL,
		constants.TOPIC.PAYMENT_EMAIL,
	})
	if err != nil {
		log.Fatalln("Error initializing kafka consumer for notification-service")
	}
	defer consumer.Close()

	lis, err := net.Listen("tcp", ":"+config.NotificationServicePort)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	defer lis.Close()
	grpcServer := grpc.NewServer(grpc.ChainUnaryInterceptor(logger.UnaryServerLoggerInterceptor))

	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(grpcServer, healthServer)

	log.Printf("notification service started on port %s", config.NotificationServicePort)

	consumerDone := make(chan error, 1)
	go func() {
		consumerDone <- consumer.Consume(
			consumerCtx, notification_consumer.ConsumeNotification(mailService),
		)
	}()
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- grpcServer.Serve(lis)
	}()

	select {
	case err := <-consumerDone:
		if err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("notification consumer stopped: %v", err)
		}
		cancelConsumer()
		grpcServer.Stop()
	case err := <-serveErr:
		if err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			log.Printf("gRPC server stopped unexpectedly: %v", err)
		}
		cancelConsumer()
	case <-runCtx.Done():
		log.Printf("shutdown signal received")

		// Set health service status to not serving
		healthServer.SetServingStatus(
			"",
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
