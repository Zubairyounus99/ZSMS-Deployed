package tests

import (
	"os"
	"testing"

	"us.ztechai.zsms/backend/internal/config"
)

func TestConfigLoadDefaults(t *testing.T) {
	os.Clearenv()

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("expected config.Load() to succeed with defaults, got: %v", err)
	}

	if cfg.Port != 8080 {
		t.Errorf("expected default Port to be 8080, got %d", cfg.Port)
	}

	if cfg.AppName != "ZSMS" {
		t.Errorf("expected default AppName to be 'ZSMS', got %s", cfg.AppName)
	}

	if cfg.AppEnv != "development" {
		t.Errorf("expected default AppEnv to be 'development', got %s", cfg.AppEnv)
	}

	if cfg.StorageProvider != "minio" {
		t.Errorf("expected default StorageProvider to be 'minio', got %s", cfg.StorageProvider)
	}
}

func TestConfigMaskedSummary(t *testing.T) {
	cfg := &config.Config{
		AppEnv:        "development",
		AppName:       "ZSMS",
		JWTSecret:     "super_secret_jwt_key_12345",
		DatabaseURL:   "postgres://user:secretpass@localhost:5432/db",
		EncryptionKey: "32bytesecretforaes256payload",
	}

	summary := cfg.MaskedSummary()

	if summary["jwt_secret"] != "[REDACTED]" {
		t.Errorf("expected jwt_secret to be redacted, got: %v", summary["jwt_secret"])
	}

	if summary["db_password"] != "[REDACTED]" {
		t.Errorf("expected db_password to be redacted, got: %v", summary["db_password"])
	}

	if summary["encryption_key"] != "[REDACTED]" {
		t.Errorf("expected encryption_key to be redacted, got: %v", summary["encryption_key"])
	}
}

func TestConfigProductionValidation(t *testing.T) {
	cfg := &config.Config{
		AppEnv:      "production",
		Port:        8080,
		JWTSecret:   "dev_default_secret",
		DatabaseURL: "",
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatalf("expected production config with dev JWT and empty DB to fail validation")
	}
}
