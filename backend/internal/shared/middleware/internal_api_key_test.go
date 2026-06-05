package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestInternalAPIKey_ValidKey(t *testing.T) {
	app := fiber.New()
	apiKey := "secret-key-123"
	app.Use(InternalAPIKey(apiKey))
	app.Get("/internal/test", func(c fiber.Ctx) error {
		return c.SendString("success")
	})

	req := httptest.NewRequest("GET", "/internal/test", nil)
	req.Header.Set("X-Internal-Key", apiKey)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Test request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestInternalAPIKey_InvalidKey(t *testing.T) {
	app := fiber.New()
	apiKey := "secret-key-123"
	app.Use(InternalAPIKey(apiKey))
	app.Get("/internal/test", func(c fiber.Ctx) error {
		return c.SendString("success")
	})

	req := httptest.NewRequest("GET", "/internal/test", nil)
	req.Header.Set("X-Internal-Key", "wrong-key")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Test request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestInternalAPIKey_MissingKey(t *testing.T) {
	app := fiber.New()
	apiKey := "secret-key-123"
	app.Use(InternalAPIKey(apiKey))
	app.Get("/internal/test", func(c fiber.Ctx) error {
		return c.SendString("success")
	})

	req := httptest.NewRequest("GET", "/internal/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Test request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}
