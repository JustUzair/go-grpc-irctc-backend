package services

import (
	"context"

	userv1 "github.com/JustUzair/irctc-microservice/gen/go/user/v1"
)

type UserService struct {
	userv1.UnimplementedUserServiceServer
}

func (*UserService) GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
	return &userv1.GetUserResponse{
		User: &userv1.User{Name: "user"},
	}, nil
}
