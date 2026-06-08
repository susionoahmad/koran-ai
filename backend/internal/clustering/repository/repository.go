package repository

import (
	"context"
	"errors"

	"koran-ai-backend/internal/clustering/entity"
)

var ErrNotFound = errors.New("cluster not found")

// Repository defines data-access operations for news clustering.
type Repository interface {
	CreateCluster(ctx context.Context, cluster *entity.Cluster) error
	AddArticleToCluster(ctx context.Context, clusterID string, articleID string) error
	ListClusters(ctx context.Context, page int, limit int) ([]entity.Cluster, int64, error)
	GetClusterByID(ctx context.Context, id string) (*entity.Cluster, error)
	UpdateClusterStats(ctx context.Context, clusterID string, articleCount int, confidence float64) error
	CountClusteredArticles(ctx context.Context) (int64, error)
}
