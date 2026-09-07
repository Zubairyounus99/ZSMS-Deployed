package middleware

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"
)

// Logger logs inbound HTTP requests with duration and status code.
func Logger(log *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		duration := time.Since(start)

		reqID, _ := c.Locals(LocalsRequestID).(string)
		status := c.Response().StatusCode()

		attrs := []slog.Attr{
			slog.String("request_id", reqID),
			slog.String("method", c.Method()),
			slog.String("path", c.Path()),
			slog.Int("status", status),
			slog.Duration("latency", duration),
			slog.String("ip", c.IP()),
		}

		if err != nil {
			attrs = append(attrs, slog.String("error", err.Error()))
			log.LogAttrs(c.Context(), slog.LevelError, "HTTP request failed", attrs...)
		} else {
			log.LogAttrs(c.Context(), slog.LevelInfo, "HTTP request handled", attrs...)
		}

		return err
	}
}
