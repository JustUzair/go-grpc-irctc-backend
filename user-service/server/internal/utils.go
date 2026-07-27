package service

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/JustUzair/go-grpc-irctc-backend/utils/env"
	custom_errors "github.com/JustUzair/go-grpc-irctc-backend/utils/errors"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

// HashPassword hashes a plain-text password using bcrypt
func HashPassword(password string) (string, error) {
	// Cost of 10 or 12 is recommended for modern web apps
	// (bcrypt.DefaultCost is 10)
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// CheckPasswordHash compares a hashed password with a plain-text input
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	// Returns nil if passwords match; non-nil error if they don't
	return err == nil
}

func RemoveStoredOTP(
	ctx context.Context,
	redis *redis.Client,
	otpSessionID string,

) error {
	key := fmt.Sprintf("otp:session:%s", otpSessionID)
	return redis.Del(ctx, key).Err()
}

func GenerateAndStoreOTP(ctx context.Context, redis *redis.Client, config env.Config, meta *Meta) (string, string, error) {

	var rateKey string = fmt.Sprintf("otp:rate:%s", meta.Email)

	sentCount, err := redis.Get(ctx, rateKey).Int()

	if err != nil {
		sentCount = 0
	}

	if sentCount >= config.OtpRateMaxPerHour {
		return "", "", custom_errors.ERR_TOO_MANY_REQUESTS
	}

	otp, err := generateNumericOTP(6)
	if err != nil {
		return "", "", err
	}

	otpSessionId := uuid.New()
	hashed := hmacFor(config.OtpHmacSecret, meta.Email, string(otp))

	sessionPayload := OTPSessionData{
		HashedOTP: hashed,
		Meta:      *meta, // or meta if meta is already a value struct
	}

	sessionJSON, err := json.Marshal(sessionPayload)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal session data: %w", err)
	}
	ttl := time.Duration(config.OTPTTL) * time.Second
	redisOtpSessionKey := fmt.Sprintf("otp:session:%s", otpSessionId.String())
	err = redis.Set(ctx, redisOtpSessionKey, sessionJSON, ttl).Err()

	if err != nil {
		return "", "", fmt.Errorf("failed to save otp session in redis: %w", err)
	}
	redis.Incr(ctx, rateKey)
	redis.Expire(ctx, rateKey, time.Hour)
	return otp, otpSessionId.String(), nil
}

// creates 6 digit otp
func generateNumericOTP(length int) (string, error) {
	const digits = "0123456789"
	otp := make([]byte, length)

	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		otp[i] = digits[num.Int64()]
	}

	return string(otp), nil
}

// hmacFor hashes (email + ":" + otp) using the HMAC secret key
func hmacFor(secretKey string, email string, otp string) string {
	// 1. Prepare data string: "email:otp"
	data := fmt.Sprintf("%s:%s", email, otp)

	// 2. Create HMAC using SHA256 and the HMAC_SECRET
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(data))

	// 3. Return as hex string
	return hex.EncodeToString(h.Sum(nil))
}
