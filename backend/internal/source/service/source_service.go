package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"koran-ai-backend/internal/source/dto"
	"koran-ai-backend/internal/source/entity"
	"koran-ai-backend/internal/source/repository"
)

var (
	ErrDuplicateName    = errors.New("source name already exists")
	ErrDuplicateBaseURL = errors.New("source base URL already exists")
	ErrNotFound         = repository.ErrNotFound
)

type sourceService struct {
	repo repository.Repository
}

func NewSourceService(repo repository.Repository) Service {
	return &sourceService{repo: repo}
}

func (s *sourceService) Create(ctx context.Context, req dto.CreateSourceRequest) (*dto.SourceResponse, error) {
	// 1. Check duplicate name
	existingName, err := s.repo.GetByName(ctx, req.Name)
	if err == nil && existingName != nil {
		return nil, ErrDuplicateName
	}
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, fmt.Errorf("failed to check duplicate name: %w", err)
	}

	// 2. Check duplicate base url
	existingBaseURL, err := s.repo.GetByBaseURL(ctx, req.BaseURL)
	if err == nil && existingBaseURL != nil {
		return nil, ErrDuplicateBaseURL
	}
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, fmt.Errorf("failed to check duplicate base URL: %w", err)
	}

	// 3. Construct entity
	src := &entity.Source{
		ID:         uuid.New(),
		Name:       req.Name,
		BaseURL:    req.BaseURL,
		RSSURL:     req.RSSURL,
		SourceType: req.SourceType,
		IsActive:   true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// 4. Save source
	err = s.repo.Create(ctx, src)
	if err != nil {
		return nil, err
	}

	return s.toResponse(src), nil
}

func (s *sourceService) GetByID(ctx context.Context, id string) (*dto.SourceResponse, error) {
	src, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.toResponse(src), nil
}

func (s *sourceService) List(ctx context.Context, page, limit int) (*dto.SourceListResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	sources, total, err := s.repo.List(ctx, page, limit)
	if err != nil {
		return nil, err
	}

	var responses []dto.SourceResponse
	for _, src := range sources {
		responses = append(responses, *s.toResponse(&src))
	}

	return &dto.SourceListResponse{
		Sources: responses,
		Total:   total,
		Page:    page,
		Limit:   limit,
	}, nil
}

func (s *sourceService) Update(ctx context.Context, id string, req dto.UpdateSourceRequest) (*dto.SourceResponse, error) {
	// 1. Verify existence
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 2. Check duplicate name if changing
	if existing.Name != req.Name {
		existingName, err := s.repo.GetByName(ctx, req.Name)
		if err == nil && existingName != nil {
			return nil, ErrDuplicateName
		}
		if err != nil && !errors.Is(err, ErrNotFound) {
			return nil, fmt.Errorf("failed to check duplicate name: %w", err)
		}
	}

	// 3. Check duplicate base url if changing
	if existing.BaseURL != req.BaseURL {
		existingBaseURL, err := s.repo.GetByBaseURL(ctx, req.BaseURL)
		if err == nil && existingBaseURL != nil {
			return nil, ErrDuplicateBaseURL
		}
		if err != nil && !errors.Is(err, ErrNotFound) {
			return nil, fmt.Errorf("failed to check duplicate base URL: %w", err)
		}
	}

	// 4. Update entity fields
	existing.Name = req.Name
	existing.BaseURL = req.BaseURL
	existing.RSSURL = req.RSSURL
	existing.SourceType = req.SourceType
	existing.IsActive = *req.IsActive
	existing.UpdatedAt = time.Now()

	// 5. Update repo
	err = s.repo.Update(ctx, existing)
	if err != nil {
		return nil, err
	}

	return s.toResponse(existing), nil
}

func (s *sourceService) Delete(ctx context.Context, id string) error {
	// soft delete: set is_active=false
	return s.repo.Delete(ctx, id)
}

func (s *sourceService) toResponse(src *entity.Source) *dto.SourceResponse {
	return &dto.SourceResponse{
		ID:         src.ID.String(),
		Name:       src.Name,
		BaseURL:    src.BaseURL,
		RSSURL:     src.RSSURL,
		SourceType: src.SourceType,
		IsActive:   src.IsActive,
		CreatedAt:  src.CreatedAt,
		UpdatedAt:  src.UpdatedAt,
	}
}
