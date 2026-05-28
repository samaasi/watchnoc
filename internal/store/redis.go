package store

import (
	"context"
	"fmt"
	"log"

	"github.com/samaasi/watchnoc/internal/config"

	"github.com/redis/go-redis/v9"
)

func NewRedis(cfg config.RedisConfig) (*redis.Client, error) {
	opts, err := redis.ParseURL(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis url: %w", err)
	}

	rdb := redis.NewClient(opts)

	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	log.Println("Successfully connected to Redis")
	return rdb, nil
}
