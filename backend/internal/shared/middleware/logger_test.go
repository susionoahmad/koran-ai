package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestLogger_SuccessRequest(t *testing.T) {
	app := fiber.New()
	ml := &mockLogger{}
	app.Use(Logger(ml))
	app.Get("/test-success", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test-success", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Test request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if !ml.infoCalled {
		t.Error("Expected logger.Info to be called on successful request")
	}
}

func TestLogger_FailedRequest(t *testing.T) {
	app := fiber.New()
	ml := &mockLogger{}
	app.Use(Logger(ml))
	app.Get("/test-fail", func(c fiber.Ctx) error {
		return errors.New("custom test error")
	})

	req := httptest.NewRequest("GET", "/test-fail", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Test request failed: %v", err)
	}
	defer resp.Body.Close()

	if !ml.errorCalled {
		t.Error("Expected logger.Error to be called on failed request")
	}
}
