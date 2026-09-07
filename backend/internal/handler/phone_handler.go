package handler

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"us.ztechai.zsms/backend/internal/gateway"
	"us.ztechai.zsms/backend/internal/repository"
)

type PhoneHandler struct {
	phoneRepo *repository.PhoneRepository
	hub       *gateway.Hub
	log       *slog.Logger
}

func NewPhoneHandler(phoneRepo *repository.PhoneRepository, hub *gateway.Hub, log *slog.Logger) *PhoneHandler {
	return &PhoneHandler{
		phoneRepo: phoneRepo,
		hub:       hub,
		log:       log,
	}
}

func (h *PhoneHandler) ListPhones(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{"code": "UNAUTHORIZED", "message": "Sign in required."},
		})
	}

	phones, err := h.phoneRepo.ListByUser(c.Context(), userID)
	if err != nil {
		h.log.Error("failed to list phones", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{"code": "SERVER_ERROR", "message": "Failed to retrieve phones."},
		})
	}

	// Enrich with live WebSocket connection state
	for _, p := range phones {
		if h.hub.IsConnected(p.ID) {
			p.Status = "online"
		}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    phones,
	})
}

func (h *PhoneHandler) GetPhone(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{"code": "UNAUTHORIZED", "message": "Sign in required."},
		})
	}

	idParam := c.Params("id")
	phoneID, err := uuid.Parse(idParam)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{"code": "INVALID_ID", "message": "Invalid phone UUID."},
		})
	}

	phone, err := h.phoneRepo.GetByID(c.Context(), phoneID, userID)
	if err != nil || phone == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{"code": "NOT_FOUND", "message": "Phone gateway not found."},
		})
	}

	if h.hub.IsConnected(phone.ID) {
		phone.Status = "online"
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    phone,
	})
}

type HeartbeatRequest struct {
	BatteryLevel    int    `json:"battery_level"`
	BatteryCharging bool   `json:"battery_charging"`
	NetworkType     string `json:"network_type"`
	SignalStrength  *int   `json:"signal_strength"`
	AppVersion      string `json:"app_version"`
	OSVersion       string `json:"os_version"`
}

// Heartbeat is invoked by the Android Gateway background service every 30s.
func (h *PhoneHandler) Heartbeat(c *fiber.Ctx) error {
	devicePhoneID, ok := c.Locals("device_phone_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{"code": "UNAUTHORIZED", "message": "Device authentication required."},
		})
	}

	// Verify route phone_id matches authenticated device
	paramID, err := uuid.Parse(c.Params("id"))
	if err != nil || paramID != devicePhoneID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{"code": "FORBIDDEN", "message": "Cannot heartbeat on behalf of another device."},
		})
	}

	var req HeartbeatRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{"code": "INVALID_BODY", "message": "Malformed heartbeat payload."},
		})
	}

	err = h.phoneRepo.UpdateHeartbeat(
		c.Context(),
		devicePhoneID,
		req.BatteryLevel,
		req.BatteryCharging,
		req.NetworkType,
		req.SignalStrength,
		req.AppVersion,
		req.OSVersion,
	)
	if err != nil {
		h.log.Error("failed to update heartbeat", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{"code": "SERVER_ERROR", "message": "Failed to record heartbeat."},
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"status": "online",
		},
	})
}
