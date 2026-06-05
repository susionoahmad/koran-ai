package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"koran-ai-backend/internal/source/dto"
	"koran-ai-backend/internal/source/entity"
)

type mockRepository struct {
	GetByNameFunc    func(ctx context.Context, name string) (*entity.Source, error)
	GetByBaseURLFunc func(ctx context.Context, baseURL string) (*entity.Source, error)
	GetByIDFunc      func(ctx context.Context, id string) (*entity.Source, error)
	CreateFunc       func(ctx context.Context, s *entity.Source) error
	UpdateFunc       func(ctx context.Context, s *entity.Source) error
	DeleteFunc       func(ctx context.Context, id string) error
	ListFunc         func(ctx context.Context, page, limit int) ([]entity.Source, int64, error)
	ListActiveFunc   func(ctx context.Context) ([]entity.Source, error)
}

func (m *mockRepository) Create(ctx context.Context, s *entity.Source) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, s)
	}
	return nil
}

func (m *mockRepository) GetByID(ctx context.Context, id string) (*entity.Source, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, ErrNotFound
}

func (m *mockRepository) GetByName(ctx context.Context, name string) (*entity.Source, error) {
	if m.GetByNameFunc != nil {
		return m.GetByNameFunc(ctx, name)
	}
	return nil, ErrNotFound
}

func (m *mockRepository) GetByBaseURL(ctx context.Context, baseURL string) (*entity.Source, error) {
	if m.GetByBaseURLFunc != nil {
		return m.GetByBaseURLFunc(ctx, baseURL)
	}
	return nil, ErrNotFound
}

func (m *mockRepository) List(ctx context.Context, page, limit int) ([]entity.Source, int64, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, page, limit)
	}
	return nil, 0, nil
}

func (m *mockRepository) Update(ctx context.Context, s *entity.Source) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, s)
	}
	return nil
}

func (m *mockRepository) Delete(ctx context.Context, id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, id)
	}
	return nil
}

func (m *mockRepository) ListActive(ctx context.Context) ([]entity.Source, error) {
	if m.ListActiveFunc != nil {
		return m.ListActiveFunc(ctx)
	}
	return nil, nil
}

func TestCreate_Success(t *testing.T) {
	mockRepo := &mockRepository{
		GetByNameFunc: func(ctx context.Context, name string) (*entity.Source, error) {
			return nil, ErrNotFound
		},
		GetByBaseURLFunc: func(ctx context.Context, baseURL string) (*entity.Source, error) {
			return nil, ErrNotFound
		},
		CreateFunc: func(ctx context.Context, s *entity.Source) error {
			return nil
		},
	}

	svc := NewSourceService(mockRepo)
	req := dto.CreateSourceRequest{
		Name:       "Test Source",
		BaseURL:    "https://test.com",
		RSSURL:     "https://test.com/rss",
		SourceType: "RSS",
	}

	res, err := svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if res.Name != req.Name {
		t.Errorf("Expected Name %s, got %s", req.Name, res.Name)
	}
	if res.BaseURL != req.BaseURL {
		t.Errorf("Expected BaseURL %s, got %s", req.BaseURL, res.BaseURL)
	}
}

func TestCreate_DuplicateName(t *testing.T) {
	mockRepo := &mockRepository{
		GetByNameFunc: func(ctx context.Context, name string) (*entity.Source, error) {
			return &entity.Source{ID: uuid.New(), Name: name}, nil
		},
	}

	svc := NewSourceService(mockRepo)
	req := dto.CreateSourceRequest{
		Name:    "Test Source",
		BaseURL: "https://test.com",
	}

	_, err := svc.Create(context.Background(), req)
	if !errors.Is(err, ErrDuplicateName) {
		t.Errorf("Expected ErrDuplicateName, got %v", err)
	}
}

func TestCreate_DuplicateBaseURL(t *testing.T) {
	mockRepo := &mockRepository{
		GetByNameFunc: func(ctx context.Context, name string) (*entity.Source, error) {
			return nil, ErrNotFound
		},
		GetByBaseURLFunc: func(ctx context.Context, baseURL string) (*entity.Source, error) {
			return &entity.Source{ID: uuid.New(), BaseURL: baseURL}, nil
		},
	}

	svc := NewSourceService(mockRepo)
	req := dto.CreateSourceRequest{
		Name:    "Test Source",
		BaseURL: "https://test.com",
	}

	_, err := svc.Create(context.Background(), req)
	if !errors.Is(err, ErrDuplicateBaseURL) {
		t.Errorf("Expected ErrDuplicateBaseURL, got %v", err)
	}
}

