package middleware

import (
	"fmt"
	"log/slog"
	"runtime/debug"

	"github.com/gofiber/fiber/v2"
)

// Recover intercepts panics and returns a structured 500 JSON response.
func Recover(log *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) (err error) {
		defer func() {
			if r := recover(); r != nil {
				reqID, _ := c.Locals(LocalsRequestID).(string)
				stack := string(debug.Stack())

				log.Error("panic recovered in HTTP handler",
					slog.String("request_id", reqID),
					slog.Any("panic", r),
					slog.String("stack", stack),
				)

				err = c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"success": false,
					"error": fiber.Map{
						"code":    "INTERNAL_SERVER_ERROR",
						"message": "An unexpected error occurred. Please try again later.",
					},
					"request_id": reqID,
				})
			}
		}()

		return c.Next()
	}
}
