package tests

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"us.ztechai.zsms/backend/internal/health"
)

func TestHealthEndpoints(t *testing.T) {
	app := fiber.New()
	handler := health.NewHandler("zsms-api-test", nil, nil)

	app.Get("/health", handler.Health)
	app.Get("/health/live", handler.Live)
	app.Get("/health/ready", handler.Ready)

	// Test 1: GET /health
	req1 := httptest.NewRequest("GET", "/health", nil)
	resp1, err := app.Test(req1)
	if err != nil {
		t.Fatalf("failed to test /health: %v", err)
	}
	if resp1.StatusCode != fiber.StatusOK {
		t.Errorf("expected /health status 200, got %d", resp1.StatusCode)
	}

	body1, _ := io.ReadAll(resp1.Body)
	var status1 health.Status
	if err := json.Unmarshal(body1, &status1); err != nil {
		t.Fatalf("failed to unmarshal /health JSON: %v", err)
	}
	if status1.Status != "ok" || status1.Service != "zsms-api-test" {
		t.Errorf("unexpected /health status: %+v", status1)
	}

	// Test 2: GET /health/live
	req2 := httptest.NewRequest("GET", "/health/live", nil)
	resp2, err := app.Test(req2)
	if err != nil {
		t.Fatalf("failed to test /health/live: %v", err)
	}
	if resp2.StatusCode != fiber.StatusOK {
		t.Errorf("expected /health/live status 200, got %d", resp2.StatusCode)
	}
}
