package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/JustUzair/go-grpc-irctc-backend/user-service/server/internal/kafka/producer"
	models "github.com/JustUzair/go-grpc-irctc-backend/user-service/server/models"
	"github.com/JustUzair/go-grpc-irctc-backend/utils"
	"github.com/JustUzair/go-grpc-irctc-backend/utils/auth"
	custom_errors "github.com/JustUzair/go-grpc-irctc-backend/utils/errors"
	"golang.org/x/crypto/bcrypt"
	idtoken "google.golang.org/api/idtoken"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

func handleSendOTP(ctx context.Context, input SendOTPInput) (string, error) {

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	db := input.DB
	kafkaEventProducer := input.Kafka
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
			name := firstname + " " + lastname
			if err = producer.SendOtpEmail(ctx, kafkaEventProducer, meta.Email, name, string(otp), expiresInMinutes); err != nil {
				if errors.Is(err, utils.ErrDeliveryUnknown) {
					// Kafka may still deliver the event. Keep the OTP
					// until its normal Redis TTL expires.
					log.Printf("OTP notification delivery outcome unknown: %v", err)
					return "", status.Error(
						codes.Unavailable,
						"OTP delivery is still being processed",
					)
				}
				if deleteErr := RemoveStoredOTP(ctx, redis, otpSessionId); deleteErr != nil {
					log.Printf("failed to remove OTP after publish failure: %v", deleteErr)
				}

				return "", status.Error(
					codes.Unavailable,
					"could not queue OTP email",
				)
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
	kafkaEventProducer := input.Kafka
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

	if err := producer.VerifyOtpEmail(ctx, kafkaEventProducer, meta.Email, meta.FirstName, meta.LastName); err != nil {
		// The user was already created successfully. Do not make the
		// client retry account creation because email delivery failed.
		log.Printf(
			"welcome email event failed for user %s: %v",
			new_user.ID,
			err,
		)
	}

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
	refreskTokenKey := GetRefreshTokenKey(deviceId, existingUser.ID)
	redis.Set(ctx, refreskTokenKey, jti, time.Duration(config.RefreshTokenExp*int(time.Second)))
	user, err := json.Marshal(existingUser)
	if err != nil {
		return "", "", nil, fmt.Errorf("error marshaling user: %w", err)
	}
	userKey := GetUserKey(existingUser.ID)
	redis.Set(ctx, userKey, user, time.Duration(config.RedisUserTTL*int(time.Second)))
	return accessToken, refreshToken, existingUser, nil
}

func handleRotateRefreshToken(ctx context.Context, input RotateRefreshTokenInput) (string, string, error) {
	redis := input.Redis
	payload, err := auth.VerifyRefreshToken(input.RefreshToken, input.Config)

	if err != nil {
		return "", "", err
	}
	userId := payload.UserID
	deviceId := input.DeviceId
	jti := payload.ID
	refreshTokenKey := GetRefreshTokenKey(deviceId, userId)
	storedJTI := redis.Get(ctx, refreshTokenKey).Val()
	if len(storedJTI) == 0 {
		return "", "", custom_errors.ERR_SESSION_EXPIRED
	}
	// Check if JTI has been reused
	if storedJTI != jti {
		redis.Del(ctx, refreshTokenKey)
		return "", "", custom_errors.ERR_SESSION_EXPIRED
	}
	newAccessToken, err := GenerateAccessToken(userId, input.Config)
	if err != nil {
		return "", "", fmt.Errorf("error generating access token: %w", err)
	}
	newRefreshToken, newJTI, err := GenerateRefreshToken(userId, input.Config)
	if err != nil {
		return "", "", fmt.Errorf("error generating refresh token: %w", err)
	}
	redis.Set(ctx, refreshTokenKey, newJTI, time.Duration(input.Config.RefreshTokenExp*int(time.Second)))
	return newAccessToken, newRefreshToken, nil

}

func handleVerifyGoogleIDToken(ctx context.Context, input VerifyGoogleIDTokenInput) (string, string, *models.User, error) {
	idToken := input.IDToken
	config := input.Config
	db := input.DB
	redis := input.Redis
	deviceId := input.DeviceId
	payload, err := idtoken.Validate(ctx, idToken, config.GoogleClientID)
	if err != nil {
		return "", "", nil, custom_errors.ERR_UNAUTHORIZED
	}

	email, ok := payload.Claims["email"].(string)
	if !ok || len(email) == 0 {
		return "", "", nil, custom_errors.ERR_BAD_REQUEST
	}

	firstName, _ := payload.Claims["given_name"].(string)
	lastName, _ := payload.Claims["family_name"].(string)
	emailVerified, ok := payload.Claims["email_verified"].(bool)
	if !ok || !emailVerified || len(payload.Subject) == 0 {
		return "", "", nil, custom_errors.ERR_UNAUTHORIZED
	}

	googleUser := &VerifyGoogleIDTokenOutput{
		Provider:      payload.Issuer,
		ProviderID:    payload.Subject,
		Email:         email,
		FirstName:     firstName,
		LastName:      lastName,
		EmailVerified: emailVerified,
	}

	var targetUser *models.User
	txErr := db.Transaction(func(tx *gorm.DB) error {

		// CASE 1. Check if google account already exists in auth-providers table
		var googleAuth *models.AuthProvider = nil
		err := tx.WithContext(ctx).Preload("User").Where("provider_id = ? AND provider = ?", googleUser.ProviderID, googleUser.Provider).First(&googleAuth).Error
		if err == nil {
			if googleAuth == nil || googleAuth.User == nil {
				return fmt.Errorf("google auth provider has no associated user")
			}

			targetUser = googleAuth.User
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("db error checking auth provider: %w", err)
		}

		// CASE 2. Check if user normally signed up with email, password; and is now trying to login with google, if so link the account with auth provider
		var existingUser *models.User
		err = tx.WithContext(ctx).Where("email = ?", googleUser.Email).First(&existingUser).Error
		if err == nil {
			newGoogleAuthUser := models.AuthProvider{
				Provider:   googleUser.Provider,
				ProviderID: googleUser.ProviderID,
				UserID:     existingUser.ID,
			}
			err := tx.WithContext(ctx).Create(&newGoogleAuthUser).Error
			if err != nil {
				return fmt.Errorf("failed to link google auth provider: %w", err)
			}
			targetUser = existingUser
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("db error checking user email: %w", err)
		}

		// CASE 3. User and Auth Provider do not exist, create both
		newUser := models.User{
			Email:         email,
			FirstName:     firstName,
			LastName:      lastName,
			EmailVerified: emailVerified,
			AuthProviders: []models.AuthProvider{
				{
					Provider:   googleUser.Provider,
					ProviderID: googleUser.ProviderID,
				},
			},
		}
		if err := tx.WithContext(ctx).Create(&newUser).Error; err != nil {
			return fmt.Errorf("failed to create new user with google provider: %w", err)
		}
		targetUser = &newUser
		return nil
	})
	if txErr != nil {
		return "", "", nil, txErr
	}
	accessToken, err := GenerateAccessToken(targetUser.ID, config)
	if err != nil {
		return "", "", nil, fmt.Errorf("error generating access token: %w", err)
	}
	refreshToken, jti, err := GenerateRefreshToken(targetUser.ID, config)
	if err != nil {
		return "", "", nil, fmt.Errorf("error generating refresh token: %w", err)
	}
	refreskTokenKey := GetRefreshTokenKey(deviceId, targetUser.ID)
	redis.Set(ctx, refreskTokenKey, jti, time.Duration(config.RefreshTokenExp*int(time.Second)))
	user, err := json.Marshal(targetUser)
	if err != nil {
		return "", "", nil, fmt.Errorf("error marshaling user: %w", err)
	}
	userKey := GetUserKey(targetUser.ID)
	redis.Set(ctx, userKey, user, time.Duration(config.RedisUserTTL*int(time.Second)))
	return accessToken, refreshToken, targetUser, nil

}
