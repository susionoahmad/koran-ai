package health

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestHealthCheck_DownState(t *testing.T) {
	app := fiber.New()
	handler := NewHandler(nil, nil)
	app.Get("/health", handler.Check)

	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test health endpoint: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if result["success"].(bool) != false {
		t.Error("Expected success to be false when systems are down")
	}
	if result["message"].(string) != "Service Unavailable" {
		t.Errorf("Expected message 'Service Unavailable', got '%s'", result["message"])
	}

	data := result["data"].(map[string]interface{})
	if data["database"].(string) != "DOWN" {
		t.Errorf("Expected database DOWN, got %s", data["database"])
	}
	if data["redis"].(string) != "DOWN" {
		t.Errorf("Expected redis DOWN, got %s", data["redis"])
	}
}
