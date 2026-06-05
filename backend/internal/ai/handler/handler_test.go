package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"koran-ai-backend/internal/ai/service"
)

type mockService struct {
	service.Service
	processFunc  func(ctx context.Context, limit int) (*service.ProcessResult, error)
	getStatsFunc func(ctx context.Context) (*service.AIStats, error)
}

func (m *mockService) Process(ctx context.Context, limit int) (*service.ProcessResult, error) {
	return m.processFunc(ctx, limit)
}

func (m *mockService) GetStats(ctx context.Context) (*service.AIStats, error) {
	return m.getStatsFunc(ctx)
}

func TestHandler_Process_Success(t *testing.T) {
	app := fiber.New()
	svc := &mockService{
		processFunc: func(ctx context.Context, limit int) (*service.ProcessResult, error) {
			if limit != 25 {
				t.Errorf("expected limit 25, got %d", limit)
			}
			return &service.ProcessResult{Processed: 20, Failed: 5}, nil
		},
	}
	h := NewHandler(svc)
	app.Post("/internal/ai/process", h.Process)

	reqBody, _ := json.Marshal(ProcessRequest{Limit: 25})
	req := httptest.NewRequest("POST", "/internal/ai/process", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to run test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if result["success"].(bool) != true {
		t.Errorf("expected success=true, got %v", result["success"])
	}

	data := result["data"].(map[string]interface{})
	if int(data["processed"].(float64)) != 20 || int(data["failed"].(float64)) != 5 {
		t.Errorf("expected processed=20, failed=5, got processed=%v, failed=%v", data["processed"], data["failed"])
	}
}

func TestHandler_Process_Conflict(t *testing.T) {
	app := fiber.New()
	svc := &mockService{
		processFunc: func(ctx context.Context, limit int) (*service.ProcessResult, error) {
			return nil, errors.New("worker already running")
		},
	}
	h := NewHandler(svc)
	app.Post("/internal/ai/process", h.Process)

	req := httptest.NewRequest("POST", "/internal/ai/process", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to run test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("expected status 409, got %d", resp.StatusCode)
	}
}

func TestHandler_Process_ServerError(t *testing.T) {
	app := fiber.New()
	svc := &mockService{
		processFunc: func(ctx context.Context, limit int) (*service.ProcessResult, error) {
			return nil, errors.New("something went wrong")
		},
	}
	h := NewHandler(svc)
	app.Post("/internal/ai/process", h.Process)

	req := httptest.NewRequest("POST", "/internal/ai/process", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to run test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", resp.StatusCode)
	}
}

func TestHandler_GetStats_Success(t *testing.T) {
	app := fiber.New()
	svc := &mockService{
		getStatsFunc: func(ctx context.Context) (*service.AIStats, error) {
			return &service.AIStats{Processed: 100, Pending: 10, Failed: 2}, nil
		},
	}
	h := NewHandler(svc)
	app.Get("/internal/ai/stats", h.GetStats)

	req := httptest.NewRequest("GET", "/internal/ai/stats", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to run test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	body, _ := io.ReadAll(resp.Body)
	_ = json.Unmarshal(body, &result)

	data := result["data"].(map[string]interface{})
	if int(data["processed"].(float64)) != 100 || int(data["pending"].(float64)) != 10 || int(data["failed"].(float64)) != 2 {
		t.Errorf("unexpected stats: %v", data)
	}
}

func TestHandler_GetStats_Error(t *testing.T) {
	app := fiber.New()
	svc := &mockService{
		getStatsFunc: func(ctx context.Context) (*service.AIStats, error) {
			return nil, errors.New("db error")
		},
	}
	h := NewHandler(svc)
	app.Get("/internal/ai/stats", h.GetStats)

	req := httptest.NewRequest("GET", "/internal/ai/stats", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to run test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", resp.StatusCode)
	}
}
