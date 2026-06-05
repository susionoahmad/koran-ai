package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"koran-ai-backend/internal/shared/validator"
	"koran-ai-backend/internal/source/dto"
	"koran-ai-backend/internal/source/service"
)

type mockService struct {
	CreateFunc  func(ctx context.Context, req dto.CreateSourceRequest) (*dto.SourceResponse, error)
	GetByIDFunc func(ctx context.Context, id string) (*dto.SourceResponse, error)
	ListFunc    func(ctx context.Context, page, limit int) (*dto.SourceListResponse, error)
	UpdateFunc  func(ctx context.Context, id string, req dto.UpdateSourceRequest) (*dto.SourceResponse, error)
	DeleteFunc  func(ctx context.Context, id string) error
}

func (m *mockService) Create(ctx context.Context, req dto.CreateSourceRequest) (*dto.SourceResponse, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, req)
	}
	return nil, nil
}

func (m *mockService) GetByID(ctx context.Context, id string) (*dto.SourceResponse, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockService) List(ctx context.Context, page, limit int) (*dto.SourceListResponse, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, page, limit)
	}
	return nil, nil
}

func (m *mockService) Update(ctx context.Context, id string, req dto.UpdateSourceRequest) (*dto.SourceResponse, error) {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, id, req)
	}
	return nil, nil
}

func (m *mockService) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

