package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"us.ztechai.zsms/backend/internal/auth"
)

func TestPasswordHashing(t *testing.T) {
	password := "SecretP@ssw0rd123"

	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	if hash == password {
		t.Fatalf("hash must not match plaintext password")
	}

	if !auth.CheckPassword(hash, password) {
		t.Errorf("CheckPassword returned false for correct password")
	}

	if auth.CheckPassword(hash, "WrongPassword") {
		t.Errorf("CheckPassword returned true for incorrect password")
	}
}

func TestJWTGenerationAndValidation(t *testing.T) {
	userID := uuid.New()
	email := "test@ztechai.us"
	secret := "test_secret_for_unit_tests_only_do_not_use_in_production"

	token, err := auth.GenerateUserToken(userID, email, secret, 2)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	claims, err := auth.ValidateUserToken(token, secret)
	if err != nil {
		t.Fatalf("failed to validate valid token: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("expected userID %s, got %s", userID, claims.UserID)
	}
	if claims.Email != email {
		t.Errorf("expected email %s, got %s", email, claims.Email)
	}

	// Validate with incorrect secret fails
	_, err = auth.ValidateUserToken(token, "wrong_secret")
	if err == nil {
		t.Errorf("expected validation to fail with wrong secret")
	}
}

func TestTokenHashing(t *testing.T) {
	token := "zsms_dev_1234567890abcdef"
	hash1 := auth.HashToken(token)
	hash2 := auth.HashToken(token)

	if hash1 != hash2 {
		t.Errorf("hash must be deterministic")
	}

	if len(hash1) != 64 {
		t.Errorf("SHA-256 hex string should be 64 characters, got %d", len(hash1))
	}

	if auth.HashToken("different_token") == hash1 {
		t.Errorf("different tokens must have different hashes")
	}
}

func TestRequireUserAuthBypass(t *testing.T) {
	app := fiber.New()
	secret := "test_secret_for_bypass_mode"

	app.Get("/test-bypass", auth.RequireUserAuth(secret, true, nil), func(c *fiber.Ctx) error {
		userID := c.Locals("user_id").(uuid.UUID)
		userEmail := c.Locals("user_email").(string)
		return c.JSON(fiber.Map{
			"user_id":    userID.String(),
			"user_email": userEmail,
		})
	})

	// 1. Request with no auth header in bypass mode -> must succeed with 200 and default test user
	req := httptest.NewRequest("GET", "/test-bypass", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 in bypass mode, got: %d", resp.StatusCode)
	}

	// 2. Request with invalid auth header in bypass mode -> must still succeed with 200
	reqInvalid := httptest.NewRequest("GET", "/test-bypass", nil)
	reqInvalid.Header.Set("Authorization", "Bearer invalid_gibberish_token")
	respInvalid, err := app.Test(reqInvalid)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	if respInvalid.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 in bypass mode with invalid token, got: %d", respInvalid.StatusCode)
	}
}

func TestRequireUserAuthEnforced(t *testing.T) {
	app := fiber.New()
	secret := "test_secret_for_enforced_mode"

	app.Get("/test-enforced", auth.RequireUserAuth(secret, false, nil), func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	// 1. Request with no auth header in enforced mode -> must fail with 401
	req := httptest.NewRequest("GET", "/test-enforced", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status 401 in enforced mode without token, got: %d", resp.StatusCode)
	}

	// 2. Request with invalid token in enforced mode -> must fail with 401
	reqInvalid := httptest.NewRequest("GET", "/test-enforced", nil)
	reqInvalid.Header.Set("Authorization", "Bearer bad_token")
	respInvalid, err := app.Test(reqInvalid)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	if respInvalid.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status 401 in enforced mode with bad token, got: %d", respInvalid.StatusCode)
	}

	// 3. Request with valid token in enforced mode -> must succeed with 200
	validUserID := uuid.New()
	validToken, err := auth.GenerateUserToken(validUserID, "user@ztechai.us", secret, 1)
	if err != nil {
		t.Fatalf("failed to generate valid token: %v", err)
	}

	reqValid := httptest.NewRequest("GET", "/test-enforced", nil)
	reqValid.Header.Set("Authorization", "Bearer "+validToken)
	respValid, err := app.Test(reqValid)
	if err != nil {
		t.Fatalf("app.Test failed: %v", err)
	}
	if respValid.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 in enforced mode with valid token, got: %d", respValid.StatusCode)
	}
}
