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
	// res, err := client.GetUser(context.Background(), &userv1.GetUserRequest{})

	res, err := client.SendOTP(ctx, &userv1.SendOTPRequest{
		FirstName:       "John",
		LastName:        "Doe",
		Email:           "johndoe@gmail.com",
		Password:        "123456",
		ConfirmPassword: "123456",
	})

	if err != nil {
		log.Fatalf("Failed to get user: %v", err)
	}

	fmt.Printf("%+v\n", res)
}
