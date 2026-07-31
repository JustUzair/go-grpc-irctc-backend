package service

import (
	"github.com/JustUzair/go-grpc-irctc-backend/utils/env"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type SendOTPInput struct {
	Config    env.Config
	Redis     *redis.Client
	DB        *gorm.DB
	Firstname string
	Lastname  string
	Email     string
	Password  string
}

type VerifyOTPInput struct {
	Config       env.Config
	Redis        *redis.Client
	DB           *gorm.DB
	Otp          string
	OtpSessionId string
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
