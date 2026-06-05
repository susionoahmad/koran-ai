package repository

import (
	"context"
	"koran-ai-backend/internal/source/entity"
)

type Repository interface {
	Create(ctx context.Context, s *entity.Source) error
	GetByID(ctx context.Context, id string) (*entity.Source, error)
	GetByName(ctx context.Context, name string) (*entity.Source, error)
	GetByBaseURL(ctx context.Context, baseURL string) (*entity.Source, error)
	List(ctx context.Context, page, limit int) ([]entity.Source, int64, error)
	Update(ctx context.Context, s *entity.Source) error
	Delete(ctx context.Context, id string) error
}
