package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"koran-ai-backend/internal/edition/entity"
	edRepo "koran-ai-backend/internal/edition/repository"
	"koran-ai-backend/internal/edition/service"
)

type mockService struct {
	service.Service
	genEd      *entity.Edition
	genErr     error
	statsTotal int64
	statsLatest string
	statsErr   error
	editions   []entity.Edition
	listTotal  int64
	listErr    error
	details    *entity.EditionDetailResponse
	detailsErr error
}

func (m *mockService) GenerateEdition(ctx context.Context, dateStr string) (*entity.Edition, error) {
	return m.genEd, m.genErr
}

func (m *mockService) GetStats(ctx context.Context) (int64, string, error) {
	return m.statsTotal, m.statsLatest, m.statsErr
}

func (m *mockService) ListEditions(ctx context.Context, page, limit int) ([]entity.Edition, int64, error) {
	return m.editions, m.listTotal, m.listErr
}

func (m *mockService) GetEditionByID(ctx context.Context, id string) (*entity.EditionDetailResponse, error) {
	return m.details, m.detailsErr
}

func TestGenerate_Success(t *testing.T) {
	app := fiber.New()
	id := uuid.New()
	mockSvc := &mockService{
		genEd: &entity.Edition{ID: id},
	}
	hdl := NewHandler(mockSvc, nil)
	app.Post("/internal/editions/generate", hdl.Generate)

	reqBody, _ := json.Marshal(GenerateRequest{Date: "2026-06-09"})
	req := httptest.NewRequest("POST", "/internal/editions/generate", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to test request: %v", err)
	}

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var res map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&res)
	if res["success"] != true || res["edition_id"] != id.String() {
		t.Fatalf("unexpected response body: %v", res)
	}
}

func TestGenerate_BadRequest(t *testing.T) {
	app := fiber.New()
	hdl := NewHandler(&mockService{}, nil)
	app.Post("/internal/editions/generate", hdl.Generate)

	// Missing body
	req := httptest.NewRequest("POST", "/internal/editions/generate", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}

	// Invalid date
	reqBody, _ := json.Marshal(GenerateRequest{Date: "invalid-date"})
	req = httptest.NewRequest("POST", "/internal/editions/generate", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	resp, _ = app.Test(req)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400 for invalid date, got %d", resp.StatusCode)
	}
}

func TestGenerate_Conflicts(t *testing.T) {
	app := fiber.New()
	mockSvc := &mockService{genErr: service.ErrWorkerAlreadyRunning}
	hdl := NewHandler(mockSvc, nil)
	app.Post("/internal/editions/generate", hdl.Generate)

	reqBody, _ := json.Marshal(GenerateRequest{Date: "2026-06-09"})
	req := httptest.NewRequest("POST", "/internal/editions/generate", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusConflict {
		t.Fatalf("expected 409 for lock already held, got %d", resp.StatusCode)
	}

	// Already exists conflict
	mockSvc.genErr = edRepo.ErrAlreadyExists
	resp, _ = app.Test(req)
	if resp.StatusCode != fiber.StatusConflict {
		t.Fatalf("expected 409 for duplicate date, got %d", resp.StatusCode)
	}

	// No summaries
	mockSvc.genErr = service.ErrNoSummariesFound
	resp, _ = app.Test(req)
	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400 for no summaries, got %d", resp.StatusCode)
	}
}

func TestGenerate_InternalError(t *testing.T) {
	app := fiber.New()
	mockSvc := &mockService{genErr: errors.New("db error")}
	hdl := NewHandler(mockSvc, nil)
	app.Post("/internal/editions/generate", hdl.Generate)

	reqBody, _ := json.Marshal(GenerateRequest{Date: "2026-06-09"})
	req := httptest.NewRequest("POST", "/internal/editions/generate", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.StatusCode)
	}
}

func TestStats(t *testing.T) {
	app := fiber.New()
	mockSvc := &mockService{statsTotal: 10, statsLatest: "2026-06-09"}
	hdl := NewHandler(mockSvc, nil)
	app.Get("/internal/editions/stats", hdl.Stats)

	req := httptest.NewRequest("GET", "/internal/editions/stats", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var res map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&res)
	if res["total_editions"] != float64(10) || res["latest_edition"] != "2026-06-09" {
		t.Fatalf("unexpected response: %v", res)
	}
}

