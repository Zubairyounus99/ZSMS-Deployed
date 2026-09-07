package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

// CORS configures cross-origin resource sharing.
func CORS(allowedOrigins []string) fiber.Handler {
	origins := strings.Join(allowedOrigins, ", ")
	if origins == "" {
		origins = "http://localhost:3000"
	}

	return cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization, X-Request-ID, Idempotency-Key",
		AllowMethods:     "GET, POST, HEAD, PUT, DELETE, PATCH, OPTIONS",
		AllowCredentials: true,
	})
}
