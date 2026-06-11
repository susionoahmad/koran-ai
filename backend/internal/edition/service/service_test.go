package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"koran-ai-backend/internal/edition/entity"
	edRepo "koran-ai-backend/internal/edition/repository"
)

type mockRepository struct {
	edRepo.Repository
	exists      bool
	existsErr   error
	summaries   []entity.EditionArticleSummary
	loadErr     error
	createErr   error
	statsTotal  int64
	statsLatest string
	statsErr    error
	editions    []entity.Edition
	listTotal   int64
	listErr     error
	details     *entity.EditionDetailResponse
	detailsErr  error
	getEd       *entity.Edition
	getEdErr    error

	createdEdition  *entity.Edition
	createdArticles []entity.EditionArticle
}

func (m *mockRepository) EditionExists(ctx context.Context, date time.Time) (bool, error) {
	return m.exists, m.existsErr
}

func (m *mockRepository) LoadSummariesForDate(ctx context.Context, date time.Time) ([]entity.EditionArticleSummary, error) {
	return m.summaries, m.loadErr
}

func (m *mockRepository) CreateEdition(ctx context.Context, ed *entity.Edition, arts []entity.EditionArticle) error {
	m.createdEdition = ed
	m.createdArticles = arts
	return m.createErr
}

func (m *mockRepository) GetStats(ctx context.Context) (int64, string, error) {
	return m.statsTotal, m.statsLatest, m.statsErr
}

func (m *mockRepository) ListEditions(ctx context.Context, page, limit int) ([]entity.Edition, int64, error) {
	return m.editions, m.listTotal, m.listErr
}

func (m *mockRepository) GetEditionDetails(ctx context.Context, id string) (*entity.EditionDetailResponse, error) {
	return m.details, m.detailsErr
}

func (m *mockRepository) GetEditionByDate(ctx context.Context, date string) (*entity.Edition, error) {
	return m.getEd, m.getEdErr
}

