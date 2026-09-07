package health

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"us.ztechai.zsms/backend/internal/database"
	"us.ztechai.zsms/backend/internal/redis"
)

// Status represents health status of a service component.
type Status struct {
	Status  string            `json:"status"`
	Service string            `json:"service"`
	Time    string            `json:"time"`
	Checks  map[string]string `json:"checks,omitempty"`
}

// Handler provides HTTP health checks for Fiber.
type Handler struct {
	serviceName string
	db          *database.Client
	redis       *redis.Client
}

// NewHandler creates a new health handler.
func NewHandler(serviceName string, db *database.Client, redisClient *redis.Client) *Handler {
	return &Handler{
		serviceName: serviceName,
		db:          db,
		redis:       redisClient,
	}
}

// Health is a basic root health check.
func (h *Handler) Health(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(Status{
		Status:  "ok",
		Service: h.serviceName,
		Time:    time.Now().UTC().Format(time.RFC3339),
	})
}

// Live is a Kubernetes/Dokploy liveness probe endpoint.
func (h *Handler) Live(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(Status{
		Status:  "alive",
		Service: h.serviceName,
		Time:    time.Now().UTC().Format(time.RFC3339),
	})
}

// Ready is a Kubernetes/Dokploy readiness probe endpoint.
// It verifies that dependent stores (PostgreSQL and Redis) are reachable.
func (h *Handler) Ready(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 2*time.Second)
	defer cancel()

	checks := make(map[string]string)
	isReady := true

	// Check PostgreSQL
	if h.db != nil {
		if err := h.db.Ping(ctx); err != nil {
			checks["database"] = "unreachable"
			isReady = false
		} else {
			checks["database"] = "ok"
		}
	} else {
		checks["database"] = "unconfigured"
	}

	// Check Redis
	if h.redis != nil {
		if err := h.redis.Ping(ctx); err != nil {
			checks["redis"] = "unreachable"
			isReady = false
		} else {
			checks["redis"] = "ok"
		}
	} else {
		checks["redis"] = "unconfigured"
	}

	statusCode := fiber.StatusOK
	overallStatus := "ready"

	if !isReady {
		statusCode = fiber.StatusServiceUnavailable
		overallStatus = "degraded"
	}

	return c.Status(statusCode).JSON(Status{
		Status:  overallStatus,
		Service: h.serviceName,
		Time:    time.Now().UTC().Format(time.RFC3339),
		Checks:  checks,
	})
}
