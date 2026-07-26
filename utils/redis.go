package utils

import (
	"context"
	"fmt"

	redis "github.com/redis/go-redis/v9"
)

func NewRedisClient(ctx context.Context, addr string) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:       addr,
		MaxRetries: 3,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("connect to Redis: %w", err)
	}

	return client, nil
}
