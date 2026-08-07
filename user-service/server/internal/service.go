package service

import (
	"context"

	userv1 "github.com/JustUzair/go-grpc-irctc-backend/gen/go/user/v1"
	"github.com/JustUzair/go-grpc-irctc-backend/user-service/server/interceptors"
	"github.com/JustUzair/go-grpc-irctc-backend/utils/env"
	custom_errors "github.com/JustUzair/go-grpc-irctc-backend/utils/errors"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
		User: &userv1.User{},
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
		SendOTPInput{
			Config:    this.Config,
			Redis:     this.RedisClient,
			DB:        this.DB,
			Firstname: req.FirstName,
			Lastname:  req.LastName,
			Email:     req.Email,
			Password:  req.Password,
		},
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

func (this *UserService) VerifyOTP(ctx context.Context, req *userv1.VerifyOTPRequest) (*userv1.VerifyOTPResponse, error) {
	otp := req.Otp
	otp_session_id := req.OtpSessionId

	if len(otp) == 0 || len(otp_session_id) == 0 {
		return nil, custom_errors.ERR_BAD_REQUEST
	}

	new_user, err := handleVerifyOTP(ctx, VerifyOTPInput{
		Config:       this.Config,
		Redis:        this.RedisClient,
		DB:           this.DB,
		Otp:          otp,
		OtpSessionId: otp_session_id,
	})

	if err != nil {
		return nil, err
	}
	return &userv1.VerifyOTPResponse{
		User: &userv1.User{
			FirstName:     new_user.FirstName,
			LastName:      new_user.LastName,
			Email:         new_user.Email,
			EmailVerified: true,
		},
		Success:       true,
		StatusMessage: "User account created successfully!",
	}, nil
}

func (this *UserService) Login(ctx context.Context, req *userv1.LoginRequest) (*userv1.LoginResponse, error) {
	email := req.Email
	password := req.Password
	meta, ok := interceptors.GetMetaFromContext(ctx)
	if !ok || meta == nil {
		return nil, status.Error(codes.Internal, "request metadata unavailable")
	}

	if len(email) == 0 || len(password) == 0 {
		return nil, custom_errors.ERR_BAD_REQUEST
	}

	accessToken, refreshToken, loggedInUser, err := handleLogin(
		ctx, LoginInput{
			Config:   this.Config,
			Redis:    this.RedisClient,
			DB:       this.DB,
			Email:    email,
			Password: password,
			DeviceId: meta.DeviceFingerprint,
		},
	)

	if err != nil {
		return nil, err
	}
	return &userv1.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User: &userv1.User{
			FirstName:     loggedInUser.FirstName,
			LastName:      loggedInUser.LastName,
			Email:         loggedInUser.Email,
			EmailVerified: loggedInUser.EmailVerified,
		},
		AccessTokenExpiresIn:  int64(this.Config.AccessTokenExp),
		RefreshTokenExpiresIn: int64(this.Config.RefreshTokenExp),
	}, nil

}

func (this *UserService) RotateRefreshToken(ctx context.Context, req *userv1.RotateRefreshTokenRequest) (*userv1.RotateRefreshTokenResponse, error) {
	refreshToken := req.RefreshToken
	meta, ok := interceptors.GetMetaFromContext(ctx)
	if !ok || meta == nil {
		return nil, status.Error(codes.Internal, "request metadata unavailable")
	}

	if len(refreshToken) == 0 {
		return nil, custom_errors.ERR_UNAUTHORIZED
	}
	deviceId := meta.DeviceFingerprint

	newAccessToken, newRefreshToken, err := handleRotateRefreshToken(
		ctx, RotateRefreshTokenInput{
			Config:       this.Config,
			Redis:        this.RedisClient,
			DB:           this.DB,
			RefreshToken: refreshToken,
			DeviceId:     deviceId,
		},
	)
	if err != nil {
		return nil, err
	}
	return &userv1.RotateRefreshTokenResponse{
		AccessToken:           newAccessToken,
		RefreshToken:          newRefreshToken,
		AccessTokenExpiresIn:  int64(this.Config.AccessTokenExp),
		RefreshTokenExpiresIn: int64(this.Config.RefreshTokenExp),
	}, nil
}