func (m *mockRepository) GetEditionByID(ctx context.Context, id string) (*entity.Edition, error) {
	return m.getEd, m.getEdErr
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

func TestGenerateEdition_Success(t *testing.T) {
	cluster1 := uuid.New()
	cluster2 := uuid.New()
	cluster3 := uuid.New()
	now := time.Now()

	repo := &mockRepository{
		exists: false,
		summaries: []entity.EditionArticleSummary{
			{
				ClusterID:    cluster1,
				Title:        "Politics 1",
				Category:     "politik",
				ArticleCount: 3,
				Confidence:   0.9,
				GeneratedAt:  now,
			},
			{
				ClusterID:    cluster2,
				Title:        "National 1",
				Category:     "nasional",
				ArticleCount: 10, // Headline (highest count)
				Confidence:   0.98,
				GeneratedAt:  now,
			},
			{
				ClusterID:    cluster3,
				Title:        "Economy 1",
				Category:     "ekonomi",
				ArticleCount: 5,
				Confidence:   0.95,
				GeneratedAt:  now,
			},
		},
	}
	rdb := &mockRedis{acquired: true}

	svc := NewService(repo, rdb, nil)
	ed, err := svc.GenerateEdition(context.Background(), "2026-06-09")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ed.Title != "Koran AI - Edisi 2026-06-09" || *ed.HeadlineClusterID != cluster2 {
		t.Fatalf("headline not selected correctly: %+v", ed)
	}

	if len(repo.createdArticles) != 3 {
		t.Fatalf("expected 3 articles generated, got %d", len(repo.createdArticles))
	}

	var headlineSeen, politicsSeen, economySeen bool
	for _, art := range repo.createdArticles {
		switch art.Section {
		case "Headline News":
			headlineSeen = true
			if art.ClusterID != cluster2 || art.DisplayOrder != 1 {
				t.Fatalf("invalid headline article layout: %+v", art)
			}
		case "Politics":
			politicsSeen = true
			if art.ClusterID != cluster1 || art.DisplayOrder != 1 {
				t.Fatalf("invalid politics layout: %+v", art)
			}
		case "Economy":
			economySeen = true
			if art.ClusterID != cluster3 || art.DisplayOrder != 1 {
				t.Fatalf("invalid economy layout: %+v", art)
			}
		}
	}

	if !headlineSeen || !politicsSeen || !economySeen {
		t.Fatalf("missing mapped sections: headline=%t, politics=%t, economy=%t", headlineSeen, politicsSeen, economySeen)
	}

	if !rdb.deleted {
		t.Fatal("expected lock to be released")
	}
}

func TestGenerateEdition_InvalidDate(t *testing.T) {
	svc := NewService(&mockRepository{}, nil, nil)
	_, err := svc.GenerateEdition(context.Background(), "invalid-date")
	if err == nil {
		t.Fatal("expected invalid date format error")
	}
}

func TestGenerateEdition_LockAlreadyHeld(t *testing.T) {
	rdb := &mockRedis{acquired: false}
	svc := NewService(&mockRepository{}, rdb, nil)
	_, err := svc.GenerateEdition(context.Background(), "2026-06-09")
	if !errors.Is(err, ErrWorkerAlreadyRunning) {
		t.Fatalf("expected ErrWorkerAlreadyRunning, got %v", err)
	}
}

func TestGenerateEdition_RedisErrorContinues(t *testing.T) {
	rdb := &mockRedis{err: errors.New("redis error")}
	repo := &mockRepository{
		exists:    false,
		summaries: []entity.EditionArticleSummary{{ClusterID: uuid.New(), Title: "Title", ArticleCount: 1}},
	}
	svc := NewService(repo, rdb, nil)
	_, err := svc.GenerateEdition(context.Background(), "2026-06-09")
	if err != nil {
		t.Fatalf("expected redis error to be ignored and generation to succeed, got %v", err)
	}
}

func TestGenerateEdition_DuplicateEdition(t *testing.T) {
	rdb := &mockRedis{acquired: true}
	repo := &mockRepository{exists: true}
	svc := NewService(repo, rdb, nil)
	_, err := svc.GenerateEdition(context.Background(), "2026-06-09")
	if !errors.Is(err, edRepo.ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}
}

func TestGenerateEdition_ExistsCheckError(t *testing.T) {
	repo := &mockRepository{existsErr: errors.New("exists check failed")}
	svc := NewService(repo, nil, nil)
	_, err := svc.GenerateEdition(context.Background(), "2026-06-09")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGenerateEdition_NoSummaries(t *testing.T) {
	rdb := &mockRedis{acquired: true}
	repo := &mockRepository{exists: false, summaries: nil}
	svc := NewService(repo, rdb, nil)
	_, err := svc.GenerateEdition(context.Background(), "2026-06-09")
	if !errors.Is(err, ErrNoSummariesFound) {
		t.Fatalf("expected ErrNoSummariesFound, got %v", err)
	}
}

func TestGenerateEdition_LoadSummariesError(t *testing.T) {
	repo := &mockRepository{exists: false, loadErr: errors.New("load failed")}
	svc := NewService(repo, nil, nil)
	_, err := svc.GenerateEdition(context.Background(), "2026-06-09")
	if err == nil {
		t.Fatal("expected load error")
	}
}

func TestGenerateEdition_CreateEditionError(t *testing.T) {
	repo := &mockRepository{
		exists:    false,
		summaries: []entity.EditionArticleSummary{{ClusterID: uuid.New(), Title: "Title", ArticleCount: 1}},
		createErr: errors.New("insert failed"),
	}
	svc := NewService(repo, nil, nil)
	_, err := svc.GenerateEdition(context.Background(), "2026-06-09")
	if err == nil {
		t.Fatal("expected insert error")
	}
}

func TestGetStats(t *testing.T) {
	repo := &mockRepository{statsTotal: 12, statsLatest: "2026-06-09"}
	svc := NewService(repo, nil, nil)
	total, latest, err := svc.GetStats(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 12 || latest != "2026-06-09" {
		t.Fatalf("unexpected stats: total=%d latest=%s", total, latest)
	}

	repo = &mockRepository{statsErr: errors.New("stats error")}
	svc = NewService(repo, nil, nil)
	_, _, err = svc.GetStats(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestListEditions(t *testing.T) {
	repo := &mockRepository{
		editions:  []entity.Edition{{Title: "E1"}},
		listTotal: 1,
	}
	svc := NewService(repo, nil, nil)
	list, total, err := svc.ListEditions(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 || len(list) != 1 || list[0].Title != "E1" {
		t.Fatalf("unexpected list: %+v, total=%d", list, total)
	}

	repo = &mockRepository{listErr: errors.New("list failed")}
	svc = NewService(repo, nil, nil)
	_, _, err = svc.ListEditions(context.Background(), 1, 10)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetEditionByID(t *testing.T) {
	details := &entity.EditionDetailResponse{
		Title: "Details Title",
	}
	repo := &mockRepository{
		details: details,
	}
	svc := NewService(repo, nil, nil)
	res, err := svc.GetEditionByID(context.Background(), uuid.New().String())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Title != "Details Title" {
		t.Fatalf("unexpected details title: %s", res.Title)
	}

	repo = &mockRepository{detailsErr: errors.New("details failed")}
	svc = NewService(repo, nil, nil)
	_, err = svc.GetEditionByID(context.Background(), uuid.New().String())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetEditionByDate(t *testing.T) {
	edID := uuid.New()
	repo := &mockRepository{
		getEd:   &entity.Edition{ID: edID},
		details: &entity.EditionDetailResponse{Title: "Details Title"},
	}
	svc := NewService(repo, nil, nil)
	res, err := svc.GetEditionByDate(context.Background(), "2026-06-09")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Title != "Details Title" {
		t.Fatalf("unexpected title: %s", res.Title)
	}

	_, err = svc.GetEditionByDate(context.Background(), "invalid-date")
	if err == nil {
		t.Fatal("expected validation error")
	}

	repo = &mockRepository{getEdErr: errors.New("get by date failed")}
	svc = NewService(repo, nil, nil)
	_, err = svc.GetEditionByDate(context.Background(), "2026-06-09")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMapCategory(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"politik", "Politics"},
		{"nasional", "National"},
		{"bisnis", "Economy"},
		{"ekonomi", "Economy"},
		{"internasional", "International"},
		{"teknologi", "Technology"},
		{"olahraga", "Sports"},
		{"kesehatan", "Health"},
		{"pendidikan", "Education"},
		{"travel", "Lifestyle"},
		{"unknown", "General"},
		{"  POLITIK  ", "Politics"},
	}

	for _, tc := range tests {
		got := MapCategory(tc.input)
		if got != tc.expected {
			t.Errorf("MapCategory(%q) = %q; want %q", tc.input, got, tc.expected)
		}
	}
}

func TestSortSummaries(t *testing.T) {
	now := time.Now()
	// Test tie-breaking sorting
	summaries := []entity.EditionArticleSummary{
		{
			Title:        "Lowest priority",
			ArticleCount: 1,
			Confidence:   0.8,
			GeneratedAt:  now.Add(-1 * time.Hour),
		},
		{
			Title:        "Higher count",
			ArticleCount: 5,
			Confidence:   0.5,
			GeneratedAt:  now,
		},
		{
			Title:        "Same count, higher confidence",
			ArticleCount: 1,
			Confidence:   0.9,
			GeneratedAt:  now,
		},
		{
			Title:        "Same count, same confidence, more recent",
			ArticleCount: 1,
			Confidence:   0.9,
			GeneratedAt:  now.Add(1 * time.Hour),
		},
	}

	sortSummaries(summaries)

	if summaries[0].Title != "Higher count" {
		t.Errorf("expected index 0 to be 'Higher count', got %q", summaries[0].Title)
	}
	if summaries[1].Title != "Same count, same confidence, more recent" {
		t.Errorf("expected index 1 to be 'Same count, same confidence, more recent', got %q", summaries[1].Title)
	}
	if summaries[2].Title != "Same count, higher confidence" {
		t.Errorf("expected index 2 to be 'Same count, higher confidence', got %q", summaries[2].Title)
	}
	if summaries[3].Title != "Lowest priority" {
		t.Errorf("expected index 3 to be 'Lowest priority', got %q", summaries[3].Title)
	}
}
