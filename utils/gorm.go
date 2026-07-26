package utils

import (
	"context"
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type PostgresGorm struct {
	MaxOpenConns int
	MaxIdleConns int
}

func NewGormClient(ctx context.Context, dbURL string, config PostgresGorm) (*gorm.DB, error) {
	db, err := gorm.Open(
		postgres.Open(dbURL),
		&gorm.Config{
			DisableAutomaticPing: true, // ping later with useful startup context
		},
	)

	if err != nil {
		return nil, fmt.Errorf("open PostgreSQL database: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get PostgreSQL connection pool: %w", err)
	}

	if config.MaxOpenConns != 0 {
		sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	}
	if config.MaxIdleConns != 0 {
		sqlDB.SetMaxIdleConns(config.MaxIdleConns)
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("connect to PostgreSQL: %w", err)
	}

	return db, nil

}
