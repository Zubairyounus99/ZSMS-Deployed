package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

const (
	HeaderRequestID = "X-Request-ID"
	LocalsRequestID = "request_id"
)

// RequestID attaches or propagates a unique Request ID across the HTTP context.
func RequestID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		reqID := c.Get(HeaderRequestID)
		if reqID == "" {
			reqID = uuid.New().String()
		}

		c.Locals(LocalsRequestID, reqID)
		c.Set(HeaderRequestID, reqID)

		return c.Next()
	}
}
