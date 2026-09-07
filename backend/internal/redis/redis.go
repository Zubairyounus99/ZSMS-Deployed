package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"us.ztechai.zsms/backend/internal/config"
)

// Client wraps the go-redis client.
type Client struct {
	RDB *redis.Client
}

// Connect initializes a connection to Redis.
func Connect(ctx context.Context, cfg *config.Config) (*Client, error) {
	if cfg.RedisURL == "" {
		return nil, fmt.Errorf("redis URL is empty")
	}

	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis URL: %w", err)
	}

	opt.MaxRetries = cfg.RedisMaxRetries

	rdb := redis.NewClient(opt)

	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if err := rdb.Ping(pingCtx).Err(); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	return &Client{RDB: rdb}, nil
}

// Ping checks if Redis is reachable.
func (c *Client) Ping(ctx context.Context) error {
	if c == nil || c.RDB == nil {
		return fmt.Errorf("redis client is not initialized")
	}
	ctxTimeout, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return c.RDB.Ping(ctxTimeout).Err()
}

// Close gracefully closes the Redis client.
func (c *Client) Close() error {
	if c != nil && c.RDB != nil {
		return c.RDB.Close()
	}
	return nil
}
