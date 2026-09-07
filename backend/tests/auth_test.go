package tests

import (
	"testing"

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
