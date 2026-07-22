package env

import (
	"errors"
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

const (
	userServicePortKey    = "USER_SERVICE_PORT"
	bookingServicePortKey = "BOOKING_SERVICE_PORT"
	paymentServicePortKey = "PAYMENT_SERVICE_PORT"
	searchServicePortKey  = "SEARCH_SERVICE_PORT"
)

type Config struct {
	UserServicePort    string
	BookingServicePort string
	PaymentServicePort string
	SearchServicePort  string
}

func Load() (Config, error) {
	if err := loadOptionalDotEnvFiles(".env.local", ".env"); err != nil {
		return Config{}, err
	}

	return Config{
		UserServicePort:    os.Getenv(userServicePortKey),
		BookingServicePort: os.Getenv(bookingServicePortKey),
		PaymentServicePort: os.Getenv(paymentServicePortKey),
		SearchServicePort:  os.Getenv(searchServicePortKey),
	}, nil
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
