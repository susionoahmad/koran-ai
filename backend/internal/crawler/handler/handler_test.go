package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"koran-ai-backend/internal/crawler/handler"
	"koran-ai-backend/internal/crawler/service"
)

type mockCrawlerService struct {
	RunSourceFunc     func(ctx context.Context, sourceID string) (*service.CrawlResult, error)
	RunAllSourcesFunc func(ctx context.Context) ([]*service.CrawlResult, error)
	GetStatsFunc      func(ctx context.Context) (*service.CrawlerStats, error)
}

func (m *mockCrawlerService) RunSource(ctx context.Context, sourceID string) (*service.CrawlResult, error) {
	if m.RunSourceFunc != nil {
		return m.RunSourceFunc(ctx, sourceID)
	}
	return nil, nil
}

func (m *mockCrawlerService) RunAllSources(ctx context.Context) ([]*service.CrawlResult, error) {
	if m.RunAllSourcesFunc != nil {
		return m.RunAllSourcesFunc(ctx)
	}
	return nil, nil
}

func (m *mockCrawlerService) GetStats(ctx context.Context) (*service.CrawlerStats, error) {
	if m.GetStatsFunc != nil {
		return m.GetStatsFunc(ctx)
	}
	return nil, nil
}

func TestHandler_RunSource_Success(t *testing.T) {
	app := fiber.New()
	mockSvc := &mockCrawlerService{
		RunSourceFunc: func(ctx context.Context, sourceID string) (*service.CrawlResult, error) {
			return &service.CrawlResult{
				SourceID:      sourceID,
				ArticlesFound: 10,
				ArticlesSaved: 5,
			}, nil
		},
	}

	h := handler.NewHandler(mockSvc)
	app.Post("/internal/crawler/run/:id", h.RunSource)

	req := httptest.NewRequest("POST", "/internal/crawler/run/some-uuid", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to test handler: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var envelope map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if envelope["success"] != true {
		t.Errorf("expected success=true, got %v", envelope["success"])
	}

	data := envelope["data"].(map[string]interface{})
	if data["source_id"] != "some-uuid" {
		t.Errorf("expected source_id 'some-uuid', got %v", data["source_id"])
	}
}

func TestHandler_RunSource_Inactive(t *testing.T) {
	app := fiber.New()
	mockSvc := &mockCrawlerService{
		RunSourceFunc: func(ctx context.Context, sourceID string) (*service.CrawlResult, error) {
			return nil, service.ErrSourceInactive
		},
	}

	h := handler.NewHandler(mockSvc)
	app.Post("/internal/crawler/run/:id", h.RunSource)

	req := httptest.NewRequest("POST", "/internal/crawler/run/some-uuid", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}
}

func TestHandler_RunSource_NoRSS(t *testing.T) {
	app := fiber.New()
	mockSvc := &mockCrawlerService{
		RunSourceFunc: func(ctx context.Context, sourceID string) (*service.CrawlResult, error) {
			return nil, service.ErrSourceNoRSS
		},
	}

	h := handler.NewHandler(mockSvc)
	app.Post("/internal/crawler/run/:id", h.RunSource)

	req := httptest.NewRequest("POST", "/internal/crawler/run/some-uuid", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", resp.StatusCode)
	}
}

func TestHandler_RunSource_InternalError(t *testing.T) {
	app := fiber.New()
	mockSvc := &mockCrawlerService{
		RunSourceFunc: func(ctx context.Context, sourceID string) (*service.CrawlResult, error) {
			return nil, errors.New("db generic error")
		},
	}

	h := handler.NewHandler(mockSvc)
	app.Post("/internal/crawler/run/:id", h.RunSource)

	req := httptest.NewRequest("POST", "/internal/crawler/run/some-uuid", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", resp.StatusCode)
	}
}

func TestHandler_RunAll_Success(t *testing.T) {
	app := fiber.New()
	mockSvc := &mockCrawlerService{
		RunAllSourcesFunc: func(ctx context.Context) ([]*service.CrawlResult, error) {
			return []*service.CrawlResult{
				{SourceID: "src-1", ArticlesFound: 10, ArticlesSaved: 8},
				{SourceID: "src-2", ArticlesFound: 5, ArticlesSaved: 3, Error: errors.New("partial error")},
			}, nil
		},
	}

	h := handler.NewHandler(mockSvc)
	app.Post("/internal/crawler/run-all", h.RunAll)

	req := httptest.NewRequest("POST", "/internal/crawler/run-all", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestHandler_RunAll_Error(t *testing.T) {
	app := fiber.New()
	mockSvc := &mockCrawlerService{
		RunAllSourcesFunc: func(ctx context.Context) ([]*service.CrawlResult, error) {
			return nil, errors.New("overall failure")
		},
	}

	h := handler.NewHandler(mockSvc)
	app.Post("/internal/crawler/run-all", h.RunAll)

	req := httptest.NewRequest("POST", "/internal/crawler/run-all", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", resp.StatusCode)
	}
}

func TestHandler_GetStats_Success(t *testing.T) {
	app := fiber.New()
	mockSvc := &mockCrawlerService{
		GetStatsFunc: func(ctx context.Context) (*service.CrawlerStats, error) {
			return &service.CrawlerStats{
				Sources:       5,
				ArticlesTotal: 1000,
				ArticlesToday: 42,
				LastCrawl:     "2026-06-05T10:00:00Z",
				FailedCrawls:  2,
			}, nil
		},
	}

	h := handler.NewHandler(mockSvc)
	app.Get("/internal/crawler/stats", h.GetStats)

	req := httptest.NewRequest("GET", "/internal/crawler/stats", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var envelope map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if envelope["success"] != true {
		t.Errorf("expected success=true, got %v", envelope["success"])
	}

	data, ok := envelope["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected data to be a map, got %T", envelope["data"])
	}

	if data["sources"].(float64) != 5 {
		t.Errorf("expected sources=5, got %v", data["sources"])
	}
	if data["articles_total"].(float64) != 1000 {
		t.Errorf("expected articles_total=1000, got %v", data["articles_total"])
	}
	if data["articles_today"].(float64) != 42 {
		t.Errorf("expected articles_today=42, got %v", data["articles_today"])
	}
}

func TestHandler_GetStats_Error(t *testing.T) {
	app := fiber.New()
	mockSvc := &mockCrawlerService{
		GetStatsFunc: func(ctx context.Context) (*service.CrawlerStats, error) {
			return nil, errors.New("db connection lost")
		},
	}

	h := handler.NewHandler(mockSvc)
	app.Get("/internal/crawler/stats", h.GetStats)

	req := httptest.NewRequest("GET", "/internal/crawler/stats", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to test: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", resp.StatusCode)
	}
}