func TestCreate_Handler_Success(t *testing.T) {
	app := fiber.New()
	mockSvc := &mockService{
		CreateFunc: func(ctx context.Context, req dto.CreateSourceRequest) (*dto.SourceResponse, error) {
			return &dto.SourceResponse{
				ID:         "123",
				Name:       req.Name,
				BaseURL:    req.BaseURL,
				RSSURL:     req.RSSURL,
				SourceType: req.SourceType,
				IsActive:   true,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}, nil
		},
	}

	h := NewHandler(mockSvc, validator.NewValidator())
	app.Post("/sources", h.Create)

	body := dto.CreateSourceRequest{
		Name:       "Test",
		BaseURL:    "https://test.com",
		RSSURL:     "https://test.com/rss",
		SourceType: "RSS",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/sources", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test handler: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}
}

func TestCreate_Handler_ValidationFailure(t *testing.T) {
	app := fiber.New()
	h := NewHandler(&mockService{}, validator.NewValidator())
	app.Post("/sources", h.Create)

	body := dto.CreateSourceRequest{
		Name:       "", // empty name violates validation
		BaseURL:    "invalid-url",
		RSSURL:     "https://test.com/rss",
		SourceType: "INVALID",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/sources", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test handler: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}
}

func TestGetByID_Handler_Success(t *testing.T) {
	app := fiber.New()
	mockSvc := &mockService{
		GetByIDFunc: func(ctx context.Context, id string) (*dto.SourceResponse, error) {
			return &dto.SourceResponse{
				ID:   id,
				Name: "Test Source",
			}, nil
		},
	}

	h := NewHandler(mockSvc, validator.NewValidator())
	app.Get("/sources/:id", h.GetByID)

	req := httptest.NewRequest("GET", "/sources/123", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test handler: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestGetByID_Handler_NotFound(t *testing.T) {
	app := fiber.New()
	mockSvc := &mockService{
		GetByIDFunc: func(ctx context.Context, id string) (*dto.SourceResponse, error) {
			return nil, service.ErrNotFound
		},
	}

	h := NewHandler(mockSvc, validator.NewValidator())
	app.Get("/sources/:id", h.GetByID)

	req := httptest.NewRequest("GET", "/sources/123", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test handler: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}
}

func TestList_Handler_Success(t *testing.T) {
	app := fiber.New()
	mockSvc := &mockService{
		ListFunc: func(ctx context.Context, page, limit int) (*dto.SourceListResponse, error) {
			return &dto.SourceListResponse{
				Sources: []dto.SourceResponse{
					{ID: "1", Name: "Source 1"},
				},
				Total: 1,
				Page:  page,
				Limit: limit,
			}, nil
		},
	}

	h := NewHandler(mockSvc, validator.NewValidator())
	app.Get("/sources", h.List)

	req := httptest.NewRequest("GET", "/sources?page=1&limit=5", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test handler: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestUpdate_Handler_Success(t *testing.T) {
	app := fiber.New()
	mockSvc := &mockService{
		UpdateFunc: func(ctx context.Context, id string, req dto.UpdateSourceRequest) (*dto.SourceResponse, error) {
			return &dto.SourceResponse{
				ID:   id,
				Name: req.Name,
			}, nil
		},
	}

	h := NewHandler(mockSvc, validator.NewValidator())
	app.Put("/sources/:id", h.Update)

	isActive := true
	body := dto.UpdateSourceRequest{
		Name:       "Updated Name",
		BaseURL:    "https://test.com",
		RSSURL:     "https://test.com/rss",
		SourceType: "RSS",
		IsActive:   &isActive,
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("PUT", "/sources/123", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test handler: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestDelete_Handler_Success(t *testing.T) {
	app := fiber.New()
	mockSvc := &mockService{
		DeleteFunc: func(ctx context.Context, id string) error {
			return nil
		},
	}

	h := NewHandler(mockSvc, validator.NewValidator())
	app.Delete("/sources/:id", h.Delete)

	req := httptest.NewRequest("DELETE", "/sources/123", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test handler: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestDelete_Handler_NotFound(t *testing.T) {
	app := fiber.New()
	mockSvc := &mockService{
		DeleteFunc: func(ctx context.Context, id string) error {
			return service.ErrNotFound
		},
	}

	h := NewHandler(mockSvc, validator.NewValidator())
	app.Delete("/sources/:id", h.Delete)

	req := httptest.NewRequest("DELETE", "/sources/123", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test handler: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}
}

func TestCreate_Handler_Conflict(t *testing.T) {
	app := fiber.New()
	mockSvc := &mockService{
		CreateFunc: func(ctx context.Context, req dto.CreateSourceRequest) (*dto.SourceResponse, error) {
			return nil, service.ErrDuplicateName
		},
	}

	h := NewHandler(mockSvc, validator.NewValidator())
	app.Post("/sources", h.Create)

	body := dto.CreateSourceRequest{
		Name:       "Duplicate Name",
		BaseURL:    "https://test.com",
		RSSURL:     "https://test.com/rss",
		SourceType: "RSS",
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/sources", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test handler: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("Expected status 409, got %d", resp.StatusCode)
	}
}

func TestGetByID_Handler_Error(t *testing.T) {
	app := fiber.New()
	mockSvc := &mockService{
		GetByIDFunc: func(ctx context.Context, id string) (*dto.SourceResponse, error) {
			return nil, errors.New("db error")
		},
	}

	h := NewHandler(mockSvc, validator.NewValidator())
	app.Get("/sources/:id", h.GetByID)

	req := httptest.NewRequest("GET", "/sources/123", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test handler: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", resp.StatusCode)
	}
}

func TestUpdate_Handler_NotFound(t *testing.T) {
	app := fiber.New()
	mockSvc := &mockService{
		UpdateFunc: func(ctx context.Context, id string, req dto.UpdateSourceRequest) (*dto.SourceResponse, error) {
			return nil, service.ErrNotFound
		},
	}

	h := NewHandler(mockSvc, validator.NewValidator())
	app.Put("/sources/:id", h.Update)

	isActive := true
	body := dto.UpdateSourceRequest{
		Name:       "Updated Name",
		BaseURL:    "https://test.com",
		RSSURL:     "https://test.com/rss",
		SourceType: "RSS",
		IsActive:   &isActive,
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("PUT", "/sources/123", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test handler: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}
}

func TestUpdate_Handler_Conflict(t *testing.T) {
	app := fiber.New()
	mockSvc := &mockService{
		UpdateFunc: func(ctx context.Context, id string, req dto.UpdateSourceRequest) (*dto.SourceResponse, error) {
			return nil, service.ErrDuplicateName
		},
	}

	h := NewHandler(mockSvc, validator.NewValidator())
	app.Put("/sources/:id", h.Update)

	isActive := true
	body := dto.UpdateSourceRequest{
		Name:       "Duplicate Name",
		BaseURL:    "https://test.com",
		RSSURL:     "https://test.com/rss",
		SourceType: "RSS",
		IsActive:   &isActive,
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("PUT", "/sources/123", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test handler: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("Expected status 409, got %d", resp.StatusCode)
	}
}

func TestUpdate_Handler_GenericError(t *testing.T) {
	app := fiber.New()
	mockSvc := &mockService{
		UpdateFunc: func(ctx context.Context, id string, req dto.UpdateSourceRequest) (*dto.SourceResponse, error) {
			return nil, errors.New("something went wrong")
		},
	}

	h := NewHandler(mockSvc, validator.NewValidator())
	app.Put("/sources/:id", h.Update)

	isActive := true
	body := dto.UpdateSourceRequest{
		Name:       "New Name",
		BaseURL:    "https://test.com",
		RSSURL:     "https://test.com/rss",
		SourceType: "RSS",
		IsActive:   &isActive,
	}
	jsonBody, _ := json.Marshal(body)

	req := httptest.NewRequest("PUT", "/sources/123", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to test handler: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", resp.StatusCode)
	}
}

