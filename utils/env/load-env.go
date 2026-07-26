package env

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

const (
	userServicePortKey    = "USER_SERVICE_PORT"
	bookingServicePortKey = "BOOKING_SERVICE_PORT"
	paymentServicePortKey = "PAYMENT_SERVICE_PORT"
	searchServicePortKey  = "SEARCH_SERVICE_PORT"
	redisAddressKey       = "REDIS_ADDRESS"
	redisPasswordKey      = "REDIS_PASSWORD"
	userDatabaseURLKey    = "USER_DATABASE_URL"
	bookingDatabaseURLKey = "BOOKING_DATABASE_URL"
	paymentDatabaseURLKey = "PAYMENT_DATABASE_URL"
	testDatabaseURLKey    = "TEST_DATABASE_URL"
	resendAPIKey          = "RESEND_API_KEY"
	emailFromName         = "EMAIL_FROM_NAME"
	emailFromAddress      = "EMAIL_FROM_ADDRESS"
)

type Config struct {
	UserServicePort    string
	BookingServicePort string
	PaymentServicePort string
	SearchServicePort  string
	RedisAddress       string
	RedisPassword      string
	UserDatabaseURL    string
	BookingDatabaseURL string
	PaymentDatabaseURL string
	TestDatabaseURL    string
	ResendAPIKey       string
	EmailFromName      string
	EmailFromAddress   string
}

func Load() (Config, error) {
	if err := loadOptionalDotEnvFiles(dotEnvPaths()...); err != nil {
		return Config{}, err
	}

	return Config{
		UserServicePort:    os.Getenv(userServicePortKey),
		BookingServicePort: os.Getenv(bookingServicePortKey),
		PaymentServicePort: os.Getenv(paymentServicePortKey),
		SearchServicePort:  os.Getenv(searchServicePortKey),
		RedisAddress:       os.Getenv(redisAddressKey),
		RedisPassword:      os.Getenv(redisPasswordKey),
		UserDatabaseURL:    os.Getenv(userDatabaseURLKey),
		BookingDatabaseURL: os.Getenv(bookingDatabaseURLKey),
		PaymentDatabaseURL: os.Getenv(paymentDatabaseURLKey),
		TestDatabaseURL:    os.Getenv(testDatabaseURLKey),
		ResendAPIKey:       os.Getenv(resendAPIKey),
		EmailFromName:      os.Getenv(emailFromName),
		EmailFromAddress:   os.Getenv(emailFromAddress),
	}, nil
}

func dotEnvPaths() []string {
	const (
		localEnvFile = ".env.local"
		envFile      = ".env"
	)

	workingDirectory, err := os.Getwd()
	if err != nil {
		return []string{localEnvFile, envFile}
	}

	for directory := workingDirectory; ; directory = filepath.Dir(directory) {
		if _, err := os.Stat(filepath.Join(directory, "go.mod")); err == nil {
			return []string{
				filepath.Join(directory, localEnvFile),
				filepath.Join(directory, envFile),
			}
		}

		parent := filepath.Dir(directory)
		if parent == directory {
			break
		}
	}

	return []string{localEnvFile, envFile}
}

func loadOptionalDotEnvFiles(paths ...string) error {
	for _, path := range paths {
		if _, err := os.Stat(path); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return fmt.Errorf("inspect environment file %q: %w", path, err)
		}

		// Load does not overwrite variables already present in the process
		// environment. Loading .env.local first gives it precedence over .env.
		if err := godotenv.Load(path); err != nil {
			return fmt.Errorf("load environment file %q: %w", path, err)
		}
	}

	return nil
}
