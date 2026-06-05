package service_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	articleEntity "koran-ai-backend/internal/article/entity"
	articleRepo "koran-ai-backend/internal/article/repository"
	"koran-ai-backend/internal/crawler/rss"
	"koran-ai-backend/internal/crawler/service"
	sourceEntity "koran-ai-backend/internal/source/entity"
)

// MockSourceRepository implements service.SourceRepository for testing.
type MockSourceRepository struct {
	GetByIDFunc   func(ctx context.Context, id string) (*sourceEntity.Source, error)
	ListActiveFunc func(ctx context.Context) ([]sourceEntity.Source, error)
	ListFunc      func(ctx context.Context, page, limit int) ([]sourceEntity.Source, int64, error)
}

func (m *MockSourceRepository) GetByID(ctx context.Context, id string) (*sourceEntity.Source, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockSourceRepository) ListActive(ctx context.Context) ([]sourceEntity.Source, error) {
	if m.ListActiveFunc != nil {
		return m.ListActiveFunc(ctx)
	}
	return nil, nil
}

func (m *MockSourceRepository) List(ctx context.Context, page, limit int) ([]sourceEntity.Source, int64, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, page, limit)
	}
	return nil, 0, nil
}

// MockArticleRepository implements articleRepo.Repository for testing.
type MockArticleRepository struct {
	CreateFunc          func(ctx context.Context, a *articleEntity.Article) error
	ExistsByURLFunc      func(ctx context.Context, url string) (bool, error)
	ExistsByHashFunc      func(ctx context.Context, hash string) (bool, error)
	ListUnprocessedFunc func(ctx context.Context, limit int) ([]articleEntity.Article, error)
	GetByIDFunc         func(ctx context.Context, id string) (*articleEntity.Article, error)
	ListUnprocessedForAIFunc func(ctx context.Context, limit int) ([]articleEntity.Article, error)
	ListUnclusteredFunc func(ctx context.Context, limit int) ([]articleEntity.Article, error)
	CountTotalFunc      func(ctx context.Context) (int64, error)
	CountTodayFunc      func(ctx context.Context) (int64, error)
}

func (m *MockArticleRepository) Create(ctx context.Context, a *articleEntity.Article) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, a)
	}
	return nil
}

func (m *MockArticleRepository) ExistsByURL(ctx context.Context, url string) (bool, error) {
	if m.ExistsByURLFunc != nil {
		return m.ExistsByURLFunc(ctx, url)
	}
	return false, nil
}

func (m *MockArticleRepository) ExistsByHash(ctx context.Context, hash string) (bool, error) {
	if m.ExistsByHashFunc != nil {
		return m.ExistsByHashFunc(ctx, hash)
	}
	return false, nil
}

func (m *MockArticleRepository) ListUnprocessed(ctx context.Context, limit int) ([]articleEntity.Article, error) {
	if m.ListUnprocessedFunc != nil {
		return m.ListUnprocessedFunc(ctx, limit)
	}
	return nil, nil
}

func (m *MockArticleRepository) GetByID(ctx context.Context, id string) (*articleEntity.Article, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *MockArticleRepository) ListUnprocessedForAI(ctx context.Context, limit int) ([]articleEntity.Article, error) {
	if m.ListUnprocessedForAIFunc != nil {
		return m.ListUnprocessedForAIFunc(ctx, limit)
	}
	return nil, nil
}

func (m *MockArticleRepository) ListUnclustered(ctx context.Context, limit int) ([]articleEntity.Article, error) {
	if m.ListUnclusteredFunc != nil {
		return m.ListUnclusteredFunc(ctx, limit)
	}
	return nil, nil
}

func (m *MockArticleRepository) CountTotal(ctx context.Context) (int64, error) {
	if m.CountTotalFunc != nil {
		return m.CountTotalFunc(ctx)
	}
	return 0, nil
}

