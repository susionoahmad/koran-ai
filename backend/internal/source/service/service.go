package service

import (
	"context"
	"koran-ai-backend/internal/source/dto"
)

type Service interface {
	Create(ctx context.Context, req dto.CreateSourceRequest) (*dto.SourceResponse, error)
	GetByID(ctx context.Context, id string) (*dto.SourceResponse, error)
	List(ctx context.Context, page, limit int) (*dto.SourceListResponse, error)
	Update(ctx context.Context, id string, req dto.UpdateSourceRequest) (*dto.SourceResponse, error)
	Delete(ctx context.Context, id string) error
}
