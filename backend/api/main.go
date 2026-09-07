package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"us.ztechai.zsms/backend/internal/auth"
	"us.ztechai.zsms/backend/internal/config"
	"us.ztechai.zsms/backend/internal/database"
	"us.ztechai.zsms/backend/internal/gateway"
	"us.ztechai.zsms/backend/internal/handler"
	"us.ztechai.zsms/backend/internal/health"
	"us.ztechai.zsms/backend/internal/logger"
	"us.ztechai.zsms/backend/internal/middleware"
	"us.ztechai.zsms/backend/internal/redis"
	"us.ztechai.zsms/backend/internal/repository"
)

func main() {
	// 1. Load Configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	// 2. Initialize Structured Logger
	log := logger.New(cfg.AppEnv, "zsms-api")
	log.Info("initializing ZSMS API server",
		slog.String("app_name", cfg.AppName),
		slog.String("env", cfg.AppEnv),
		slog.Int("port", cfg.Port),
	)
	log.Info("configuration loaded safely", slog.Any("summary", cfg.MaskedSummary()))

	// 3. Connect to PostgreSQL
	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Second)
	defer cancel()

	dbClient, err := database.Connect(ctx, cfg)
	if err != nil {
		log.Error("database connection failed (API running in degraded mode)",
			slog.String("error", err.Error()),
		)
	} else {
		log.Info("successfully connected to PostgreSQL store")
		defer dbClient.Close()

		// Execute schema migrations safely
		migrationCtx, mCancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := dbClient.Migrate(migrationCtx); err != nil {
			log.Error("database migration error", slog.String("error", err.Error()))
		} else {
			log.Info("database schema migrations verified successfully")
		}
		mCancel()
	}

	// 4. Connect to Redis
	redisClient, err := redis.Connect(ctx, cfg)
	if err != nil {
		log.Warn("redis connection not immediately available (running in degraded readiness mode)",
			slog.String("error", err.Error()),
		)
	} else {
		log.Info("successfully connected to Redis")
		defer redisClient.Close()
	}

	// 5. Initialize Real-Time Command Delivery Hub
	hub := gateway.GetHub(log)

	// 6. Initialize Repositories
	userRepo := repository.NewUserRepository(dbClient)
	phoneRepo := repository.NewPhoneRepository(dbClient, cfg.DeviceTokenPrefix, cfg.PairingCodeExpirationMin)
	msgRepo := repository.NewMessageRepository(dbClient)

	// 7. Initialize Handlers
	authHandler := handler.NewAuthHandler(userRepo, cfg, log)
	pairingHandler := handler.NewPairingHandler(phoneRepo, log)
	phoneHandler := handler.NewPhoneHandler(phoneRepo, hub, log)
	msgHandler := handler.NewMessageHandler(msgRepo, phoneRepo, hub, log)
	wsHandler := handler.NewGatewayWSHandler(dbClient, hub, log)

	// 8. Initialize Fiber HTTP Engine
	app := fiber.New(fiber.Config{
		AppName:               fmt.Sprintf("%s API v1.0", cfg.AppName),
		DisableStartupMessage: cfg.AppEnv == "production",
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			reqID, _ := c.Locals(middleware.LocalsRequestID).(string)

			return c.Status(code).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    fmt.Sprintf("HTTP_%d", code),
					"message": err.Error(),
				},
				"request_id": reqID,
			})
		},
	})

	// 9. Global Middlewares
	app.Use(middleware.RequestID())
	app.Use(middleware.Logger(log))
	app.Use(middleware.Recover(log))
	app.Use(middleware.CORS(cfg.CORSAllowedOrigins))

	// 10. Health Endpoints
	healthHandler := health.NewHandler("zsms-api", dbClient, redisClient)
	app.Get("/health", healthHandler.Health)
	app.Get("/health/live", healthHandler.Live)
	app.Get("/health/ready", healthHandler.Ready)
	app.Get("/livez", healthHandler.Live)
	app.Get("/readyz", healthHandler.Ready)

	// 11. Register Routes (both /api/v1 and /v1 for flexibility)
	userAuth := auth.RequireUserAuth(cfg.JWTSecret, cfg.AuthDisabled, dbClient)

	registerV1Routes := func(router fiber.Router) {
		// Public Auth
		router.Get("/auth/config", authHandler.Config)
		router.Post("/auth/register", authHandler.Register)
		router.Post("/auth/login", authHandler.Login)
		router.Post("/auth/logout", authHandler.Logout)

		// Protected User Profile
		router.Get("/me", userAuth, authHandler.Me)

		// Phone Pairing
		router.Post("/pairing/sessions", userAuth, pairingHandler.CreateSession)
		router.Post("/pairing/complete", pairingHandler.CompletePairing)

		// Phone Gateways
		router.Get("/phones", userAuth, phoneHandler.ListPhones)
		router.Get("/phones/:id", userAuth, phoneHandler.GetPhone)
		router.Post("/phones/:id/heartbeat", auth.RequireDeviceAuth(dbClient), phoneHandler.Heartbeat)

		// Messaging Endpoints
		router.Post("/messages", userAuth, msgHandler.SendMessage)
		router.Get("/messages", userAuth, msgHandler.ListMessages)
		router.Get("/messages/:id", userAuth, msgHandler.GetMessage)

		// Two-way Conversations
		router.Get("/message-threads", userAuth, msgHandler.ListThreads)
		router.Get("/message-threads/:id/messages", userAuth, msgHandler.GetThreadMessages)

		// Dashboard Statistics
		router.Get("/dashboard/stats", userAuth, msgHandler.GetDashboardStats)

		// Android Gateway Device Callbacks
		router.Post("/android/messages/:id/result", auth.RequireDeviceAuth(dbClient), msgHandler.MessageResult)
		router.Post("/android/messages/inbound", auth.RequireDeviceAuth(dbClient), msgHandler.InboundMessage)
		router.Get("/android/messages/poll", auth.RequireDeviceAuth(dbClient), msgHandler.PollMessages)

		// Real-time Gateway WebSocket endpoint
		router.Get("/gateway/ws", wsHandler.UpgradeCheck, wsHandler.HandleWS())
	}

	registerV1Routes(app.Group("/api/v1"))
	registerV1Routes(app.Group("/v1"))

	// 12. Catch-all 404 Handler
	app.Use(func(c *fiber.Ctx) error {
		reqID, _ := c.Locals(middleware.LocalsRequestID).(string)
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "NOT_FOUND",
				"message": fmt.Sprintf("Cannot %s %s", c.Method(), c.Path()),
			},
			"request_id": reqID,
		})
	})

	// 13. Start Server in Background Goroutine
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	go func() {
		log.Info("ZSMS API server listening", slog.String("address", addr))
		if err := app.Listen(addr); err != nil {
			log.Info("server stopped listening", slog.String("reason", err.Error()))
		}
	}()

	// 14. Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	sig := <-quit

	log.Info("shutdown signal received", slog.String("signal", sig.String()))

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		log.Error("error during server shutdown", slog.String("error", err.Error()))
	} else {
		log.Info("API server shutdown completed cleanly")
	}
}
