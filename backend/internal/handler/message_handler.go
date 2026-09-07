package handler

import (
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"us.ztechai.zsms/backend/internal/gateway"
	"us.ztechai.zsms/backend/internal/repository"
)

type MessageHandler struct {
	msgRepo   *repository.MessageRepository
	phoneRepo *repository.PhoneRepository
	hub       *gateway.Hub
	log       *slog.Logger
}

func NewMessageHandler(
	msgRepo *repository.MessageRepository,
	phoneRepo *repository.PhoneRepository,
	hub *gateway.Hub,
	log *slog.Logger,
) *MessageHandler {
	return &MessageHandler{
		msgRepo:   msgRepo,
		phoneRepo: phoneRepo,
		hub:       hub,
		log:       log,
	}
}

type SendMessageRequest struct {
	PhoneID   uuid.UUID `json:"phone_id"`
	Recipient string    `json:"recipient"`
	Content   string    `json:"content"`
	SimSlot   int       `json:"sim_slot"`
}

// SendMessage creates an outbound SMS job and delivers it to the Android gateway.
func (h *MessageHandler) SendMessage(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{"code": "UNAUTHORIZED", "message": "Sign in required."},
		})
	}

	var req SendMessageRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{"code": "INVALID_BODY", "message": "Malformed JSON request body."},
		})
	}

	req.Recipient = strings.TrimSpace(req.Recipient)
	req.Content = strings.TrimSpace(req.Content)

	if req.Recipient == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{"code": "INVALID_RECIPIENT", "message": "Recipient phone number is required."},
		})
	}
	if req.Content == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{"code": "EMPTY_CONTENT", "message": "Message content cannot be empty."},
		})
	}
	if req.SimSlot <= 0 {
		req.SimSlot = 1
	}

	// Verify phone ownership and existence
	phone, err := h.phoneRepo.GetByID(c.Context(), req.PhoneID, userID)
	if err != nil || phone == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "PHONE_NOT_FOUND",
				"message": "Selected phone gateway does not exist or does not belong to your account.",
			},
		})
	}

	// Check idempotency header if present
	var reqIDPtr *string
	reqIDHeader := c.Get("Idempotency-Key")
	if reqIDHeader == "" {
		reqIDHeader = c.Get("X-Request-ID")
	}
	if reqIDHeader != "" {
		reqIDPtr = &reqIDHeader
	}

	// 1. Insert message into database with initial status 'queued'
	msg, err := h.msgRepo.CreateOutboundMessage(
		c.Context(),
		userID,
		phone.ID,
		phone.PhoneNumber,
		req.Recipient,
		req.Content,
		req.SimSlot,
		reqIDPtr,
	)
	if err != nil {
		h.log.Error("failed to create outbound message", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{"code": "SERVER_ERROR", "message": "Failed to create message record."},
		})
	}

	// 2. Dispatch real-time command to Android gateway device via Hub
	cmd := gateway.Command{
		Type:      "sms.send",
		MessageID: msg.ID,
		RequestID: msg.RequestID,
		Recipient: msg.Recipient,
		Content:   msg.Content,
		SimSlot:   msg.SimSlot,
		Timestamp: time.Now().UTC(),
	}

	err = h.hub.DispatchCommand(phone.ID, cmd)
	if err != nil {
		h.log.Warn("gateway hub dispatch warning", "error", err)
	}

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"success": true,
		"data":    msg,
	})
}

// ListMessages returns paginated messages with optional filters.
func (h *MessageHandler) ListMessages(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{"code": "UNAUTHORIZED", "message": "Sign in required."},
		})
	}

	var phoneIDPtr *uuid.UUID
	if pParam := c.Query("phone_id"); pParam != "" {
		if parsed, err := uuid.Parse(pParam); err == nil {
			phoneIDPtr = &parsed
		}
	}

	direction := c.Query("direction")
	status := c.Query("status")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))

	messages, total, err := h.msgRepo.ListByUser(c.Context(), userID, phoneIDPtr, direction, status, page, limit)
	if err != nil {
		h.log.Error("failed to list messages", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{"code": "SERVER_ERROR", "message": "Failed to retrieve messages."},
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    messages,
		"meta": fiber.Map{
			"page":  page,
			"limit": limit,
			"total": total,
		},
	})
}

func (h *MessageHandler) GetMessage(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{"code": "UNAUTHORIZED", "message": "Sign in required."},
		})
	}

	idParam, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{"code": "INVALID_ID", "message": "Invalid message UUID."},
		})
	}

	msg, err := h.msgRepo.GetByID(c.Context(), idParam, userID)
	if err != nil || msg == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{"code": "NOT_FOUND", "message": "Message not found."},
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    msg,
	})
}

type MessageResultRequest struct {
	Status       string `json:"status"` // "sent", "delivered", "failed"
	ErrorCode    string `json:"error_code"`
	ErrorMessage string `json:"error_message"`
}

