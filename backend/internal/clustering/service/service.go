package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	articleEntity "koran-ai-backend/internal/article/entity"
	articleRepo "koran-ai-backend/internal/article/repository"
	clusterEntity "koran-ai-backend/internal/clustering/entity"
	clusterRepo "koran-ai-backend/internal/clustering/repository"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	defaultBatchLimit    = 100
	clusterLookupLimit   = 10000
	clusteringLockKey    = "clustering_worker_lock"
	clusteringLockTTL    = 5 * time.Minute
	defaultNewConfidence = 1.0
)

var ErrWorkerAlreadyRunning = errors.New("clustering worker already running")

type ClusteringResult struct {
	ArticlesProcessed int `json:"articles_processed"`
	ClustersCreated   int `json:"clusters_created"`
	ArticlesClustered int `json:"articles_clustered"`
}

type ClusteringStats struct {
	TotalClusters     int64 `json:"total_clusters"`
	ClusteredArticles int64 `json:"clustered_articles"`
	PendingArticles   int64 `json:"pending_articles"`
}

// Service defines news clustering operations.
type Service interface {
	RunClustering(ctx context.Context) (*ClusteringResult, error)
	GetStats(ctx context.Context) (*ClusteringStats, error)
	ListClusters(ctx context.Context, page int, limit int) ([]clusterEntity.Cluster, int64, error)
	GetClusterByID(ctx context.Context, id string) (*clusterEntity.Cluster, error)
}

type clusteringService struct {
	articleRepo articleRepo.Repository
	clusterRepo clusterRepo.Repository
	rdb         redis.Cmdable
	batchLimit  int
}

func NewService(articleRepo articleRepo.Repository, clusterRepo clusterRepo.Repository, rdb redis.Cmdable) Service {
	return &clusteringService{
		articleRepo: articleRepo,
		clusterRepo: clusterRepo,
		rdb:         rdb,
		batchLimit:  defaultBatchLimit,
	}
}

func (s *clusteringService) RunClustering(ctx context.Context) (*ClusteringResult, error) {
	lockAcquired, err := s.acquireLock(ctx)
	if err != nil {
		return nil, err
	}
	if lockAcquired {
		defer s.releaseLock(ctx)
	}

	articles, err := s.articleRepo.ListClusterCandidates(ctx, s.batchLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to list cluster candidates: %w", err)
	}

	result := &ClusteringResult{ArticlesProcessed: len(articles)}
	if len(articles) == 0 {
		return result, nil
	}

	clusters, _, err := s.clusterRepo.ListClusters(ctx, 1, clusterLookupLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to list existing clusters: %w", err)
	}

	for _, article := range articles {
		match, score := findBestCluster(article, clusters)
		if match != nil && score >= SimilarityThreshold {
			nextCount := match.ArticleCount + 1
			nextConfidence := rollingConfidence(match.Confidence, score, match.ArticleCount)
			if err := s.clusterRepo.AddArticleToCluster(ctx, match.ID.String(), article.ID.String()); err != nil {
				return nil, err
			}
			if err := s.clusterRepo.UpdateClusterStats(ctx, match.ID.String(), nextCount, nextConfidence); err != nil {
				return nil, err
			}
			match.ArticleCount = nextCount
			match.Confidence = nextConfidence
			result.ArticlesClustered++
			continue
		}

		cluster := clusterEntity.Cluster{
			ID:           uuid.New(),
			Title:        article.Title,
			Category:     stringValue(article.AICategory),
			ArticleCount: 1,
			Confidence:   defaultNewConfidence,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		if err := s.clusterRepo.CreateCluster(ctx, &cluster); err != nil {
			return nil, err
		}
		if err := s.clusterRepo.AddArticleToCluster(ctx, cluster.ID.String(), article.ID.String()); err != nil {
			return nil, err
		}
		clusters = append(clusters, cluster)
		result.ClustersCreated++
		result.ArticlesClustered++
	}

	return result, nil
}

func (s *clusteringService) GetStats(ctx context.Context) (*ClusteringStats, error) {
	_, totalClusters, err := s.clusterRepo.ListClusters(ctx, 1, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to count clusters: %w", err)
	}
	clustered, err := s.clusterRepo.CountClusteredArticles(ctx)
	if err != nil {
		return nil, err
	}
	pending, err := s.articleRepo.CountClusterCandidates(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count pending cluster candidates: %w", err)
	}
	return &ClusteringStats{
		TotalClusters:     totalClusters,
		ClusteredArticles: clustered,
		PendingArticles:   pending,
	}, nil
}

func (s *clusteringService) ListClusters(ctx context.Context, page int, limit int) ([]clusterEntity.Cluster, int64, error) {
	return s.clusterRepo.ListClusters(ctx, page, limit)
}

func (s *clusteringService) GetClusterByID(ctx context.Context, id string) (*clusterEntity.Cluster, error) {
	return s.clusterRepo.GetClusterByID(ctx, id)
}

func (s *clusteringService) acquireLock(ctx context.Context) (bool, error) {
	if s.rdb == nil {
		return false, nil
	}
	acquired, err := s.rdb.SetNX(ctx, clusteringLockKey, "running", clusteringLockTTL).Result()
	if err != nil {
		return false, nil
	}
	if !acquired {
		return false, ErrWorkerAlreadyRunning
	}
	return true, nil
}

func (s *clusteringService) releaseLock(ctx context.Context) {
	if s.rdb != nil {
		_, _ = s.rdb.Del(ctx, clusteringLockKey).Result()
	}
}

func findBestCluster(article articleEntity.Article, clusters []clusterEntity.Cluster) (*clusterEntity.Cluster, float64) {
	var best *clusterEntity.Cluster
	bestScore := 0.0
	for i := range clusters {
		articleCategory := stringValue(article.AICategory)
		if articleCategory != "" && clusters[i].Category != "" && articleCategory != clusters[i].Category {
			continue
		}
		score := CombinedSimilarity(article.Title, clusters[i].Title)
		if score > bestScore {
			best = &clusters[i]
			bestScore = score
		}
	}
	return best, bestScore
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func rollingConfidence(current float64, score float64, articleCount int) float64 {
	if articleCount <= 0 {
		return score
	}
	return ((current * float64(articleCount)) + score) / float64(articleCount+1)
}
