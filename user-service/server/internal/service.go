package service

import (
	"context"

	userv1 "github.com/JustUzair/irctc-microservice/gen/go/user/v1"
	"github.com/JustUzair/irctc-microservice/utils/env"
	custom_errors "github.com/JustUzair/irctc-microservice/utils/errors"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type UserService struct {
	userv1.UnimplementedUserServiceServer
	DB          *gorm.DB
	RedisClient *redis.Client
	Config      env.Config
}

func (*UserService) GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
	return &userv1.GetUserResponse{
		User: &userv1.User{Name: "user"},
	}, nil
}

func (this *UserService) SendOTP(ctx context.Context, req *userv1.SendOTPRequest) (*userv1.SendOTPResponse, error) {

	firstname := req.FirstName
	lastname := req.LastName
	password := req.Password
	email := req.Email
	confirmPassword := req.ConfirmPassword

	if len(firstname) == 0 ||
		len(lastname) == 0 ||
		len(password) == 0 ||
		len(email) == 0 ||
		len(confirmPassword) == 0 {
		return nil, custom_errors.ERR_BAD_REQUEST
	}
	if password != confirmPassword {
		return nil, custom_errors.ERR_PASSWORD_MISMATCH
	}

	otpSessionId, err := handleSendOTP(
		ctx,
		this.Config,
		this.RedisClient,
		this.DB,
		firstname,
		lastname,
		email,
		password,
	)

	if err != nil {
		return nil, err
	}

	return &userv1.SendOTPResponse{
		Status:       true,
		Message:      "OTP sent successfully",
		OtpSessionId: otpSessionId,
	}, nil
}
