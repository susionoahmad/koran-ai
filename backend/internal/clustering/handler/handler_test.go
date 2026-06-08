package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	clusterEntity "koran-ai-backend/internal/clustering/entity"
	clusterRepo "koran-ai-backend/internal/clustering/repository"
	"koran-ai-backend/internal/clustering/service"
)

type mockService struct {
	runFunc    func(ctx context.Context) (*service.ClusteringResult, error)
	statsFunc  func(ctx context.Context) (*service.ClusteringStats, error)
	listFunc   func(ctx context.Context, page int, limit int) ([]clusterEntity.Cluster, int64, error)
	detailFunc func(ctx context.Context, id string) (*clusterEntity.Cluster, error)
}

func (m *mockService) RunClustering(ctx context.Context) (*service.ClusteringResult, error) {
	return m.runFunc(ctx)
}
func (m *mockService) GetStats(ctx context.Context) (*service.ClusteringStats, error) {
	return m.statsFunc(ctx)
}
func (m *mockService) ListClusters(ctx context.Context, page int, limit int) ([]clusterEntity.Cluster, int64, error) {
	return m.listFunc(ctx, page, limit)
}
func (m *mockService) GetClusterByID(ctx context.Context, id string) (*clusterEntity.Cluster, error) {
	return m.detailFunc(ctx, id)
}

func TestRun(t *testing.T) {
	app := fiber.New()
	h := NewHandler(&mockService{runFunc: func(ctx context.Context) (*service.ClusteringResult, error) {
		return &service.ClusteringResult{ArticlesProcessed: 1, ClustersCreated: 1, ArticlesClustered: 1}, nil
	}})
	app.Post("/internal/clustering/run", h.Run)

	resp, err := app.Test(httptest.NewRequest(http.MethodPost, "/internal/clustering/run", nil))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestRun_Conflict(t *testing.T) {
	app := fiber.New()
	h := NewHandler(&mockService{runFunc: func(ctx context.Context) (*service.ClusteringResult, error) {
		return nil, service.ErrWorkerAlreadyRunning
	}})
	app.Post("/internal/clustering/run", h.Run)

	resp, err := app.Test(httptest.NewRequest(http.MethodPost, "/internal/clustering/run", nil))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409, got %d", resp.StatusCode)
	}
}

func TestRun_Error(t *testing.T) {
	app := fiber.New()
	h := NewHandler(&mockService{runFunc: func(ctx context.Context) (*service.ClusteringResult, error) {
		return nil, errors.New("db failed")
	}})
	app.Post("/internal/clustering/run", h.Run)

	resp, err := app.Test(httptest.NewRequest(http.MethodPost, "/internal/clustering/run", nil))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.StatusCode)
	}
}

func TestStatsEndpoint(t *testing.T) {
	app := fiber.New()
	h := NewHandler(&mockService{statsFunc: func(ctx context.Context) (*service.ClusteringStats, error) {
		return &service.ClusteringStats{TotalClusters: 2}, nil
	}})
	app.Get("/internal/clustering/stats", h.Stats)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/internal/clustering/stats", nil))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestList(t *testing.T) {
	app := fiber.New()
	h := NewHandler(&mockService{listFunc: func(ctx context.Context, page int, limit int) ([]clusterEntity.Cluster, int64, error) {
		if page != 2 || limit != 10 {
			t.Fatalf("unexpected pagination page=%d limit=%d", page, limit)
		}
		return []clusterEntity.Cluster{{ID: uuid.New(), Title: "Cluster"}}, 1, nil
	}})
	app.Get("/api/v1/clusters", h.List)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/v1/clusters?page=2&limit=10", nil))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestList_DefaultsAndError(t *testing.T) {
	app := fiber.New()
	h := NewHandler(&mockService{listFunc: func(ctx context.Context, page int, limit int) ([]clusterEntity.Cluster, int64, error) {
		if page != 1 || limit != 100 {
			t.Fatalf("unexpected pagination page=%d limit=%d", page, limit)
		}
		return nil, 0, errors.New("db down")
	}})
	app.Get("/api/v1/clusters", h.List)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/v1/clusters?page=-1&limit=500", nil))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.StatusCode)
	}
}

func TestDetail(t *testing.T) {
	id := uuid.New()
	app := fiber.New()
	h := NewHandler(&mockService{detailFunc: func(ctx context.Context, gotID string) (*clusterEntity.Cluster, error) {
		return &clusterEntity.Cluster{ID: id, Title: "Cluster"}, nil
	}})
	app.Get("/api/v1/clusters/:id", h.Detail)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/v1/clusters/"+id.String(), nil))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var body struct {
		Success bool                  `json:"success"`
		Data    clusterEntity.Cluster `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !body.Success || body.Data.Title != "Cluster" {
		t.Fatalf("unexpected response: %+v", body)
	}
}

func TestDetail_NotFound(t *testing.T) {
	app := fiber.New()
	h := NewHandler(&mockService{detailFunc: func(ctx context.Context, id string) (*clusterEntity.Cluster, error) {
		return nil, clusterRepo.ErrNotFound
	}})
	app.Get("/api/v1/clusters/:id", h.Detail)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/v1/clusters/missing", nil))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestDetail_Error(t *testing.T) {
	app := fiber.New()
	h := NewHandler(&mockService{detailFunc: func(ctx context.Context, id string) (*clusterEntity.Cluster, error) {
		return nil, errors.New("db down")
	}})
	app.Get("/api/v1/clusters/:id", h.Detail)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/api/v1/clusters/id", nil))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.StatusCode)
	}
}

func TestStatsEndpoint_Error(t *testing.T) {
	app := fiber.New()
	h := NewHandler(&mockService{statsFunc: func(ctx context.Context) (*service.ClusteringStats, error) {
		return nil, errors.New("db down")
	}})
	app.Get("/internal/clustering/stats", h.Stats)

	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/internal/clustering/stats", nil))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", resp.StatusCode)
	}
}