func TestList(t *testing.T) {
	app := fiber.New()
	mockSvc := &mockService{
		editions:  []entity.Edition{{Title: "Edition 1"}},
		listTotal: 1,
	}
	hdl := NewHandler(mockSvc, nil)
	app.Get("/api/v1/editions", hdl.List)

	req := httptest.NewRequest("GET", "/api/v1/editions", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestDetail(t *testing.T) {
	app := fiber.New()
	id := uuid.New().String()
	mockSvc := &mockService{
		details: &entity.EditionDetailResponse{Title: "Details"},
	}
	hdl := NewHandler(mockSvc, nil)
	app.Get("/api/v1/editions/:id", hdl.Detail)

	req := httptest.NewRequest("GET", "/api/v1/editions/"+id, nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// Not found
	mockSvc.detailsErr = edRepo.ErrNotFound
	resp, _ = app.Test(req)
	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestLatest(t *testing.T) {
	app := fiber.New()
	id := uuid.New()
	mockSvc := &mockService{
		editions: []entity.Edition{{ID: id}},
		details:  &entity.EditionDetailResponse{Title: "Latest Details"},
	}
	hdl := NewHandler(mockSvc, nil)
	app.Get("/api/v1/editions/latest", hdl.Latest)

	req := httptest.NewRequest("GET", "/api/v1/editions/latest", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// No editions
	mockSvc.editions = nil
	resp, _ = app.Test(req)
	if resp.StatusCode != fiber.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}

	// Latest List error
	mockSvc.listErr = errors.New("list error")
	resp, _ = app.Test(req)
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.StatusCode)
	}
	mockSvc.listErr = nil

	// Latest Details error
	mockSvc.editions = []entity.Edition{{ID: id}}
	mockSvc.detailsErr = errors.New("details error")
	resp, _ = app.Test(req)
	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.StatusCode)
	}
}

func TestStats_Error(t *testing.T) {
	app := fiber.New()
	mockSvc := &mockService{statsErr: errors.New("stats error")}
	hdl := NewHandler(mockSvc, nil)
	app.Get("/internal/editions/stats", hdl.Stats)

	req := httptest.NewRequest("GET", "/internal/editions/stats", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.StatusCode)
	}
}

func TestList_Error(t *testing.T) {
	app := fiber.New()
	mockSvc := &mockService{listErr: errors.New("list error")}
	hdl := NewHandler(mockSvc, nil)
	app.Get("/api/v1/editions", hdl.List)

	req := httptest.NewRequest("GET", "/api/v1/editions", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.StatusCode)
	}
}

func TestList_PaginationLimit(t *testing.T) {
	app := fiber.New()
	mockSvc := &mockService{
		editions:  []entity.Edition{{Title: "Edition 1"}},
		listTotal: 1,
	}
	hdl := NewHandler(mockSvc, nil)
	app.Get("/api/v1/editions", hdl.List)

	// Test page, limit parsing and limit cap at 100
	req := httptest.NewRequest("GET", "/api/v1/editions?page=-1&limit=200", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestDetail_Error(t *testing.T) {
	app := fiber.New()
	id := uuid.New().String()
	mockSvc := &mockService{
		detailsErr: errors.New("db error"),
	}
	hdl := NewHandler(mockSvc, nil)
	app.Get("/api/v1/editions/:id", hdl.Detail)

	req := httptest.NewRequest("GET", "/api/v1/editions/"+id, nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.StatusCode)
	}
}

func TestDetail_EmptyID(t *testing.T) {
	app := fiber.New()
	hdl := NewHandler(&mockService{}, nil)
	app.Get("/api/v1/editions-test", hdl.Detail)

	req := httptest.NewRequest("GET", "/api/v1/editions-test", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestGenerate_InvalidJSON(t *testing.T) {
	app := fiber.New()
	hdl := NewHandler(&mockService{}, nil)
	app.Post("/internal/editions/generate", hdl.Generate)

	req := httptest.NewRequest("POST", "/internal/editions/generate", bytes.NewBufferString("{invalid json"))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestGenerate_EmptyFields(t *testing.T) {
	app := fiber.New()
	hdl := NewHandler(&mockService{}, nil)
	app.Post("/internal/editions/generate", hdl.Generate)

	reqBody, _ := json.Marshal(GenerateRequest{Date: ""})
	req := httptest.NewRequest("POST", "/internal/editions/generate", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}
