package auth

import (
	"fmt"

	"github.com/JustUzair/go-grpc-irctc-backend/utils/env"
	"github.com/golang-jwt/jwt/v5"
)

type JWTPayload struct {
	UserID string `json:"id"`
	jwt.RegisteredClaims
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
	if tokenString == "" || secret == "" {
		return nil, fmt.Errorf("missing token or signing secret")
	}

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
