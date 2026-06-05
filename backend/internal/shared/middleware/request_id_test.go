package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestRequestID_GeneratesNewID(t *testing.T) {
	app := fiber.New()
	app.Use(RequestID())
	app.Get("/test", func(c fiber.Ctx) error {
		rid := c.Locals("request_id")
		if rid == nil {
			t.Error("Expected request_id local to be set")
		}
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Test request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.Header.Get("X-Request-ID") == "" {
		t.Error("Expected X-Request-ID response header to be set")
	}
}

func TestRequestID_UsesExistingID(t *testing.T) {
	app := fiber.New()
	app.Use(RequestID())
	app.Get("/test", func(c fiber.Ctx) error {
		rid := c.Locals("request_id")
		if rid.(string) != "existing-uuid" {
			t.Errorf("Expected request_id to be 'existing-uuid', got '%v'", rid)
		}
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", "existing-uuid")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Test request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.Header.Get("X-Request-ID") != "existing-uuid" {
		t.Errorf("Expected X-Request-ID response header to be 'existing-uuid', got '%s'", resp.Header.Get("X-Request-ID"))
	}
}
