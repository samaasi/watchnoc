package store

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/samaasi/watchnoc/internal/config"

	"github.com/redis/go-redis/v9"
)

func NewRedis(cfg config.RedisConfig) (*redis.Client, error) {
	opts, err := redis.ParseURL(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis url: %w", err)
	}

	// Configure connection pool
	opts.PoolSize = 100
	opts.MinIdleConns = 10
	opts.ConnMaxLifetime = time.Hour
	opts.ConnMaxIdleTime = 10 * time.Minute

	rdb := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	log.Println("Successfully connected to Redis")
	return rdb, nil
}
