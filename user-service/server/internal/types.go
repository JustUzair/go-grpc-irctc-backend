package service

import (
	"github.com/JustUzair/go-grpc-irctc-backend/utils"
	"github.com/JustUzair/go-grpc-irctc-backend/utils/env"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type SendOTPInput struct {
	Config    env.Config
	Redis     *redis.Client
	Kafka     *utils.KafkaProducer
	DB        *gorm.DB
	Firstname string
	Lastname  string
	Email     string
	Password  string
}

type VerifyOTPInput struct {
	Config       env.Config
	Redis        *redis.Client
	Kafka        *utils.KafkaProducer
	DB           *gorm.DB
	Otp          string
	OtpSessionId string
}

type LoginInput struct {
	Config   env.Config
	Redis    *redis.Client
	DB       *gorm.DB
	Email    string
	Password string
	DeviceId string
}

type RotateRefreshTokenInput struct {
	Config       env.Config
	Redis        *redis.Client
	DB           *gorm.DB
	RefreshToken string
	DeviceId     string
}

type VerifyGoogleIDTokenInput struct {
	Config   env.Config
	Redis    *redis.Client
	DB       *gorm.DB
	IDToken  string
	DeviceId string
}

type VerifyGoogleIDTokenOutput struct {
	Provider      string
	ProviderID    string
	Email         string
	FirstName     string
	LastName      string
	EmailVerified bool
}

type Meta struct {
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name"`
	Email          string `json:"email"`
	HashedPassword string `json:"hashed_password"`
}

type OTPSessionData struct {
	HashedOTP string `json:"hashed_otp"`
	Meta      Meta   `json:"meta"`
}

type JWTPayload struct {
	UserID string `json:"id"`
	jwt.RegisteredClaims
}
