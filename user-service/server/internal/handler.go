package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	models "github.com/JustUzair/go-grpc-irctc-backend/user-service/server/models"
	custom_errors "github.com/JustUzair/go-grpc-irctc-backend/utils/errors"
	"github.com/JustUzair/go-grpc-irctc-backend/utils/mailer"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func handleSendOTP(ctx context.Context, input SendOTPInput) (string, error) {

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	db := input.DB
	redis := input.Redis
	config := input.Config
	firstname := input.Firstname
	lastname := input.Lastname
	email := input.Email
	password := input.Password
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

			mailingService, err := mailer.New(config)
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

func handleVerifyOTP(ctx context.Context, input VerifyOTPInput) (*models.User, error) {

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	db := input.DB
	redis := input.Redis
	config := input.Config
	otp := input.Otp
	otp_session_id := input.OtpSessionId

	// -------------------------------------------------------------
	// Handler Logic
	// -------------------------------------------------------------
	meta := VerifyAndConsumeOTP(ctx, redis, config, otp, otp_session_id)
	if meta == nil {
		return nil, fmt.Errorf("incorrect or expired otp entered")
	}

	new_user := models.User{
		FirstName:     meta.FirstName,
		LastName:      meta.LastName,
		Email:         meta.Email,
		Password:      &meta.HashedPassword,
		EmailVerified: true,
	}

	// var user userv1.User = userv1.User
	result := db.Create(&new_user)
	if result.RowsAffected != 1 || result.Error != nil {
		return nil, fmt.Errorf("error creating user record in db")
	}

	mailingService, err := mailer.New(config)
	if err != nil {
		return nil, err
	}
	_, err = mailingService.SendEmail(ctx, mailer.EmailTemplate(mailer.VerifyOTP), mailer.EmailParams{
		ToEmailAddress: meta.Email,
		TemplateData: mailer.VerifyOTPTemplateData{
			Name: meta.FirstName + " " + meta.LastName,
		},
	})
	return &new_user, nil

}

func handleLogin(ctx context.Context, input LoginInput) (string, string, *models.User, error) {
	db := input.DB
	redis := input.Redis
	config := input.Config
	email := input.Email
	password := input.Password
	deviceId := input.DeviceId

	var existingUser *models.User = nil
	err := db.WithContext(ctx).Where("email = ?", email).First(&existingUser).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { // User doesnt exist
			return "", "", nil, custom_errors.ERR_EMAIL_NOT_FOUND
		} else {
			log.Printf("DB query error: %v", err)
			return "", "", nil, err
		}
	}
	// User found; compare the supplied password with the stored hash.
	if existingUser.Password == nil {
		return "", "", nil, custom_errors.ERR_INCORRECT_PASSWORD
	}
	err = bcrypt.CompareHashAndPassword([]byte(*existingUser.Password), []byte(password))
	if err != nil {
		return "", "", nil, custom_errors.ERR_INCORRECT_PASSWORD
	}

	accessToken, err := GenerateAccessToken(existingUser.ID, config)
	if err != nil {
		return "", "", nil, fmt.Errorf("error generating access token: %w", err)
	}
	refreshToken, jti, err := GenerateRefreshToken(existingUser.ID, config)
	if err != nil {
		return "", "", nil, fmt.Errorf("error generating refresh token: %w", err)
	}
	refreskTokenKey := fmt.Sprintf("refresh:%s:%s", existingUser.ID, deviceId)
	redis.Set(ctx, refreskTokenKey, jti, time.Duration(config.RefreshTokenExp*int(time.Second)))
	user, err := json.Marshal(existingUser)
	if err != nil {
		return "", "", nil, fmt.Errorf("error marshaling user: %w", err)
	}
	userKey := fmt.Sprintf("user:%s", existingUser.ID)
	redis.Set(ctx, userKey, user, time.Duration(config.RedisUserTTL*int(time.Second)))
	return accessToken, refreshToken, existingUser, nil
}
