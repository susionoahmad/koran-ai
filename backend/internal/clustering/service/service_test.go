package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	articleEntity "koran-ai-backend/internal/article/entity"
	clusterEntity "koran-ai-backend/internal/clustering/entity"
)

type mockArticleRepository struct {
	candidates      []articleEntity.Article
	countCandidates int64
	err             error
}

func (m *mockArticleRepository) Create(ctx context.Context, a *articleEntity.Article) error {
	return nil
}
func (m *mockArticleRepository) ExistsByURL(ctx context.Context, url string) (bool, error) {
	return false, nil
}
func (m *mockArticleRepository) ExistsByHash(ctx context.Context, hash string) (bool, error) {
	return false, nil
}
func (m *mockArticleRepository) ListUnprocessed(ctx context.Context, limit int) ([]articleEntity.Article, error) {
	return nil, nil
}
func (m *mockArticleRepository) GetByID(ctx context.Context, id string) (*articleEntity.Article, error) {
	return nil, nil
}
func (m *mockArticleRepository) ListUnprocessedForAI(ctx context.Context, limit int) ([]articleEntity.Article, error) {
	return nil, nil
}
func (m *mockArticleRepository) ListUnclustered(ctx context.Context, limit int) ([]articleEntity.Article, error) {
	return nil, nil
}
func (m *mockArticleRepository) ListClusterCandidates(ctx context.Context, limit int) ([]articleEntity.Article, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.candidates, nil
}
func (m *mockArticleRepository) CountClusterCandidates(ctx context.Context) (int64, error) {
	if m.err != nil {
		return 0, m.err
	}
	return m.countCandidates, nil
}
func (m *mockArticleRepository) CountTotal(ctx context.Context) (int64, error) { return 0, nil }
func (m *mockArticleRepository) CountToday(ctx context.Context) (int64, error) { return 0, nil }
func (m *mockArticleRepository) UpdateAIResult(ctx context.Context, id string, category string, confidence float64) error {
	return nil
}
func (m *mockArticleRepository) UpdateAIError(ctx context.Context, id string, errMsg string) error {
	return nil
}
func (m *mockArticleRepository) CountAIPending(ctx context.Context) (int64, error) {
	return 0, nil
}
func (m *mockArticleRepository) CountAIFailed(ctx context.Context) (int64, error) {
	return 0, nil
}
func (m *mockArticleRepository) CountAIProcessed(ctx context.Context) (int64, error) {
	return 0, nil
}

type mockClusterRepository struct {
	clusters         []clusterEntity.Cluster
	created          []clusterEntity.Cluster
	added            []string
	statsUpdated     int
	clusteredArticle int64
	err              error
}

type mockRedis struct {
	redis.Cmdable
	acquired bool
	err      error
	deleted  bool
}

func (m *mockRedis) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd {
	cmd := redis.NewBoolCmd(ctx)
	if m.err != nil {
		cmd.SetErr(m.err)
		return cmd
	}
	cmd.SetVal(m.acquired)
	return cmd
}

func (m *mockRedis) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	m.deleted = true
	cmd := redis.NewIntCmd(ctx)
	cmd.SetVal(1)
	return cmd
}

func (m *mockClusterRepository) CreateCluster(ctx context.Context, cluster *clusterEntity.Cluster) error {
	if m.err != nil {
		return m.err
	}
	m.created = append(m.created, *cluster)
	return nil
}
func (m *mockClusterRepository) AddArticleToCluster(ctx context.Context, clusterID string, articleID string) error {
	if m.err != nil {
		return m.err
	}
	m.added = append(m.added, clusterID+":"+articleID)
	return nil
}
func (m *mockClusterRepository) ListClusters(ctx context.Context, page int, limit int) ([]clusterEntity.Cluster, int64, error) {
	if m.err != nil {
		return nil, 0, m.err
	}
	return m.clusters, int64(len(m.clusters)), nil
}
func (m *mockClusterRepository) GetClusterByID(ctx context.Context, id string) (*clusterEntity.Cluster, error) {
	for i := range m.clusters {
		if m.clusters[i].ID.String() == id {
			return &m.clusters[i], nil
		}
	}
	return nil, errors.New("not found")
}
func (m *mockClusterRepository) UpdateClusterStats(ctx context.Context, clusterID string, articleCount int, confidence float64) error {
	if m.err != nil {
		return m.err
	}
	m.statsUpdated++
	return nil
}
func (m *mockClusterRepository) CountClusteredArticles(ctx context.Context) (int64, error) {
	if m.err != nil {
		return 0, m.err
	}
	return m.clusteredArticle, nil
}

func TestRunClustering_JoinsExistingCluster(t *testing.T) {
	articleID := uuid.New()
	clusterID := uuid.New()
	artRepo := &mockArticleRepository{candidates: []articleEntity.Article{{
		ID:         articleID,
		Title:      "Bank Indonesia Pertahankan BI Rate",
		AICategory: stringPtr("ekonomi"),
	}}}
	clRepo := &mockClusterRepository{clusters: []clusterEntity.Cluster{{
		ID:           clusterID,
		Title:        "Bank Indonesia Pertahankan BI Rate",
		Category:     "ekonomi",
		ArticleCount: 2,
		Confidence:   0.9,
	}}}

	result, err := NewService(artRepo, clRepo, nil).RunClustering(context.Background())
	if err != nil {
		t.Fatalf("RunClustering returned error: %v", err)
	}
	if result.ClustersCreated != 0 || result.ArticlesClustered != 1 || clRepo.statsUpdated != 1 {
		t.Fatalf("unexpected result: %+v statsUpdated=%d", result, clRepo.statsUpdated)
	}
}

