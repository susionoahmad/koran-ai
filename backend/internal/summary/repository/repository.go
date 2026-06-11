package repository

import (
	"context"
	"errors"

	articleEntity "koran-ai-backend/internal/article/entity"
	"koran-ai-backend/internal/summary/entity"
)

var ErrNotFound = errors.New("summary not found")

// Stats contains summary engine persistence metrics.
type Stats struct {
	TotalSummaries  int64 `json:"total_summaries"`
	PendingClusters int64 `json:"pending_clusters"`
}

// Repository defines data-access operations for summaries.
type Repository interface {
	CreateSummary(ctx context.Context, summary *entity.Summary) error
	GetSummaryByID(ctx context.Context, id string) (*entity.Summary, error)
	GetSummaryByClusterID(ctx context.Context, clusterID string) (*entity.Summary, error)
	ListSummaries(ctx context.Context, page int, limit int) ([]entity.Summary, int64, error)
	CountSummaries(ctx context.Context) (int64, error)
	ListPendingClusters(ctx context.Context, limit int) ([]string, error)
	ListClusterArticles(ctx context.Context, clusterID string) ([]articleEntity.Article, error)
	Stats(ctx context.Context) (*Stats, error)
}
