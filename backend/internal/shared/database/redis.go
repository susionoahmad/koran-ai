package database

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"koran-ai-backend/internal/shared/config"
)

// ConnectRedis initializes and returns a redis.Client.
func ConnectRedis(ctx context.Context, cfg *config.Config) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       0, // Use default DB
	})

	// Ping redis to ensure connectivity
	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := rdb.Ping(pingCtx).Err(); err != nil {
		rdb.Close()
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	return rdb, nil
}
