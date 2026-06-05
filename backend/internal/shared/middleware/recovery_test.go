package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

// mockLogger tracks log calls.
type mockLogger struct {
	errorCalled bool
	errorMsg    string
	infoCalled  bool
}

func (m *mockLogger) Debug(msg string, fields ...zap.Field) {}
func (m *mockLogger) Info(msg string, fields ...zap.Field)  { m.infoCalled = true }
func (m *mockLogger) Warn(msg string, fields ...zap.Field)  {}
func (m *mockLogger) Error(msg string, fields ...zap.Field) {
	m.errorCalled = true
	m.errorMsg = msg
}
func (m *mockLogger) Fatal(msg string, fields ...zap.Field) {}
func (m *mockLogger) Sync() error                           { return nil }
func (m *mockLogger) GetZapLogger() *zap.Logger            { return zap.NewNop() }

func TestRecovery_NoPanic(t *testing.T) {
	app := fiber.New()
	ml := &mockLogger{}
	app.Use(Recovery(ml))
	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Test request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if ml.errorCalled {
		t.Error("Expected logger.Error not to be called")
	}
}

func TestRecovery_Panic(t *testing.T) {
	app := fiber.New()
	ml := &mockLogger{}
	app.Use(Recovery(ml))
	app.Get("/test-panic", func(c fiber.Ctx) error {
		panic(errors.New("something went wrong"))
	})

	req := httptest.NewRequest("GET", "/test-panic", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Test request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", resp.StatusCode)
	}

	if !ml.errorCalled {
		t.Error("Expected logger.Error to be called on panic")
	}
	if ml.errorMsg != "System panic recovered" {
		t.Errorf("Expected panic message 'System panic recovered', got '%s'", ml.errorMsg)
	}
}