func TestRunClustering_CreatesNewCluster(t *testing.T) {
	articleID := uuid.New()
	artRepo := &mockArticleRepository{candidates: []articleEntity.Article{{
		ID:         articleID,
		Title:      "Pemerintah Bangun Sekolah Baru",
		AICategory: stringPtr("nasional"),
	}}}
	clRepo := &mockClusterRepository{clusters: []clusterEntity.Cluster{{
		ID:           uuid.New(),
		Title:        "Bank Indonesia Pertahankan BI Rate",
		Category:     "ekonomi",
		ArticleCount: 1,
		Confidence:   1,
		CreatedAt:    time.Now(),
	}}}

	result, err := NewService(artRepo, clRepo, nil).RunClustering(context.Background())
	if err != nil {
		t.Fatalf("RunClustering returned error: %v", err)
	}
	if result.ClustersCreated != 1 || len(clRepo.created) != 1 || result.ArticlesClustered != 1 {
		t.Fatalf("expected one new cluster, got result=%+v created=%d", result, len(clRepo.created))
	}
	if clRepo.created[0].Category != "nasional" {
		t.Fatalf("expected category nasional, got %q", clRepo.created[0].Category)
	}
}

func TestRunClustering_RedisUnavailableContinues(t *testing.T) {
	artRepo := &mockArticleRepository{}
	clRepo := &mockClusterRepository{}
	result, err := NewService(artRepo, clRepo, nil).RunClustering(context.Background())
	if err != nil {
		t.Fatalf("expected nil redis to continue, got %v", err)
	}
	if result.ArticlesProcessed != 0 {
		t.Fatalf("expected no processed articles, got %+v", result)
	}
}

func TestRunClustering_RedisErrorContinues(t *testing.T) {
	artRepo := &mockArticleRepository{}
	clRepo := &mockClusterRepository{}
	result, err := NewService(artRepo, clRepo, &mockRedis{err: errors.New("redis down")}).RunClustering(context.Background())
	if err != nil {
		t.Fatalf("expected redis errors to be ignored, got %v", err)
	}
	if result.ArticlesProcessed != 0 {
		t.Fatalf("expected no processed articles, got %+v", result)
	}
}

func TestRunClustering_LockAlreadyHeld(t *testing.T) {
	_, err := NewService(&mockArticleRepository{}, &mockClusterRepository{}, &mockRedis{acquired: false}).RunClustering(context.Background())
	if !errors.Is(err, ErrWorkerAlreadyRunning) {
		t.Fatalf("expected ErrWorkerAlreadyRunning, got %v", err)
	}
}

func TestRunClustering_ReleasesLock(t *testing.T) {
	rdb := &mockRedis{acquired: true}
	_, err := NewService(&mockArticleRepository{}, &mockClusterRepository{}, rdb).RunClustering(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !rdb.deleted {
		t.Fatal("expected lock to be released")
	}
}

func TestRunClustering_ArticleRepoError(t *testing.T) {
	_, err := NewService(&mockArticleRepository{err: errors.New("db down")}, &mockClusterRepository{}, nil).RunClustering(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRunClustering_ClusterRepoError(t *testing.T) {
	artRepo := &mockArticleRepository{candidates: []articleEntity.Article{{ID: uuid.New(), Title: "A"}}}
	_, err := NewService(artRepo, &mockClusterRepository{err: errors.New("db down")}, nil).RunClustering(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetStats(t *testing.T) {
	artRepo := &mockArticleRepository{countCandidates: 7}
	clRepo := &mockClusterRepository{
		clusters:         []clusterEntity.Cluster{{ID: uuid.New()}},
		clusteredArticle: 4,
	}

	stats, err := NewService(artRepo, clRepo, nil).GetStats(context.Background())
	if err != nil {
		t.Fatalf("GetStats returned error: %v", err)
	}
	if stats.TotalClusters != 1 || stats.ClusteredArticles != 4 || stats.PendingArticles != 7 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

func TestListAndGetCluster(t *testing.T) {
	id := uuid.New()
	clRepo := &mockClusterRepository{clusters: []clusterEntity.Cluster{{ID: id, Title: "Cluster"}}}
	svc := NewService(&mockArticleRepository{}, clRepo, nil)

	clusters, total, err := svc.ListClusters(context.Background(), 1, 20)
	if err != nil || total != 1 || len(clusters) != 1 {
		t.Fatalf("unexpected list result clusters=%+v total=%d err=%v", clusters, total, err)
	}
	cluster, err := svc.GetClusterByID(context.Background(), id.String())
	if err != nil || cluster.ID != id {
		t.Fatalf("unexpected detail result cluster=%+v err=%v", cluster, err)
	}
}

func TestFindBestCluster_CategoryMismatch(t *testing.T) {
	article := articleEntity.Article{Title: "Bank Indonesia Pertahankan BI Rate", AICategory: stringPtr("ekonomi")}
	clusters := []clusterEntity.Cluster{{
		ID:       uuid.New(),
		Title:    "Bank Indonesia Pertahankan BI Rate",
		Category: "nasional",
	}}
	match, score := findBestCluster(article, clusters)
	if match != nil || score != 0 {
		t.Fatalf("expected no match across categories, got match=%+v score=%.2f", match, score)
	}
}

func stringPtr(value string) *string {
	return &value
}