func (m *MockArticleRepository) CountToday(ctx context.Context) (int64, error) {
	if m.CountTodayFunc != nil {
		return m.CountTodayFunc(ctx)
	}
	return 0, nil
}

func (m *MockArticleRepository) UpdateAIResult(ctx context.Context, id string, category string, confidence float64) error {
	return nil
}

func (m *MockArticleRepository) UpdateAIError(ctx context.Context, id string, errMsg string) error {
	return nil
}

func (m *MockArticleRepository) CountAIPending(ctx context.Context) (int64, error) {
	return 0, nil
}

func (m *MockArticleRepository) CountAIFailed(ctx context.Context) (int64, error) {
	return 0, nil
}

func (m *MockArticleRepository) CountAIProcessed(ctx context.Context) (int64, error) {
	return 0, nil
}

// MockCrawlLogRepository implements service.CrawlLogRepository for testing.
type MockCrawlLogRepository struct {
	CreateFunc func(ctx context.Context, log *service.CrawlLog) (int64, error)
	FinishFunc func(ctx context.Context, id int64, status string, found, saved, skipped int, durationMs int64, errMsg string) error
	GetStatsFunc func(ctx context.Context) (*time.Time, int64, error)
}

func (m *MockCrawlLogRepository) Create(ctx context.Context, log *service.CrawlLog) (int64, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, log)
	}
	return 1, nil
}

func (m *MockCrawlLogRepository) Finish(ctx context.Context, id int64, status string, found, saved, skipped int, durationMs int64, errMsg string) error {
	if m.FinishFunc != nil {
		return m.FinishFunc(ctx, id, status, found, saved, skipped, durationMs, errMsg)
	}
	return nil
}

func (m *MockCrawlLogRepository) GetStats(ctx context.Context) (*time.Time, int64, error) {
	if m.GetStatsFunc != nil {
		return m.GetStatsFunc(ctx)
	}
	return nil, 0, nil
}

// MockParser implements rss.Parser for testing.
type MockParser struct {
	ParseFunc func(ctx context.Context, rssURL string) ([]rss.FeedItem, error)
}

func (m *MockParser) Parse(ctx context.Context, rssURL string) ([]rss.FeedItem, error) {
	if m.ParseFunc != nil {
		return m.ParseFunc(ctx, rssURL)
	}
	return nil, nil
}

func TestCrawlerService_RunSource_Success(t *testing.T) {
	sourceID := uuid.New()
	src := &sourceEntity.Source{
		ID:         sourceID,
		Name:       "Test Source",
		BaseURL:    "https://example.com",
		RSSURL:     "https://example.com/rss",
		SourceType: "rss",
		IsActive:   true,
	}

	pubTime := time.Now().Add(-1 * time.Hour)
	feedItems := []rss.FeedItem{
		{
			Title:       "News 1",
			URL:         "https://example.com/news-1",
			Author:      "Author 1",
			PublishedAt: &pubTime,
			Content:     "Content of news 1 that is long enough to pass the quality filter minimum length requirement of one hundred characters.",
			ImageURL:    "https://example.com/img-1.jpg",
		},
	}

	sourceRepo := &MockSourceRepository{
		GetByIDFunc: func(ctx context.Context, id string) (*sourceEntity.Source, error) {
			if id == sourceID.String() {
				return src, nil
			}
			return nil, errors.New("not found")
		},
	}

	createdArticles := []*articleEntity.Article{}
	articleRepo := &MockArticleRepository{
		ExistsByURLFunc: func(ctx context.Context, url string) (bool, error) {
			return false, nil
		},
		ExistsByHashFunc: func(ctx context.Context, hash string) (bool, error) {
			return false, nil
		},
		CreateFunc: func(ctx context.Context, a *articleEntity.Article) error {
			createdArticles = append(createdArticles, a)
			return nil
		},
	}

	crawlLogRepo := &MockCrawlLogRepository{}
	parser := &MockParser{
		ParseFunc: func(ctx context.Context, rssURL string) ([]rss.FeedItem, error) {
			if rssURL == src.RSSURL {
				return feedItems, nil
			}
			return nil, errors.New("invalid url")
		},
	}

	svc := service.NewCrawlerService(sourceRepo, articleRepo, crawlLogRepo, parser)
	res, err := svc.RunSource(context.Background(), sourceID.String())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.SourceID != sourceID.String() {
		t.Errorf("expected source id %q, got %q", sourceID.String(), res.SourceID)
	}
	if res.ArticlesFound != 1 {
		t.Errorf("expected 1 article found, got %d", res.ArticlesFound)
	}
	if res.ArticlesSaved != 1 {
		t.Errorf("expected 1 article saved, got %d", res.ArticlesSaved)
	}

	if len(createdArticles) != 1 {
		t.Fatalf("expected 1 article created, got %d", len(createdArticles))
	}

	created := createdArticles[0]
	if created.Title != "News 1" {
		t.Errorf("expected title 'News 1', got %q", created.Title)
	}
	if created.SourceID != sourceID {
		t.Errorf("expected source id %v, got %v", sourceID, created.SourceID)
	}
}

