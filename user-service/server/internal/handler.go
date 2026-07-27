package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	models "github.com/JustUzair/go-grpc-irctc-backend/user-service/server/models"
	"github.com/JustUzair/go-grpc-irctc-backend/utils/env"
	"github.com/JustUzair/go-grpc-irctc-backend/utils/mailer"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

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

func handleSendOTP(ctx context.Context, config env.Config, redis *redis.Client, db *gorm.DB, firstname string, lastname string, email string, password string) (string, error) {

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// -------------------------------------------------------------
	// Handler Logic
	// -------------------------------------------------------------
	var existingUser *models.User = nil
	err := db.WithContext(ctx).Where("email = ?", email).First(&existingUser).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { // User doesnt exist, handle success flow
			var otpSessionId string
			hashedPassword, err := HashPassword(password)
			if err != nil {
				return "", fmt.Errorf("hash signup password: %w", err)
			}

			var meta = &Meta{
				firstname,
				lastname,
				email,
				hashedPassword,
			}

			otp, otpSessionId, err := GenerateAndStoreOTP(ctx, redis, config, meta)
			if err != nil {
				return "", err
			}

			otpTTL := time.Duration(config.OTPTTL) * time.Second
			expiresInMinutes := int((otpTTL + time.Minute - 1) / time.Minute)

			mailingService, err := mailer.NewResend(mailer.ResendConfig{
				APIKey:      config.ResendAPIKey,
				FromName:    config.EmailFromName,
				FromAddress: config.EmailFromAddress,
			},
			)

			if err != nil {
				return "", err
			}
			_, err = mailingService.SendEmail(ctx, mailer.EmailTemplate(mailer.SendOTP), mailer.EmailParams{
				ToEmailAddress: meta.Email,
				TemplateData: mailer.SendOTPTemplateData{
					Name:             firstname + " " + lastname,
					OTP:              string(otp),
					ExpiresInMinutes: expiresInMinutes,
				},
			})

			if err != nil {
				RemoveStoredOTP(ctx, redis, otpSessionId)
				return "", err
			}

			return otpSessionId, nil
		} else {
			log.Printf("DB query error: %v", err)
			return "", err
		}
	} else {
		log.Printf("User already exists")
		return "", status.Error(codes.AlreadyExists, "User already exists")
	}

}