// MessageResult is reported by the Android device when SmsManager triggers Sent/Delivery PendingIntents.
func (h *MessageHandler) MessageResult(c *fiber.Ctx) error {
	devicePhoneID, ok := c.Locals("device_phone_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{"code": "UNAUTHORIZED", "message": "Device authentication required."},
		})
	}

	msgID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{"code": "INVALID_ID", "message": "Invalid message ID."},
		})
	}

	var req MessageResultRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{"code": "INVALID_BODY", "message": "Malformed JSON."},
		})
	}

	err = h.msgRepo.UpdateResult(c.Context(), msgID, devicePhoneID, req.Status, req.ErrorCode, req.ErrorMessage)
	if err != nil {
		h.log.Error("failed to update message result", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{"code": "SERVER_ERROR", "message": "Failed to update result."},
		})
	}

	h.log.Info("SMS result recorded from Android gateway",
		slog.String("message_id", msgID.String()),
		slog.String("phone_id", devicePhoneID.String()),
		slog.String("status", req.Status),
	)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"message_id": msgID,
			"status":     req.Status,
		},
	})
}

type InboundMessageRequest struct {
	Sender     string `json:"sender"`
	Recipient  string `json:"recipient"`
	Content    string `json:"content"`
	SimSlot    int    `json:"sim_slot"`
	ReceivedAt string `json:"received_at"`
}

// InboundMessage is invoked by the Android device when SMS_RECEIVED triggers.
func (h *MessageHandler) InboundMessage(c *fiber.Ctx) error {
	devicePhoneID, ok := c.Locals("device_phone_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{"code": "UNAUTHORIZED", "message": "Device authentication required."},
		})
	}

	var req InboundMessageRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{"code": "INVALID_BODY", "message": "Malformed JSON."},
		})
	}

	req.Sender = strings.TrimSpace(req.Sender)
	req.Content = strings.TrimSpace(req.Content)

	if req.Sender == "" || req.Content == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{"code": "INVALID_INBOUND", "message": "Sender and content are required."},
		})
	}

	receivedTime := time.Now()
	if req.ReceivedAt != "" {
		if parsed, err := time.Parse(time.RFC3339, req.ReceivedAt); err == nil {
			receivedTime = parsed
		}
	}

	msg, err := h.msgRepo.IngestInboundMessage(
		c.Context(),
		devicePhoneID,
		req.Sender,
		req.Recipient,
		req.Content,
		req.SimSlot,
		receivedTime,
	)
	if err != nil {
		h.log.Error("failed to ingest inbound message", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{"code": "SERVER_ERROR", "message": "Failed to ingest inbound SMS."},
		})
	}

	h.log.Info("inbound SMS received from Android physical SIM",
		slog.String("phone_id", devicePhoneID.String()),
		slog.String("sender", msg.Sender),
		slog.String("message_id", msg.ID.String()),
	)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    msg,
	})
}

// PollMessages provides an HTTP fallback for Android devices when WebSockets drop.
func (h *MessageHandler) PollMessages(c *fiber.Ctx) error {
	devicePhoneID, ok := c.Locals("device_phone_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{"code": "UNAUTHORIZED", "message": "Device authentication required."},
		})
	}

	cmds := h.hub.PollPending(devicePhoneID)

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    cmds,
	})
}

// ListThreads returns conversational threads for the two-way chat view.
func (h *MessageHandler) ListThreads(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{"code": "UNAUTHORIZED", "message": "Sign in required."},
		})
	}

	var phoneIDPtr *uuid.UUID
	if pParam := c.Query("phone_id"); pParam != "" {
		if parsed, err := uuid.Parse(pParam); err == nil {
			phoneIDPtr = &parsed
		}
	}

	threads, err := h.msgRepo.ListThreadsByUser(c.Context(), userID, phoneIDPtr)
	if err != nil {
		h.log.Error("failed to list threads", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{"code": "SERVER_ERROR", "message": "Failed to retrieve threads."},
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    threads,
	})
}

// GetThreadMessages returns messages in a thread for two-way chat.
func (h *MessageHandler) GetThreadMessages(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{"code": "UNAUTHORIZED", "message": "Sign in required."},
		})
	}

	threadID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{"code": "INVALID_ID", "message": "Invalid thread UUID."},
		})
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))

	messages, err := h.msgRepo.GetThreadMessages(c.Context(), threadID, userID, page, limit)
	if err != nil {
		h.log.Error("failed to get thread messages", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{"code": "SERVER_ERROR", "message": "Failed to load thread messages."},
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    messages,
	})
}

// GetDashboardStats aggregates actual message volumes for the dashboard.
func (h *MessageHandler) GetDashboardStats(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{"code": "UNAUTHORIZED", "message": "Sign in required."},
		})
	}

	stats, err := h.msgRepo.GetDashboardStats(c.Context(), userID)
	if err != nil {
		h.log.Error("failed to get dashboard stats", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{"code": "SERVER_ERROR", "message": "Failed to retrieve statistics."},
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    stats,
	})
}