func TestCrawlerService_RunSource_InactiveSource(t *testing.T) {
	sourceID := uuid.New()
	src := &sourceEntity.Source{
		ID:       sourceID,
		IsActive: false,
	}

	sourceRepo := &MockSourceRepository{
		GetByIDFunc: func(ctx context.Context, id string) (*sourceEntity.Source, error) {
			return src, nil
		},
	}

	svc := service.NewCrawlerService(sourceRepo, &MockArticleRepository{}, &MockCrawlLogRepository{}, &MockParser{})
	_, err := svc.RunSource(context.Background(), sourceID.String())
	if !errors.Is(err, service.ErrSourceInactive) {
		t.Errorf("expected ErrSourceInactive, got %v", err)
	}
}

func TestCrawlerService_RunSource_NoRSS(t *testing.T) {
	sourceID := uuid.New()
	src := &sourceEntity.Source{
		ID:       sourceID,
		IsActive: true,
		RSSURL:   "",
	}

	sourceRepo := &MockSourceRepository{
		GetByIDFunc: func(ctx context.Context, id string) (*sourceEntity.Source, error) {
			return src, nil
		},
	}

	svc := service.NewCrawlerService(sourceRepo, &MockArticleRepository{}, &MockCrawlLogRepository{}, &MockParser{})
	_, err := svc.RunSource(context.Background(), sourceID.String())
	if !errors.Is(err, service.ErrSourceNoRSS) {
		t.Errorf("expected ErrSourceNoRSS, got %v", err)
	}
}

func TestCrawlerService_RunSource_Duplicate(t *testing.T) {
	sourceID := uuid.New()
	src := &sourceEntity.Source{
		ID:       sourceID,
		IsActive: true,
		RSSURL:   "https://example.com/rss",
	}

	feedItems := []rss.FeedItem{
		{Title: "News 1", URL: "https://example.com/news-1"},
	}

	sourceRepo := &MockSourceRepository{
		GetByIDFunc: func(ctx context.Context, id string) (*sourceEntity.Source, error) {
			return src, nil
		},
	}

	articleRepo := &MockArticleRepository{
		ExistsByURLFunc: func(ctx context.Context, url string) (bool, error) {
			return true, nil // Already exists by URL
		},
	}

	parser := &MockParser{
		ParseFunc: func(ctx context.Context, rssURL string) ([]rss.FeedItem, error) {
			return feedItems, nil
		},
	}

	svc := service.NewCrawlerService(sourceRepo, articleRepo, &MockCrawlLogRepository{}, parser)
	res, err := svc.RunSource(context.Background(), sourceID.String())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.ArticlesSaved != 0 {
		t.Errorf("expected 0 articles saved due to URL duplicate, got %d", res.ArticlesSaved)
	}
}

