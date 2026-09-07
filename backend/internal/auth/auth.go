package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"us.ztechai.zsms/backend/internal/database"
)

// UserClaims defines claims stored inside the session JWT.
type UserClaims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	jwt.RegisteredClaims
}

// HashPassword creates a bcrypt hash of a plaintext password.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword verifies a plaintext password against a bcrypt hash.
func CheckPassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// HashToken generates a SHA-256 hex string of an API or device token.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// GenerateUserToken signs a JWT for an authenticated user session.
func GenerateUserToken(userID uuid.UUID, email, secret string, expirationHours int) (string, error) {
	claims := UserClaims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expirationHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "zsms-api",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ValidateUserToken parses and validates a user session JWT.
func ValidateUserToken(tokenString, secret string) (*UserClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*UserClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token claims")
}

// RequireUserAuth middleware enforces that the request has a valid User session JWT.
func RequireUserAuth(secret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		var tokenStr string

		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
				tokenStr = parts[1]
			}
		}

		if tokenStr == "" {
			// Also check cookie as fallback for browser
			tokenStr = c.Cookies("zsms_token")
		}

		if tokenStr == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    "UNAUTHORIZED",
					"message": "Missing authentication token. Please sign in.",
				},
			})
		}

		claims, err := ValidateUserToken(tokenStr, secret)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    "INVALID_TOKEN",
					"message": "Session token is invalid or expired. Please sign in again.",
				},
			})
		}

		c.Locals("user_id", claims.UserID)
		c.Locals("user_email", claims.Email)

		return c.Next()
	}
}

// RequireDeviceAuth middleware verifies an Android device token against phone_credentials.
func RequireDeviceAuth(db *database.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		var tokenStr string

		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
				tokenStr = parts[1]
			}
		}

		// Also check query param ?token= (used by WebSocket handshakes)
		if tokenStr == "" {
			tokenStr = c.Query("token")
		}

		if tokenStr == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    "UNAUTHORIZED",
					"message": "Missing device gateway credential token.",
				},
			})
		}

		tokenHash := HashToken(tokenStr)

		var phoneID uuid.UUID
		var userID uuid.UUID
		var revokedAt *time.Time

		query := `
			SELECT pc.phone_id, p.user_id, pc.revoked_at
			FROM phone_credentials pc
			JOIN phones p ON p.id = pc.phone_id
			WHERE pc.token_hash = $1
			LIMIT 1
		`

		err := db.DB.QueryRowContext(c.Context(), query, tokenHash).Scan(&phoneID, &userID, &revokedAt)
		if err != nil || revokedAt != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    "INVALID_DEVICE_TOKEN",
					"message": "Device token is invalid or has been revoked.",
				},
			})
		}

		// Update last_used_at asynchronously
		go func() {
			_, _ = db.DB.Exec("UPDATE phone_credentials SET last_used_at = NOW() WHERE token_hash = $1", tokenHash)
		}()

		c.Locals("device_phone_id", phoneID)
		c.Locals("device_user_id", userID)

		return c.Next()
	}
}
