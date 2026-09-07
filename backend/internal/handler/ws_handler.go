package handler

import (
	"log/slog"
	"strings"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"us.ztechai.zsms/backend/internal/auth"
	"us.ztechai.zsms/backend/internal/database"
	"us.ztechai.zsms/backend/internal/gateway"
)

type GatewayWSHandler struct {
	db  *database.Client
	hub *gateway.Hub
	log *slog.Logger
}

func NewGatewayWSHandler(db *database.Client, hub *gateway.Hub, log *slog.Logger) *GatewayWSHandler {
	return &GatewayWSHandler{
		db:  db,
		hub: hub,
		log: log,
	}
}

// UpgradeCheck validates device token before WebSocket handshake upgrade.
func (h *GatewayWSHandler) UpgradeCheck(c *fiber.Ctx) error {
	if !websocket.IsWebSocketUpgrade(c) {
		return fiber.ErrUpgradeRequired
	}

	authHeader := c.Get("Authorization")
	var tokenStr string
	if len(authHeader) > 7 && strings.EqualFold(authHeader[:7], "Bearer ") {
		tokenStr = strings.TrimSpace(authHeader[7:])
	}
	if tokenStr == "" {
		tokenStr = c.Query("token")
	}

	if tokenStr == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{"code": "UNAUTHORIZED", "message": "Missing device token in WebSocket handshake."},
		})
	}

	tokenHash := auth.HashToken(tokenStr)
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
	err := h.db.DB.QueryRowContext(c.Context(), query, tokenHash).Scan(&phoneID, &userID, &revokedAt)
	if err != nil || revokedAt != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{"code": "INVALID_TOKEN", "message": "Invalid or revoked device token."},
		})
	}

	c.Locals("device_phone_id", phoneID)
	c.Locals("device_user_id", userID)

	return c.Next()
}

// HandleWS manages the persistent WebSocket lifecycle for an Android gateway phone.
func (h *GatewayWSHandler) HandleWS() fiber.Handler {
	return websocket.New(func(conn *websocket.Conn) {
		phoneIDVal := conn.Locals("device_phone_id")
		phoneID, ok := phoneIDVal.(uuid.UUID)
		if !ok {
			_ = conn.Close()
			return
		}

		client := h.hub.Register(phoneID, conn)
		defer h.hub.Unregister(phoneID)

		// Start background goroutine pushing commands to Android
		go client.WritePump(h.log)

		// Read loop: keeps socket alive and processes client pings
		conn.SetReadLimit(65536)
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		conn.SetPongHandler(func(string) error {
			_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			return nil
		})

		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					h.log.Warn("Android WebSocket read error",
						slog.String("phone_id", phoneID.String()),
						slog.String("error", err.Error()),
					)
				}
				break
			}

			// Handle device telemetry or ping packet
			h.log.Debug("received frame from Android device",
				slog.String("phone_id", phoneID.String()),
				slog.String("payload", string(message)),
			)
			_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		}
	})
}
