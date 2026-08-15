package main

import (
	"context"
	"fmt"
	"log"
	"time"

	userv1 "github.com/JustUzair/go-grpc-irctc-backend/gen/go/user/v1"
	env "github.com/JustUzair/go-grpc-irctc-backend/utils/env"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// Smoke test client --> TODO entrypoint client wrappers
func main() {
	config, err := env.Load()

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	conn, err := grpc.NewClient(":"+config.UserServicePort, opts...)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	defer conn.Close()

	client := userv1.NewUserServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// Supply a valid access token here. The gateway will eventually do this
	// when it forwards an authenticated browser request.
	accessToken := "<access-token-from-login>"
	ctx = metadata.AppendToOutgoingContext(
		ctx,
		"authorization",
		"Bearer "+accessToken,
	)

	res, err := client.GetUser(ctx, &userv1.GetUserRequest{})

	if err != nil {
		log.Fatalf("Failed to get user: %v", err)
	}

	fmt.Printf("%+v\n", res)
}
