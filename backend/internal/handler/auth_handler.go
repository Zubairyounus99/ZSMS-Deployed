package handler

import (
	"log/slog"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"us.ztechai.zsms/backend/internal/auth"
	"us.ztechai.zsms/backend/internal/config"
	"us.ztechai.zsms/backend/internal/repository"
)

type AuthHandler struct {
	userRepo *repository.UserRepository
	cfg      *config.Config
	log      *slog.Logger
}

func NewAuthHandler(userRepo *repository.UserRepository, cfg *config.Config, log *slog.Logger) *AuthHandler {
	return &AuthHandler{
		userRepo: userRepo,
		cfg:      cfg,
		log:      log,
	}
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
}

func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INVALID_BODY",
				"message": "Malformed JSON request body.",
			},
		})
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	if req.Email == "" || !strings.Contains(req.Email, "@") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INVALID_EMAIL",
				"message": "A valid email address is required.",
			},
		})
	}

	if len(req.Password) < 8 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "WEAK_PASSWORD",
				"message": "Password must be at least 8 characters.",
			},
		})
	}

	if strings.TrimSpace(req.FullName) == "" {
		req.FullName = "ZSMS User"
	}

	// Check if already registered
	existing, err := h.userRepo.GetByEmail(c.Context(), req.Email)
	if err != nil {
		h.log.Error("error checking existing user", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{"code": "SERVER_ERROR", "message": "Failed to verify email availability."},
		})
	}
	if existing != nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "EMAIL_EXISTS",
				"message": "An account with this email already exists.",
			},
		})
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{"code": "SERVER_ERROR", "message": "Password encryption failed."},
		})
	}

	user, err := h.userRepo.Create(c.Context(), req.Email, hash, req.FullName)
	if err != nil {
		h.log.Error("failed to create user", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{"code": "SERVER_ERROR", "message": "Failed to create account."},
		})
	}

	token, err := auth.GenerateUserToken(user.ID, user.Email, h.cfg.JWTSecret, h.cfg.JWTExpirationHours)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{"code": "SERVER_ERROR", "message": "Failed to generate session token."},
		})
	}

	c.Cookie(&fiber.Cookie{
		Name:     "zsms_token",
		Value:    token,
		HTTPOnly: true,
		SameSite: "Lax",
	})

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"user":  user,
			"token": token,
		},
	})
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{"code": "INVALID_BODY", "message": "Malformed JSON request body."},
		})
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	user, err := h.userRepo.GetByEmail(c.Context(), req.Email)
	if err != nil || user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INVALID_CREDENTIALS",
				"message": "Invalid email or password.",
			},
		})
	}

	if !auth.CheckPassword(user.PasswordHash, req.Password) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{
				"code":    "INVALID_CREDENTIALS",
				"message": "Invalid email or password.",
			},
		})
	}

	token, err := auth.GenerateUserToken(user.ID, user.Email, h.cfg.JWTSecret, h.cfg.JWTExpirationHours)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{"code": "SERVER_ERROR", "message": "Failed to sign session token."},
		})
	}

	c.Cookie(&fiber.Cookie{
		Name:     "zsms_token",
		Value:    token,
		HTTPOnly: true,
		SameSite: "Lax",
	})

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"user":  user,
			"token": token,
		},
	})
}

func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	c.ClearCookie("zsms_token")
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"message": "Successfully signed out.",
		},
	})
}

func (h *AuthHandler) Me(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{"code": "UNAUTHORIZED", "message": "Unauthorized."},
		})
	}

	user, err := h.userRepo.GetByID(c.Context(), userID)
	if err != nil || user == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error": fiber.Map{"code": "USER_NOT_FOUND", "message": "User profile not found."},
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"user": user,
		},
	})
}
