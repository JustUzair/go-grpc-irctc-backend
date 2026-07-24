package main

import (
	"context"
	"fmt"
	"log"

	"github.com/JustUzair/irctc-microservice/env"
	userv1 "github.com/JustUzair/irctc-microservice/gen/go/user/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

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
	res, err := client.GetUser(context.Background(), &userv1.GetUserRequest{})

	if err != nil {
		log.Fatalf("Failed to get user: %v", err)
	}

	fmt.Printf("%+v\n", res)
}
