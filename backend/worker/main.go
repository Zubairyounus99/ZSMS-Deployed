package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"us.ztechai.zsms/backend/internal/config"
	"us.ztechai.zsms/backend/internal/database"
	"us.ztechai.zsms/backend/internal/logger"
	"us.ztechai.zsms/backend/internal/redis"
)

func main() {
	// 1. Load Configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("worker failed to load configuration", "error", err)
		os.Exit(1)
	}

	// 2. Initialize Structured Logger
	log := logger.New(cfg.AppEnv, "zsms-worker")
	log.Info("initializing ZSMS Background Worker service",
		slog.String("app_name", cfg.AppName),
		slog.String("env", cfg.AppEnv),
	)

	// 3. Connect to PostgreSQL
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()

	dbClient, err := database.Connect(ctx, cfg)
	if err != nil {
		log.Warn("database connection not immediately available (worker running in degraded mode)",
			slog.String("error", err.Error()),
		)
	} else {
		log.Info("worker successfully connected to PostgreSQL")
		defer dbClient.Close()
	}

	// 4. Connect to Redis (Core queue backend)
	redisClient, err := redis.Connect(ctx, cfg)
	if err != nil {
		log.Warn("redis connection not immediately available (worker running in degraded mode)",
			slog.String("error", err.Error()),
		)
	} else {
		log.Info("worker successfully connected to Redis queue engine")
		defer redisClient.Close()
	}

	// 5. Worker Lifecycle Loop (Foundation Stage)
	log.Info("ZSMS background worker started successfully and awaiting job queues")

	// Ticker for periodic health heartbeat logging
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	running := true
	for running {
		select {
		case <-ticker.C:
			// Foundation heartbeat: checks connection health
			dbStatus := "unreachable"
			if dbClient != nil && dbClient.Ping(context.Background()) == nil {
				dbStatus = "ok"
			}
			redisStatus := "unreachable"
			if redisClient != nil && redisClient.Ping(context.Background()) == nil {
				redisStatus = "ok"
			}
			log.Info("worker heartbeat status",
				slog.String("database", dbStatus),
				slog.String("redis", redisStatus),
			)

		case sig := <-quit:
			log.Info("shutdown signal received by worker", slog.String("signal", sig.String()))
			running = false
		}
	}

	log.Info("worker graceful shutdown completed")
}
