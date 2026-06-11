package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	aiClient "koran-ai-backend/internal/ai/client"
	articleEntity "koran-ai-backend/internal/article/entity"
	"koran-ai-backend/internal/summary/entity"
	"koran-ai-backend/internal/summary/repository"
)

type mockRepo struct {
	pending  []string
	articles map[string][]articleEntity.Article
	created  []entity.Summary
	stats    repository.Stats
	err      error
}

func (m *mockRepo) CreateSummary(ctx context.Context, summary *entity.Summary) error {
	if m.err != nil {
		return m.err
	}
	m.created = append(m.created, *summary)
	return nil
}
func (m *mockRepo) GetSummaryByID(ctx context.Context, id string) (*entity.Summary, error) {
	return &entity.Summary{ID: uuid.MustParse(id)}, nil
}
func (m *mockRepo) GetSummaryByClusterID(ctx context.Context, clusterID string) (*entity.Summary, error) {
	return &entity.Summary{ClusterID: uuid.MustParse(clusterID)}, nil
}
func (m *mockRepo) ListSummaries(ctx context.Context, page int, limit int) ([]entity.Summary, int64, error) {
	return m.created, int64(len(m.created)), nil
}
func (m *mockRepo) CountSummaries(ctx context.Context) (int64, error) {
	return int64(len(m.created)), nil
}
func (m *mockRepo) ListPendingClusters(ctx context.Context, limit int) ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.pending, nil
}
func (m *mockRepo) ListClusterArticles(ctx context.Context, clusterID string) ([]articleEntity.Article, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.articles[clusterID], nil
}
func (m *mockRepo) Stats(ctx context.Context) (*repository.Stats, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &m.stats, nil
}

type mockGemini struct {
	err error
}