func TestCrawlerService_RunSource_GetSourceError(t *testing.T) {
	sourceRepo := &MockSourceRepository{
		GetByIDFunc: func(ctx context.Context, id string) (*sourceEntity.Source, error) {
			return nil, errors.New("db error")
		},
	}
	svc := service.NewCrawlerService(sourceRepo, &MockArticleRepository{}, &MockCrawlLogRepository{}, &MockParser{})
	_, err := svc.RunSource(context.Background(), uuid.New().String())
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestCrawlerService_RunSource_CrawlLogCreateError(t *testing.T) {
	sourceID := uuid.New()
	src := &sourceEntity.Source{
		ID:         sourceID,
		Name:       "Test",
		RSSURL:     "https://example.com/rss",
		IsActive:   true,
		SourceType: "rss",
	}
	sourceRepo := &MockSourceRepository{
		GetByIDFunc: func(ctx context.Context, id string) (*sourceEntity.Source, error) {
			return src, nil
		},
	}
	crawlLogRepo := &MockCrawlLogRepository{
		CreateFunc: func(ctx context.Context, log *service.CrawlLog) (int64, error) {
			return 0, errors.New("log create fail")
		},
	}
	parser := &MockParser{
		ParseFunc: func(ctx context.Context, rssURL string) ([]rss.FeedItem, error) {
			return []rss.FeedItem{}, nil
		},
	}
	svc := service.NewCrawlerService(sourceRepo, &MockArticleRepository{}, crawlLogRepo, parser)
	res, err := svc.RunSource(context.Background(), sourceID.String())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if res.ArticlesFound != 0 {
		t.Errorf("expected 0 found, got %d", res.ArticlesFound)
	}
}

func TestCrawlerService_RunSource_ParserError(t *testing.T) {
	sourceID := uuid.New()
	src := &sourceEntity.Source{
		ID:         sourceID,
		Name:       "Test",
		RSSURL:     "https://example.com/rss",
		IsActive:   true,
		SourceType: "rss",
	}
	sourceRepo := &MockSourceRepository{
		GetByIDFunc: func(ctx context.Context, id string) (*sourceEntity.Source, error) {
			return src, nil
		},
	}
	parser := &MockParser{
		ParseFunc: func(ctx context.Context, rssURL string) ([]rss.FeedItem, error) {
			return nil, errors.New("parse error")
		},
	}
	svc := service.NewCrawlerService(sourceRepo, &MockArticleRepository{}, &MockCrawlLogRepository{}, parser)
	res, err := svc.RunSource(context.Background(), sourceID.String())
	if err == nil {
		t.Error("expected parser error, got nil")
	}
	if res == nil || res.ArticlesFound != 0 {
		t.Errorf("expected 0 articles found on error, got %v", res)
	}
}

func TestCrawlerService_RunSource_HashDuplicateAndErrors(t *testing.T) {
	sourceID := uuid.New()
	src := &sourceEntity.Source{
		ID:         sourceID,
		Name:       "Test",
		RSSURL:     "https://example.com/rss",
		IsActive:   true,
		SourceType: "rss",
	}
	sourceRepo := &MockSourceRepository{
		GetByIDFunc: func(ctx context.Context, id string) (*sourceEntity.Source, error) {
			return src, nil
		},
	}
	feedItems := []rss.FeedItem{
		{Title: "News 1", URL: "https://example.com/news-1", Content: "Content 1"},
		{Title: "", URL: "https://example.com/news-2"}, // skipped
		{Title: "News 3", URL: ""},                     // skipped
	}
	parser := &MockParser{
		ParseFunc: func(ctx context.Context, rssURL string) ([]rss.FeedItem, error) {
			return feedItems, nil
		},
	}

	// First run: ExistsByURL returns error (item skipped)
	articleRepo := &MockArticleRepository{
		ExistsByURLFunc: func(ctx context.Context, url string) (bool, error) {
			return false, errors.New("url check error")
		},
	}
	svc := service.NewCrawlerService(sourceRepo, articleRepo, &MockCrawlLogRepository{}, parser)
	res, err := svc.RunSource(context.Background(), sourceID.String())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if res.ArticlesSaved != 0 {
		t.Errorf("expected 0 saved, got %d", res.ArticlesSaved)
	}

	// Second run: ExistsByHash returns true (duplicate hash, item skipped)
	articleRepo = &MockArticleRepository{
		ExistsByURLFunc: func(ctx context.Context, url string) (bool, error) {
			return false, nil
		},
		ExistsByHashFunc: func(ctx context.Context, hash string) (bool, error) {
			return true, nil
		},
	}
	svc = service.NewCrawlerService(sourceRepo, articleRepo, &MockCrawlLogRepository{}, parser)
	res, err = svc.RunSource(context.Background(), sourceID.String())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if res.ArticlesSaved != 0 {
		t.Errorf("expected 0 saved, got %d", res.ArticlesSaved)
	}

	// Third run: ExistsByHash returns error (item skipped)
	articleRepo = &MockArticleRepository{
		ExistsByURLFunc: func(ctx context.Context, url string) (bool, error) {
			return false, nil
		},
		ExistsByHashFunc: func(ctx context.Context, hash string) (bool, error) {
			return false, errors.New("hash check error")
		},
	}
	svc = service.NewCrawlerService(sourceRepo, articleRepo, &MockCrawlLogRepository{}, parser)
	res, err = svc.RunSource(context.Background(), sourceID.String())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if res.ArticlesSaved != 0 {
		t.Errorf("expected 0 saved, got %d", res.ArticlesSaved)
	}

	// Fourth run: Create returns duplicate error (handled gracefully, item skipped)
	articleRepo = &MockArticleRepository{
		ExistsByURLFunc: func(ctx context.Context, url string) (bool, error) {
			return false, nil
		},
		ExistsByHashFunc: func(ctx context.Context, hash string) (bool, error) {
			return false, nil
		},
		CreateFunc: func(ctx context.Context, a *articleEntity.Article) error {
			return errors.New("article url already exists") // matches articleRepo.ErrDuplicateURL via mock implementation
		},
	}
	// Note: since our mock doesn't define errors.Is exactly like original, let's inject a custom error check.
	// Actually, our repository package defines ErrDuplicateURL and ErrDuplicateHash. Let's make mock repository returns those:
}

func TestCrawlerService_RunSource_CreateRepoErrors(t *testing.T) {
	sourceID := uuid.New()
	src := &sourceEntity.Source{
		ID:         sourceID,
		Name:       "Test",
		RSSURL:     "https://example.com/rss",
		IsActive:   true,
		SourceType: "rss",
	}
	sourceRepo := &MockSourceRepository{
		GetByIDFunc: func(ctx context.Context, id string) (*sourceEntity.Source, error) {
			return src, nil
		},
	}
	feedItems := []rss.FeedItem{
		{Title: "News 1", URL: "https://example.com/news-1", Content: "Content 1"},
	}
	parser := &MockParser{
		ParseFunc: func(ctx context.Context, rssURL string) ([]rss.FeedItem, error) {
			return feedItems, nil
		},
	}

	// Create returns ErrDuplicateURL
	articleRepo := &MockArticleRepository{
		ExistsByURLFunc: func(ctx context.Context, url string) (bool, error) {
			return false, nil
		},
		ExistsByHashFunc: func(ctx context.Context, hash string) (bool, error) {
			return false, nil
		},
		CreateFunc: func(ctx context.Context, a *articleEntity.Article) error {
			// In service.go, we import: articleRepo "koran-ai-backend/internal/article/repository"
			// and check errors.Is(err, articleRepo.ErrDuplicateURL)
			// So let's return that exact error.
			// Let's import the repo package inside the test to use it.
			// Wait, the test package is service_test, so we can import it.
			return articleRepo.ErrDuplicateURL
		},
	}
	svc := service.NewCrawlerService(sourceRepo, articleRepo, &MockCrawlLogRepository{}, parser)
	res, err := svc.RunSource(context.Background(), sourceID.String())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if res.ArticlesSaved != 0 {
		t.Errorf("expected 0 saved, got %d", res.ArticlesSaved)
	}

	// Create returns a generic DB error
	articleRepo = &MockArticleRepository{
		ExistsByURLFunc: func(ctx context.Context, url string) (bool, error) {
			return false, nil
		},
		ExistsByHashFunc: func(ctx context.Context, hash string) (bool, error) {
			return false, nil
		},
		CreateFunc: func(ctx context.Context, a *articleEntity.Article) error {
			return errors.New("db generic error")
		},
	}
	svc = service.NewCrawlerService(sourceRepo, articleRepo, &MockCrawlLogRepository{}, parser)
	res, err = svc.RunSource(context.Background(), sourceID.String())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if res.ArticlesSaved != 0 {
		t.Errorf("expected 0 saved, got %d", res.ArticlesSaved)
	}
}

func TestCrawlerService_RunAllSources_ErrorListActive(t *testing.T) {
	sourceRepo := &MockSourceRepository{
		ListActiveFunc: func(ctx context.Context) ([]sourceEntity.Source, error) {
			return nil, errors.New("db error")
		},
	}
	svc := service.NewCrawlerService(sourceRepo, &MockArticleRepository{}, &MockCrawlLogRepository{}, &MockParser{})
	_, err := svc.RunAllSources(context.Background())
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestCrawlerService_RunAllSources_IndividualSourceError(t *testing.T) {
	src1 := sourceEntity.Source{ID: uuid.New(), RSSURL: "https://example.com/rss1", IsActive: true}
	src2 := sourceEntity.Source{ID: uuid.New(), RSSURL: "https://example.com/rss2", IsActive: true}

	sourceRepo := &MockSourceRepository{
		ListActiveFunc: func(ctx context.Context) ([]sourceEntity.Source, error) {
			return []sourceEntity.Source{src1, src2}, nil
		},
		GetByIDFunc: func(ctx context.Context, id string) (*sourceEntity.Source, error) {
			if id == src1.ID.String() {
				return nil, errors.New("lookup error")
			}
			return &src2, nil
		},
	}

	parser := &MockParser{
		ParseFunc: func(ctx context.Context, rssURL string) ([]rss.FeedItem, error) {
			return []rss.FeedItem{{Title: "News", URL: "https://example.com/news"}}, nil
		},
	}

	svc := service.NewCrawlerService(sourceRepo, &MockArticleRepository{}, &MockCrawlLogRepository{}, parser)
	results, err := svc.RunAllSources(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	// First result must have error
	if results[0].Error == nil {
		t.Error("expected error for first source")
	}
	// Second result must be success
	if results[1].Error != nil {
		t.Errorf("expected no error for second source, got %v", results[1].Error)
	}
}

func TestCrawlerService_Normalization(t *testing.T) {
	input := "\r\n  This  is \t a   test \n\n\n  content  with   spaces.  \r\n"
	expected := "This is a test\n\ncontent with spaces."
	got := service.NormalizeText(input)
	if got != expected {
		t.Errorf("expected normalized %q, got %q", expected, got)
	}

	title := "Go 1.25! Released & Ready - For Production??"
	expectedSlug := "go-125-released-ready-for-production"
	gotSlug := service.ToSlug(title)
	if gotSlug != expectedSlug {
		t.Errorf("expected slug %q, got %q", expectedSlug, gotSlug)
	}
}

func TestCrawlerService_QualityFilter(t *testing.T) {
	sourceID := uuid.New()
	src := &sourceEntity.Source{
		ID:         sourceID,
		Name:       "Test",
		RSSURL:     "https://example.com/rss",
		IsActive:   true,
		SourceType: "rss",
	}
	sourceRepo := &MockSourceRepository{
		GetByIDFunc: func(ctx context.Context, id string) (*sourceEntity.Source, error) {
			return src, nil
		},
	}

	pubTime := time.Now()
	feedItems := []rss.FeedItem{
		// 1. Empty title -> skipped
		{Title: "   ", URL: "https://example.com/1", Content: strings.Repeat("a", 150), PublishedAt: &pubTime},
		// 2. Short content (< 100 char) -> skipped
		{Title: "Valid Title", URL: "https://example.com/2", Content: "too short content", PublishedAt: &pubTime},
		// 3. Nil published_at -> skipped
		{Title: "Valid Title 2", URL: "https://example.com/3", Content: strings.Repeat("a", 150), PublishedAt: nil},
		// 4. Valid item -> saved
		{Title: "Valid Title 3", URL: "https://example.com/4", Content: strings.Repeat("a", 150), PublishedAt: &pubTime},
	}

	parser := &MockParser{
		ParseFunc: func(ctx context.Context, rssURL string) ([]rss.FeedItem, error) {
			return feedItems, nil
		},
	}

	var finishSkipped int
	crawlLogRepo := &MockCrawlLogRepository{
		FinishFunc: func(ctx context.Context, id int64, status string, found, saved, skipped int, durationMs int64, errMsg string) error {
			finishSkipped = skipped
			return nil
		},
	}

	articleRepo := &MockArticleRepository{}

	svc := service.NewCrawlerService(sourceRepo, articleRepo, crawlLogRepo, parser)
	res, err := svc.RunSource(context.Background(), sourceID.String())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.ArticlesSaved != 1 {
		t.Errorf("expected 1 saved article, got %d", res.ArticlesSaved)
	}
	if finishSkipped != 3 {
		t.Errorf("expected 3 skipped articles recorded in finish log, got %d", finishSkipped)
	}
}

func TestCrawlerService_GetStats(t *testing.T) {
	lastCrawlTime := time.Now().Add(-10 * time.Minute)

	sourceRepo := &MockSourceRepository{
		ListFunc: func(ctx context.Context, page, limit int) ([]sourceEntity.Source, int64, error) {
			return nil, 10, nil
		},
	}

	articleRepo := &MockArticleRepository{
		CountTotalFunc: func(ctx context.Context) (int64, error) {
			return 12000, nil
		},
		CountTodayFunc: func(ctx context.Context) (int64, error) {
			return 350, nil
		},
	}

	crawlLogRepo := &MockCrawlLogRepository{
		GetStatsFunc: func(ctx context.Context) (*time.Time, int64, error) {
			return &lastCrawlTime, 2, nil
		},
	}

	svc := service.NewCrawlerService(sourceRepo, articleRepo, crawlLogRepo, &MockParser{})
	stats, err := svc.GetStats(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stats.Sources != 10 {
		t.Errorf("expected 10 sources, got %d", stats.Sources)
	}
	if stats.ArticlesTotal != 12000 {
		t.Errorf("expected 12000 articles total, got %d", stats.ArticlesTotal)
	}
	if stats.ArticlesToday != 350 {
		t.Errorf("expected 350 articles today, got %d", stats.ArticlesToday)
	}
	if stats.FailedCrawls != 2 {
		t.Errorf("expected 2 failed crawls, got %d", stats.FailedCrawls)
	}
	if stats.LastCrawl == "" {
		t.Error("expected last crawl timestamp to be set")
	}
}


func TestCrawlerService_GetStats_NilLastCrawl(t *testing.T) {
	sourceRepo := &MockSourceRepository{
		ListFunc: func(ctx context.Context, page, limit int) ([]sourceEntity.Source, int64, error) {
			return nil, 3, nil
		},
	}
	articleRepo := &MockArticleRepository{
		CountTotalFunc: func(ctx context.Context) (int64, error) { return 50, nil },
		CountTodayFunc: func(ctx context.Context) (int64, error) { return 5, nil },
	}
	crawlLogRepo := &MockCrawlLogRepository{
		GetStatsFunc: func(ctx context.Context) (*time.Time, int64, error) {
			return nil, 0, nil // no crawl yet
		},
	}

	svc := service.NewCrawlerService(sourceRepo, articleRepo, crawlLogRepo, &MockParser{})
	stats, err := svc.GetStats(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.LastCrawl != "" {
		t.Errorf("expected empty last_crawl when nil, got %q", stats.LastCrawl)
	}
}

func TestCrawlerService_GetStats_SourcesError(t *testing.T) {
	sourceRepo := &MockSourceRepository{
		ListFunc: func(ctx context.Context, page, limit int) ([]sourceEntity.Source, int64, error) {
			return nil, 0, errors.New("db error")
		},
	}
	svc := service.NewCrawlerService(sourceRepo, &MockArticleRepository{}, &MockCrawlLogRepository{}, &MockParser{})
	_, err := svc.GetStats(context.Background())
	if err == nil {
		t.Error("expected error from GetStats when sources query fails")
	}
}

func TestCrawlerService_GetStats_ArticlesTotalError(t *testing.T) {
	sourceRepo := &MockSourceRepository{
		ListFunc: func(ctx context.Context, page, limit int) ([]sourceEntity.Source, int64, error) {
			return nil, 5, nil
		},
	}
	articleRepo := &MockArticleRepository{
		CountTotalFunc: func(ctx context.Context) (int64, error) {
			return 0, errors.New("articles total error")
		},
	}
	svc := service.NewCrawlerService(sourceRepo, articleRepo, &MockCrawlLogRepository{}, &MockParser{})
	_, err := svc.GetStats(context.Background())
	if err == nil {
		t.Error("expected error from GetStats when articles total query fails")
	}
}

func TestCrawlerService_GetStats_ArticlesTodayError(t *testing.T) {
	sourceRepo := &MockSourceRepository{
		ListFunc: func(ctx context.Context, page, limit int) ([]sourceEntity.Source, int64, error) {
			return nil, 5, nil
		},
	}
	articleRepo := &MockArticleRepository{
		CountTotalFunc: func(ctx context.Context) (int64, error) { return 100, nil },
		CountTodayFunc: func(ctx context.Context) (int64, error) {
			return 0, errors.New("articles today error")
		},
	}
	svc := service.NewCrawlerService(sourceRepo, articleRepo, &MockCrawlLogRepository{}, &MockParser{})
	_, err := svc.GetStats(context.Background())
	if err == nil {
		t.Error("expected error from GetStats when articles today query fails")
	}
}

func TestCrawlerService_GetStats_CrawlLogError(t *testing.T) {
	sourceRepo := &MockSourceRepository{
		ListFunc: func(ctx context.Context, page, limit int) ([]sourceEntity.Source, int64, error) {
			return nil, 5, nil
		},
	}
	articleRepo := &MockArticleRepository{
		CountTotalFunc: func(ctx context.Context) (int64, error) { return 100, nil },
		CountTodayFunc: func(ctx context.Context) (int64, error) { return 10, nil },
	}
	crawlLogRepo := &MockCrawlLogRepository{
		GetStatsFunc: func(ctx context.Context) (*time.Time, int64, error) {
			return nil, 0, errors.New("crawl log stats error")
		},
	}
	svc := service.NewCrawlerService(sourceRepo, articleRepo, crawlLogRepo, &MockParser{})
	_, err := svc.GetStats(context.Background())
	if err == nil {
		t.Error("expected error from GetStats when crawl log query fails")
	}
}