func TestGetByID_Success(t *testing.T) {
	id := uuid.New()
	mockRepo := &mockRepository{
		GetByIDFunc: func(ctx context.Context, idStr string) (*entity.Source, error) {
			return &entity.Source{
				ID:   id,
				Name: "Test Source",
			}, nil
		},
	}

	svc := NewSourceService(mockRepo)
	res, err := svc.GetByID(context.Background(), id.String())
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if res.ID != id.String() {
		t.Errorf("Expected ID %s, got %s", id.String(), res.ID)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	mockRepo := &mockRepository{
		GetByIDFunc: func(ctx context.Context, idStr string) (*entity.Source, error) {
			return nil, ErrNotFound
		},
	}

	svc := NewSourceService(mockRepo)
	_, err := svc.GetByID(context.Background(), uuid.New().String())
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

func TestList_Success(t *testing.T) {
	mockRepo := &mockRepository{
		ListFunc: func(ctx context.Context, page, limit int) ([]entity.Source, int64, error) {
			return []entity.Source{
				{ID: uuid.New(), Name: "Source 1"},
				{ID: uuid.New(), Name: "Source 2"},
			}, 2, nil
		},
	}

	svc := NewSourceService(mockRepo)
	res, err := svc.List(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if res.Total != 2 {
		t.Errorf("Expected Total 2, got %d", res.Total)
	}
	if len(res.Sources) != 2 {
		t.Errorf("Expected 2 sources, got %d", len(res.Sources))
	}
}

func TestUpdate_Success(t *testing.T) {
	id := uuid.New()
	mockRepo := &mockRepository{
		GetByIDFunc: func(ctx context.Context, idStr string) (*entity.Source, error) {
			return &entity.Source{
				ID:      id,
				Name:    "Old Name",
				BaseURL: "https://old.com",
			}, nil
		},
		GetByNameFunc: func(ctx context.Context, name string) (*entity.Source, error) {
			return nil, ErrNotFound
		},
		GetByBaseURLFunc: func(ctx context.Context, baseURL string) (*entity.Source, error) {
			return nil, ErrNotFound
		},
		UpdateFunc: func(ctx context.Context, s *entity.Source) error {
			return nil
		},
	}

	svc := NewSourceService(mockRepo)
	isActive := true
	req := dto.UpdateSourceRequest{
		Name:       "New Name",
		BaseURL:    "https://new.com",
		RSSURL:     "https://new.com/rss",
		SourceType: "SITEMAP",
		IsActive:   &isActive,
	}

	res, err := svc.Update(context.Background(), id.String(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if res.Name != "New Name" {
		t.Errorf("Expected Name 'New Name', got %s", res.Name)
	}
}

func TestUpdate_DuplicateName(t *testing.T) {
	id := uuid.New()
	otherID := uuid.New()
	mockRepo := &mockRepository{
		GetByIDFunc: func(ctx context.Context, idStr string) (*entity.Source, error) {
			return &entity.Source{
				ID:      id,
				Name:    "Old Name",
				BaseURL: "https://old.com",
			}, nil
		},
		GetByNameFunc: func(ctx context.Context, name string) (*entity.Source, error) {
			return &entity.Source{
				ID:   otherID,
				Name: name,
			}, nil
		},
	}

	svc := NewSourceService(mockRepo)
	isActive := true
	req := dto.UpdateSourceRequest{
		Name:     "Duplicate Name",
		BaseURL:  "https://old.com",
		IsActive: &isActive,
	}

	_, err := svc.Update(context.Background(), id.String(), req)
	if !errors.Is(err, ErrDuplicateName) {
		t.Errorf("Expected ErrDuplicateName, got %v", err)
	}
}

func TestUpdate_DuplicateBaseURL(t *testing.T) {
	id := uuid.New()
	otherID := uuid.New()
	mockRepo := &mockRepository{
		GetByIDFunc: func(ctx context.Context, idStr string) (*entity.Source, error) {
			return &entity.Source{
				ID:      id,
				Name:    "Old Name",
				BaseURL: "https://old.com",
			}, nil
		},
		GetByNameFunc: func(ctx context.Context, name string) (*entity.Source, error) {
			return nil, ErrNotFound
		},
		GetByBaseURLFunc: func(ctx context.Context, baseURL string) (*entity.Source, error) {
			return &entity.Source{
				ID:      otherID,
				BaseURL: baseURL,
			}, nil
		},
	}

	svc := NewSourceService(mockRepo)
	isActive := true
	req := dto.UpdateSourceRequest{
		Name:     "Old Name",
		BaseURL:  "https://duplicate.com",
		IsActive: &isActive,
	}

	_, err := svc.Update(context.Background(), id.String(), req)
	if !errors.Is(err, ErrDuplicateBaseURL) {
		t.Errorf("Expected ErrDuplicateBaseURL, got %v", err)
	}
}

func TestUpdate_NotFound(t *testing.T) {
	mockRepo := &mockRepository{
		GetByIDFunc: func(ctx context.Context, idStr string) (*entity.Source, error) {
			return nil, ErrNotFound
		},
	}

	svc := NewSourceService(mockRepo)
	isActive := true
	req := dto.UpdateSourceRequest{
		Name:     "Name",
		BaseURL:  "https://url.com",
		IsActive: &isActive,
	}

	_, err := svc.Update(context.Background(), uuid.New().String(), req)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

func TestDelete_Success(t *testing.T) {
	mockRepo := &mockRepository{
		DeleteFunc: func(ctx context.Context, id string) error {
			return nil
		},
	}

	svc := NewSourceService(mockRepo)
	err := svc.Delete(context.Background(), uuid.New().String())
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestDelete_NotFound(t *testing.T) {
	mockRepo := &mockRepository{
		DeleteFunc: func(ctx context.Context, id string) error {
			return ErrNotFound
		},
	}

	svc := NewSourceService(mockRepo)
	err := svc.Delete(context.Background(), uuid.New().String())
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}