func (m mockGemini) ClassifyArticle(ctx context.Context, title string, content string) (*aiClient.ClassifyResult, error) {
	return nil, errors.New("not implemented")
}
func (m mockGemini) SummarizeCluster(ctx context.Context, content string) (*aiClient.SummaryResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &aiClient.SummaryResult{
		Headline:      "Headline",
		SummaryShort:  "Short",
		SummaryMedium: "Medium",
		KeyPoints:     []string{"Point"},
		Confidence:    0.9,
	}, nil
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

func TestGenerateClusterSummary(t *testing.T) {
	clusterID := uuid.New().String()
	repo := &mockRepo{articles: map[string][]articleEntity.Article{
		clusterID: {{ID: uuid.New(), Title: "T", Content: "C"}},
	}}
	svc := NewService(repo, mockGemini{}, nil, "gemini-test", nil)

	summary, err := svc.GenerateClusterSummary(context.Background(), clusterID)
	if err != nil {
		t.Fatalf("GenerateClusterSummary returned error: %v", err)
	}
	if summary.Headline != "Headline" || len(repo.created) != 1 || repo.created[0].AIModel != "gemini-test" {
		t.Fatalf("unexpected summary=%+v created=%+v", summary, repo.created)
	}
}

func TestGenerateClusterSummary_ZeroArticles(t *testing.T) {
	_, err := NewService(&mockRepo{articles: map[string][]articleEntity.Article{}}, mockGemini{}, nil, "", nil).GenerateClusterSummary(context.Background(), uuid.New().String())
	if !errors.Is(err, ErrClusterHasNoArticles) {
		t.Fatalf("expected ErrClusterHasNoArticles, got %v", err)
	}
}

func TestGenerateClusterSummary_Errors(t *testing.T) {
	clusterID := uuid.New().String()
	_, err := NewService(&mockRepo{err: errors.New("db down")}, mockGemini{}, nil, "", nil).GenerateClusterSummary(context.Background(), clusterID)
	if err == nil {
		t.Fatal("expected repository error")
	}

	_, err = NewService(&mockRepo{articles: map[string][]articleEntity.Article{
		clusterID: {{ID: uuid.New(), Title: "T", Content: "C"}},
	}}, nil, nil, "", nil).GenerateClusterSummary(context.Background(), clusterID)
	if err == nil {
		t.Fatal("expected nil gemini client error")
	}

	_, err = NewService(&mockRepo{articles: map[string][]articleEntity.Article{
		clusterID: {{ID: uuid.New(), Title: "T", Content: "C"}},
	}}, mockGemini{err: errors.New("quota exceeded")}, nil, "", nil).GenerateClusterSummary(context.Background(), clusterID)
	if err == nil {
		t.Fatal("expected gemini error")
	}

	_, err = NewService(&mockRepo{articles: map[string][]articleEntity.Article{
		"bad-id": {{ID: uuid.New(), Title: "T", Content: "C"}},
	}}, mockGemini{}, nil, "", nil).GenerateClusterSummary(context.Background(), "bad-id")
	if err == nil {
		t.Fatal("expected invalid cluster id error")
	}

	_, err = NewService(&mockRepo{
		articles: map[string][]articleEntity.Article{clusterID: {{ID: uuid.New(), Title: "T", Content: "C"}}},
		err:      errors.New("insert failed"),
	}, mockGemini{}, nil, "", nil).GenerateClusterSummary(context.Background(), clusterID)
	if err == nil {
		t.Fatal("expected create summary error")
	}
}

func TestGenerateBatch_ContinuesAfterFailure(t *testing.T) {
	okID := uuid.New().String()
	emptyID := uuid.New().String()
	repo := &mockRepo{
		pending: []string{okID, emptyID},
		articles: map[string][]articleEntity.Article{
			okID: {{ID: uuid.New(), Title: "T", Content: "C"}},
		},
	}
	result, err := NewService(repo, mockGemini{}, nil, "", nil).GenerateBatch(context.Background(), 10)
	if err != nil {
		t.Fatalf("GenerateBatch returned error: %v", err)
	}
	if result.TotalClusters != 2 || result.Processed != 1 || result.Failed != 1 || result.SuccessRate != 50 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestGenerateBatch_RedisBehavior(t *testing.T) {
	_, err := NewService(&mockRepo{}, mockGemini{}, &mockRedis{acquired: false}, "", nil).GenerateBatch(context.Background(), 1)
	if !errors.Is(err, ErrWorkerAlreadyRunning) {
		t.Fatalf("expected lock error, got %v", err)
	}

	rdb := &mockRedis{acquired: true}
	_, err = NewService(&mockRepo{}, mockGemini{}, rdb, "", nil).GenerateBatch(context.Background(), 1)
	if err != nil {
		t.Fatalf("expected success with acquired lock, got %v", err)
	}
	if !rdb.deleted {
		t.Fatal("expected lock release")
	}

	_, err = NewService(&mockRepo{}, mockGemini{}, &mockRedis{err: errors.New("redis down")}, "", nil).GenerateBatch(context.Background(), 1)
	if err != nil {
		t.Fatalf("expected redis failure to be ignored, got %v", err)
	}
}

func TestGenerateBatch_NilRedis(t *testing.T) {
	result, err := NewService(&mockRepo{}, mockGemini{}, nil, "", nil).GenerateBatch(context.Background(), 1)
	if err != nil {
		t.Fatalf("expected nil redis to continue, got %v", err)
	}
	if result.TotalClusters != 0 {
		t.Fatalf("expected empty batch, got %+v", result)
	}
}

func TestGenerateBatch_TypedNilRedis(t *testing.T) {
	var rdb *redis.Client
	result, err := NewService(&mockRepo{}, mockGemini{}, rdb, "", nil).GenerateBatch(context.Background(), 1)
	if err != nil {
		t.Fatalf("expected typed nil redis to continue, got %v", err)
	}
	if result.TotalClusters != 0 {
		t.Fatalf("expected empty batch, got %+v", result)
	}
}

func TestGenerateBatch_RedisUnavailable(t *testing.T) {
	result, err := NewService(&mockRepo{}, mockGemini{}, &mockRedis{err: errors.New("redis unavailable")}, "", nil).GenerateBatch(context.Background(), 1)
	if err != nil {
		t.Fatalf("expected redis unavailable to continue, got %v", err)
	}
	if result.TotalClusters != 0 {
		t.Fatalf("expected empty batch, got %+v", result)
	}
}

func TestGenerateBatch_LockAlreadyHeld(t *testing.T) {
	_, err := NewService(&mockRepo{}, mockGemini{}, &mockRedis{acquired: false}, "", nil).GenerateBatch(context.Background(), 1)
	if !errors.Is(err, ErrWorkerAlreadyRunning) {
		t.Fatalf("expected ErrWorkerAlreadyRunning, got %v", err)
	}
}

func TestGenerateBatch_ListPendingErrorAndLimitDefaults(t *testing.T) {
	_, err := NewService(&mockRepo{err: errors.New("db down")}, mockGemini{}, nil, "", nil).GenerateBatch(context.Background(), -1)
	if err == nil {
		t.Fatal("expected list pending error")
	}
}

func TestGetStats(t *testing.T) {
	stats, err := NewService(&mockRepo{stats: repository.Stats{TotalSummaries: 2, PendingClusters: 3}}, mockGemini{}, nil, "", nil).GetStats(context.Background())
	if err != nil {
		t.Fatalf("GetStats returned error: %v", err)
	}
	if stats.TotalSummaries != 2 || stats.PendingClusters != 3 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

func TestGetStats_ErrorAndForwarders(t *testing.T) {
	_, err := NewService(&mockRepo{err: errors.New("db down")}, mockGemini{}, nil, "", nil).GetStats(context.Background())
	if err == nil {
		t.Fatal("expected stats error")
	}

	id := uuid.New().String()
	repo := &mockRepo{created: []entity.Summary{{ID: uuid.MustParse(id)}}}
	svc := NewService(repo, mockGemini{}, nil, "", nil)
	if _, _, err := svc.ListSummaries(context.Background(), 1, 20); err != nil {
		t.Fatalf("ListSummaries returned error: %v", err)
	}
	if _, err := svc.GetSummaryByID(context.Background(), id); err != nil {
		t.Fatalf("GetSummaryByID returned error: %v", err)
	}
	if _, err := svc.GetSummaryByClusterID(context.Background(), id); err != nil {
		t.Fatalf("GetSummaryByClusterID returned error: %v", err)
	}
}

func TestMergeArticleContent(t *testing.T) {
	got := mergeArticleContent([]articleEntity.Article{{Title: "A", Content: "B"}, {Title: "C", Content: "D"}})
	if got == "" || !contains(got, "Title: A") || !contains(got, "Content:\nD") {
		t.Fatalf("unexpected merged content: %q", got)
	}
}

func contains(s string, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s string, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
