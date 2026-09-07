package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"us.ztechai.zsms/backend/internal/config"
)

// Client wraps the SQL database pool and provides utility methods.
type Client struct {
	DB *sql.DB
}

// Connect establishes a connection pool to PostgreSQL.
func Connect(ctx context.Context, cfg *config.Config) (*Client, error) {
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("database URL is empty")
	}

	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	db.SetMaxOpenConns(cfg.DBMaxOpenConns)
	db.SetMaxIdleConns(cfg.DBMaxIdleConns)
	db.SetConnMaxLifetime(cfg.DBConnMaxLifetime())

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Client{DB: db}, nil
}

// Ping checks if the database is reachable.
func (c *Client) Ping(ctx context.Context) error {
	if c == nil || c.DB == nil {
		return fmt.Errorf("database client is not initialized")
	}
	ctxTimeout, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return c.DB.PingContext(ctxTimeout)
}

// Close gracefully closes the database pool.
func (c *Client) Close() error {
	if c != nil && c.DB != nil {
		return c.DB.Close()
	}
	return nil
}
