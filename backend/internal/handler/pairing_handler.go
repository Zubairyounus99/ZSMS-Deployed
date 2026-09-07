package handler

import (
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"us.ztechai.zsms/backend/internal/repository"
)

type PairingHandler struct {
	phoneRepo *repository.PhoneRepository
	log       *slog.Logger
}

func NewPairingHandler(phoneRepo *repository.PhoneRepository, log *slog.Logger) *PairingHandler {
	return &PairingHandler{
		phoneRepo: phoneRepo,
		log:       log,
	}
}

// CreateSession generates a new 6-digit pairing code for the authenticated user.
func (h *PairingHandler) CreateSession(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{"code": "UNAUTHORIZED", "message": "Sign in required."},
		})
	}

	session, err := h.phoneRepo.CreatePairingSession(c.Context(), userID)
	if err != nil {
		h.log.Error("failed to create pairing session", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{"code": "SERVER_ERROR", "message": "Failed to generate pairing session."},
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"session_id":    session.ID,
			"pairing_code":  session.PairingCode,
			"session_token": session.SessionToken,
			"expires_at":    session.ExpiresAt,
			"expires_in_s":  600,
		},
	})
}

type CompletePairingRequest struct {
	PairingCode      string `json:"pairing_code"`
	DeviceName       string `json:"device_name"`
	DeviceIdentifier string `json:"device_identifier"`
	PhoneNumber      string `json:"phone_number"`
	SimCarrier       string `json:"sim_carrier"`
}

// CompletePairing is invoked by the Android application to claim the code and receive credentials.
func (h *PairingHandler) CompletePairing(c *fiber.Ctx) error {
	var req CompletePairingRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{"code": "INVALID_BODY", "message": "Malformed JSON."},
		})
	}

	req.PairingCode = strings.TrimSpace(req.PairingCode)
	if len(req.PairingCode) != 6 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{"code": "INVALID_CODE", "message": "Pairing code must be 6 digits."},
		})
	}

	if req.DeviceName == "" {
		req.DeviceName = "Android Gateway"
	}
	if req.DeviceIdentifier == "" {
		req.DeviceIdentifier = uuid.New().String()
	}

	phone, deviceToken, err := h.phoneRepo.ClaimPairingSession(
		c.Context(),
		req.PairingCode,
		req.DeviceName,
		req.DeviceIdentifier,
		req.PhoneNumber,
		req.SimCarrier,
	)
	if err != nil {
		h.log.Warn("pairing claim failed",
			slog.String("code", req.PairingCode),
			slog.String("error", err.Error()),
		)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "PAIRING_FAILED",
				"message": err.Error(),
			},
		})
	}

	h.log.Info("phone successfully paired with Android gateway",
		slog.String("phone_id", phone.ID.String()),
		slog.String("device_name", phone.Name),
		slog.String("user_id", phone.UserID.String()),
	)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"phone_id":     phone.ID,
			"phone_name":   phone.Name,
			"device_token": deviceToken, // Plaintext returned ONLY ONCE!
			"status":       phone.Status,
		},
	})
}
