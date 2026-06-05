package response

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestJSON(t *testing.T) {
	app := fiber.New()
	app.Get("/test-success", func(c fiber.Ctx) error {
		return JSON(c, http.StatusOK, "Retrieval success", fiber.Map{"val": "hello"})
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

	body, _ := io.ReadAll(resp.Body)
	var env SuccessEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("Failed to parse body: %v", err)
	}

	if !env.Success {
		t.Error("Expected Success to be true")
	}
	if env.Message != "Retrieval success" {
		t.Errorf("Expected message 'Retrieval success', got '%s'", env.Message)
	}
	
	dataMap, ok := env.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Data is not a map")
	}
	if dataMap["val"] != "hello" {
		t.Errorf("Expected val 'hello', got '%v'", dataMap["val"])
	}
}

func TestError(t *testing.T) {
	app := fiber.New()
	app.Get("/test-error", func(c fiber.Ctx) error {
		return Error(c, http.StatusBadRequest, "Invalid input", fiber.Map{"field": "required"})
	})

	req := httptest.NewRequest("GET", "/test-error", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Test request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var env ErrorEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("Failed to parse body: %v", err)
	}

	if env.Success {
		t.Error("Expected Success to be false")
	}
	if env.Message != "Invalid input" {
		t.Errorf("Expected message 'Invalid input', got '%s'", env.Message)
	}

	errsMap, ok := env.Errors.(map[string]interface{})
	if !ok {
		t.Fatal("Errors is not a map")
	}
	if errsMap["field"] != "required" {
		t.Errorf("Expected field 'required', got '%v'", errsMap["field"])
	}
}

func TestPaginated(t *testing.T) {
	app := fiber.New()
	app.Get("/test-paginated", func(c fiber.Ctx) error {
		type Meta struct {
			Page int `json:"page"`
		}
		return Paginated(c, http.StatusOK, []string{"a", "b"}, Meta{Page: 1})
	})

	req := httptest.NewRequest("GET", "/test-paginated", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Test request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var env PaginatedEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("Failed to parse body: %v", err)
	}

	if !env.Success {
		t.Error("Expected Success to be true")
	}

	dataSlice, ok := env.Data.([]interface{})
	if !ok {
		t.Fatal("Data is not a slice")
	}
	if len(dataSlice) != 2 || dataSlice[0] != "a" {
		t.Errorf("Expected slice containing 'a', 'b', got %v", env.Data)
	}
}
