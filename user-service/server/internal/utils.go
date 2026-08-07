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
	jwt "github.com/golang-jwt/jwt/v5"
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
	otp_session_id string,

) error {
	key := fmt.Sprintf("otp:session:%s", otp_session_id)
	return redis.Del(ctx, key).Err()
}

func GenerateAndStoreOTP(ctx context.Context, redis *redis.Client, config env.Config, meta *Meta) (string, string, error) {

	var rate_key string = getOTPRateKey(meta.Email)

	sent_count, err := redis.Get(ctx, rate_key).Int()

	if err != nil {
		sent_count = 0
	}

	if sent_count >= config.OtpRateMaxPerHour {
		return "", "", custom_errors.ERR_TOO_MANY_REQUESTS
	}

	otp, err := generateNumericOTP(6)
	if err != nil {
		return "", "", err
	}

	otp_session_id := uuid.New()
	hashed := hmacFor(config.OtpHmacSecret, meta.Email, string(otp))

	session_payload := OTPSessionData{
		HashedOTP: hashed,
		Meta:      *meta, // or meta if meta is already a value struct
	}

	sessionJSON, err := json.Marshal(session_payload)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal session data: %w", err)
	}
	ttl := time.Duration(config.OTPTTL) * time.Second
	redis_otp_session_key := getOTPSessionKey(otp_session_id.String())
	err = redis.Set(ctx, redis_otp_session_key, sessionJSON, ttl).Err()

	if err != nil {
		return "", "", fmt.Errorf("failed to save otp session in redis: %w", err)
	}
	redis.Incr(ctx, rate_key)
	redis.Expire(ctx, rate_key, time.Hour)
	return otp, otp_session_id.String(), nil
}

func VerifyAndConsumeOTP(ctx context.Context, redis *redis.Client, config env.Config, otp string, otp_session_id string) *Meta {
	var session_payload OTPSessionData
	redis_otp_session_key := getOTPSessionKey(otp_session_id)

	otp_session_data, err := redis.Get(ctx, redis_otp_session_key).Result()
	if err != nil || otp_session_data == "" {
		return nil
	}
	if err := json.Unmarshal([]byte(otp_session_data), &session_payload); err != nil {
		return nil
	}
	storedOtp := session_payload.HashedOTP
	var meta Meta = session_payload.Meta
	var attempts_key string = getOTPAttemptsKey(meta.Email)

	attempt_count, err := redis.Get(ctx, attempts_key).Int()

	if err != nil {
		attempt_count = 0
	}

	if attempt_count >= config.OtpMaxVerifyAttempts {
		return nil
	}

	hashedOtp := hmacFor(config.OtpHmacSecret, meta.Email, string(otp))
	if hmac.Equal([]byte(storedOtp), []byte(hashedOtp)) {
		redis.Del(ctx, redis_otp_session_key, attempts_key, getOTPRateKey(meta.Email))
		return &meta
	} else {

		redis.Incr(ctx, attempts_key)
		redis.Expire(ctx, attempts_key, time.Hour)
		return nil
	}

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

// Auth Helpers

func GenerateAccessToken(userId string, config env.Config) (string, error) {
	payload := JWTPayload{
		ID: userId,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(config.AccessTokenExp) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
	tokenString, err := token.SignedString([]byte(config.JWTAccessSecretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil

}

func GenerateRefreshToken(userId string, config env.Config) (string, string, error) {
	payload := JWTPayload{
		ID: userId,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(config.RefreshTokenExp) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.NewString(), // jti
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
	tokenString, err := token.SignedString([]byte(config.JWTRefreshSecretKey))
	if err != nil {
		return "", "", err
	}

	return tokenString, payload.ID, nil

}

// VerifyAccessToken validates an access token using the access secret key
func VerifyAccessToken(tokenString string, config env.Config) (*JWTPayload, error) {
	return verifyToken(tokenString, config.JWTAccessSecretKey)
}

// VerifyRefreshToken validates a refresh token using the refresh secret key
func VerifyRefreshToken(tokenString string, config env.Config) (*JWTPayload, error) {
	return verifyToken(tokenString, config.JWTRefreshSecretKey)
}

func verifyToken(tokenString string, secret string) (*JWTPayload, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTPayload{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTPayload); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// Redis Session Keys

func getOTPRateKey(email string) string {
	return fmt.Sprintf("otp:rate:%s", email)
}

func getOTPAttemptsKey(email string) string {
	return fmt.Sprintf("otp:attempts:%s", email)
}

func getOTPSessionKey(session_id string) string {
	return fmt.Sprintf("otp:session:%s", session_id)
}
