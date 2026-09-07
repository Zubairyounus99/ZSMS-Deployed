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

	// Retry loop: Redis container may take a moment to accept connections on cold boot
	var lastErr error
	maxRetries := 15
	for attempt := 1; attempt <= maxRetries; attempt++ {
		pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		err = rdb.Ping(pingCtx).Err()
		cancel()
		if err == nil {
			return &Client{RDB: rdb}, nil
		}
		lastErr = err

		select {
		case <-ctx.Done():
			_ = rdb.Close()
			return nil, fmt.Errorf("redis connection context cancelled: %w", ctx.Err())
		case <-time.After(2 * time.Second):
		}
	}

	_ = rdb.Close()
	return nil, fmt.Errorf("failed to connect to redis after %d attempts: %w", maxRetries, lastErr)
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
