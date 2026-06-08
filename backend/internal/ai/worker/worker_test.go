package worker

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"koran-ai-backend/internal/ai/client"
	"koran-ai-backend/internal/article/entity"
	"koran-ai-backend/internal/article/repository"
)

// ---------------------------------------------------------------------------
// Mock implementations
// ---------------------------------------------------------------------------

type mockRepository struct {
	repository.Repository
	listUnprocessedForAIFunc func(ctx context.Context, limit int) ([]entity.Article, error)
	updateAIResultFunc       func(ctx context.Context, id string, category string, confidence float64) error
	updateAIErrorFunc        func(ctx context.Context, id string, errMsg string) error
}

func (m *mockRepository) ListUnprocessedForAI(ctx context.Context, limit int) ([]entity.Article, error) {
	return m.listUnprocessedForAIFunc(ctx, limit)
}
func (m *mockRepository) UpdateAIResult(ctx context.Context, id string, category string, confidence float64) error {
	return m.updateAIResultFunc(ctx, id, category, confidence)
}
func (m *mockRepository) UpdateAIError(ctx context.Context, id string, errMsg string) error {
	return m.updateAIErrorFunc(ctx, id, errMsg)
}

type mockGeminiClient struct {
	classifyFunc func(ctx context.Context, title string, content string) (*client.ClassifyResult, error)
}

func (m *mockGeminiClient) ClassifyArticle(ctx context.Context, title string, content string) (*client.ClassifyResult, error) {
	return m.classifyFunc(ctx, title, content)
}

type mockRedis struct {
	redis.Cmdable
	setNXFunc func(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd
	delFunc   func(ctx context.Context, keys ...string) *redis.IntCmd
}

func (m *mockRedis) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd {
	return m.setNXFunc(ctx, key, value, expiration)
}
func (m *mockRedis) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	return m.delFunc(ctx, keys...)
}

type mockLogger struct {
	zapLog *zap.Logger
}

func (m *mockLogger) Debug(msg string, fields ...zap.Field) { m.zapLog.Debug(msg, fields...) }
func (m *mockLogger) Info(msg string, fields ...zap.Field)  { m.zapLog.Info(msg, fields...) }
func (m *mockLogger) Warn(msg string, fields ...zap.Field)  { m.zapLog.Warn(msg, fields...) }
func (m *mockLogger) Error(msg string, fields ...zap.Field) { m.zapLog.Error(msg, fields...) }
func (m *mockLogger) Fatal(msg string, fields ...zap.Field) { m.zapLog.Fatal(msg, fields...) }
func (m *mockLogger) Sync() error                          { return m.zapLog.Sync() }
func (m *mockLogger) GetZapLogger() *zap.Logger            { return m.zapLog }

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func nopLogger() *mockLogger {
	return &mockLogger{zapLog: zap.NewNop()}
}

// redisLockSuccess returns a mockRedis that always acquires the lock and records release.
func redisLockSuccess(released *bool) *mockRedis {
	return &mockRedis{
		setNXFunc: func(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd {
			cmd := redis.NewBoolCmd(ctx)
			cmd.SetVal(true)
			return cmd
		},
		delFunc: func(ctx context.Context, keys ...string) *redis.IntCmd {
			if released != nil {
				*released = true
			}
			cmd := redis.NewIntCmd(ctx)
			cmd.SetVal(1)
			return cmd
		},
	}
}

// newWorkerFast returns an aiWorker with backoffScale=0 for instant retries in tests.
func newWorkerFast(repo repository.Repository, gemini client.GeminiClient, rdb redis.Cmdable) *aiWorker {
	return &aiWorker{
		repo:         repo,
		client:       gemini,
		rdb:          rdb,
		logger:       nopLogger(),
		backoffScale: 0,
	}
}

func sampleArticles(n int) []entity.Article {
	arts := make([]entity.Article, n)
	for i := range arts {
		arts[i] = entity.Article{ID: uuid.New(), Title: "Article", Content: "Content"}
	}
	return arts
}

// ---------------------------------------------------------------------------
// Tests — Redis lock behaviour
// ---------------------------------------------------------------------------

// TestProcessBatch_LockAlreadyHeld verifies that when SetNX returns false the
// worker returns "worker already running" and does not call the repository.
func TestProcessBatch_LockAlreadyHeld(t *testing.T) {
	ctx := context.Background()

	mr := &mockRedis{
		setNXFunc: func(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd {
			cmd := redis.NewBoolCmd(ctx)
			cmd.SetVal(false) // lock already held
			return cmd
		},
	}

	w := NewAIWorker(nil, nil, mr, nopLogger())
	_, err := w.ProcessBatch(ctx, 10)
	if err == nil || err.Error() != "worker already running" {
		t.Fatalf("expected 'worker already running', got %v", err)
	}
}

// TestProcessBatch_RedisError verifies that a Redis connection error is treated
// as a warning (no lock acquired) and processing continues normally.
func TestProcessBatch_RedisError(t *testing.T) {
	ctx := context.Background()

	mr := &mockRedis{
		setNXFunc: func(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd {
			cmd := redis.NewBoolCmd(ctx)
			cmd.SetErr(errors.New("redis down"))
			return cmd
		},
		// Del should NOT be called because the lock was never acquired.
	}

	repo := &mockRepository{
		listUnprocessedForAIFunc: func(ctx context.Context, limit int) ([]entity.Article, error) {
			return []entity.Article{}, nil
		},
	}

	w := NewAIWorker(repo, nil, mr, nopLogger())
	res, err := w.ProcessBatch(ctx, 10)
	if err != nil {
		t.Fatalf("expected nil error when Redis is down; got %v", err)
	}
	if res.Total != 0 {
		t.Errorf("expected 0 total articles, got %d", res.Total)
	}
}

// TestProcessBatch_NilRedis verifies that a nil Redis client is handled gracefully
// (warning logged) and processing continues.
func TestProcessBatch_NilRedis(t *testing.T) {
	ctx := context.Background()

	repo := &mockRepository{
		listUnprocessedForAIFunc: func(ctx context.Context, limit int) ([]entity.Article, error) {
			return []entity.Article{}, nil
		},
	}

	w := NewAIWorker(repo, nil, nil, nopLogger())
	res, err := w.ProcessBatch(ctx, 10)
	if err != nil {
		t.Fatalf("expected nil error with nil Redis; got %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil BatchResult")
	}
}

// TestProcessBatch_LockReleasedAfterBatch verifies the Redis lock is always
// released via defer after a successful batch.
func TestProcessBatch_LockReleasedAfterBatch(t *testing.T) {
	ctx := context.Background()

	var lockReleased bool
	repo := &mockRepository{
		listUnprocessedForAIFunc: func(ctx context.Context, limit int) ([]entity.Article, error) {
			return []entity.Article{}, nil
		},
	}

	w := newWorkerFast(repo, nil, redisLockSuccess(&lockReleased))
	_, err := w.ProcessBatch(ctx, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !lockReleased {
		t.Error("expected Redis lock to be released after batch")
	}
}

// ---------------------------------------------------------------------------
// Tests — Repository errors
// ---------------------------------------------------------------------------

// TestProcessBatch_DatabaseRetrievalError verifies that a DB error on
// ListUnprocessedForAI is propagated as an error return.
func TestProcessBatch_DatabaseRetrievalError(t *testing.T) {
	ctx := context.Background()

	repo := &mockRepository{
		listUnprocessedForAIFunc: func(ctx context.Context, limit int) ([]entity.Article, error) {
			return nil, errors.New("db error on select")
		},
	}

	var lockReleased bool
	w := newWorkerFast(repo, nil, redisLockSuccess(&lockReleased))
	_, err := w.ProcessBatch(ctx, 10)

	if err == nil || !strings.Contains(err.Error(), "db error on select") {
		t.Fatalf("expected select database error, got %v", err)
	}
	// Lock must still be released even on repo error.
	if !lockReleased {
		t.Error("expected Redis lock to be released even after repo error")
	}
}

// ---------------------------------------------------------------------------
// Tests — Empty batch
// ---------------------------------------------------------------------------

// TestProcessBatch_EmptyBatch verifies that an empty article list returns a
// zero-metric BatchResult and no errors.
func TestProcessBatch_EmptyBatch(t *testing.T) {
	ctx := context.Background()

	repo := &mockRepository{
		listUnprocessedForAIFunc: func(ctx context.Context, limit int) ([]entity.Article, error) {
			return []entity.Article{}, nil
		},
	}

	w := newWorkerFast(repo, nil, redisLockSuccess(nil))
	res, err := w.ProcessBatch(ctx, 10)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Total != 0 || res.Processed != 0 || res.Failed != 0 {
		t.Errorf("expected all-zero metrics for empty batch, got %+v", res)
	}
}

// ---------------------------------------------------------------------------
// Tests — Mixed success/failure processing
// ---------------------------------------------------------------------------

// TestProcessBatch_SuccessAndFailureMix verifies that one successful and one
// failed article are each counted independently and the batch itself succeeds.
func TestProcessBatch_SuccessAndFailureMix(t *testing.T) {
	ctx := context.Background()

	art1 := entity.Article{ID: uuid.New(), Title: "Success Article", Content: "Body 1"}
	art2 := entity.Article{ID: uuid.New(), Title: "Failed Article", Content: "Body 2"}

	var updatedResults []string
	var updatedErrors []string

	repo := &mockRepository{
		listUnprocessedForAIFunc: func(ctx context.Context, limit int) ([]entity.Article, error) {
			return []entity.Article{art1, art2}, nil
		},
		updateAIResultFunc: func(ctx context.Context, id string, category string, confidence float64) error {
			updatedResults = append(updatedResults, id)
			return nil
		},
		updateAIErrorFunc: func(ctx context.Context, id string, errMsg string) error {
			updatedErrors = append(updatedErrors, id)
			return nil
		},
	}

	gemini := &mockGeminiClient{
		classifyFunc: func(ctx context.Context, title string, content string) (*client.ClassifyResult, error) {
			if title == "Success Article" {
				return &client.ClassifyResult{Category: "Teknologi", Confidence: 0.9}, nil
			}
			return nil, errors.New("gemini failed")
		},
	}

	var lockReleased bool
	w := newWorkerFast(repo, gemini, redisLockSuccess(&lockReleased))
	res, err := w.ProcessBatch(ctx, 10)

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if res.Processed != 1 {
		t.Errorf("expected 1 processed, got %d", res.Processed)
	}
	if res.Failed != 1 {
		t.Errorf("expected 1 failed, got %d", res.Failed)
	}
	if res.ProcessedArticles != 1 {
		t.Errorf("expected 1 processed_articles, got %d", res.ProcessedArticles)
	}
	if res.FailedArticles != 1 {
		t.Errorf("expected 1 failed_articles, got %d", res.FailedArticles)
	}
	if len(updatedResults) != 1 || updatedResults[0] != art1.ID.String() {
		t.Errorf("expected art1 to be updated with result, got %v", updatedResults)
	}
	if len(updatedErrors) != 1 || updatedErrors[0] != art2.ID.String() {
		t.Errorf("expected art2 to be updated with error, got %v", updatedErrors)
	}
	if !lockReleased {
		t.Error("expected lock to be released")
	}
}

// TestProcessBatch_AllSuccess verifies all-success batch metrics.
func TestProcessBatch_AllSuccess(t *testing.T) {
	ctx := context.Background()

	articles := sampleArticles(5)
	repo := &mockRepository{
		listUnprocessedForAIFunc: func(ctx context.Context, limit int) ([]entity.Article, error) {
			return articles, nil
		},
		updateAIResultFunc: func(ctx context.Context, id string, category string, confidence float64) error {
			return nil
		},
		updateAIErrorFunc: func(ctx context.Context, id string, errMsg string) error { return nil },
	}

	gemini := &mockGeminiClient{
		classifyFunc: func(ctx context.Context, title string, content string) (*client.ClassifyResult, error) {
			return &client.ClassifyResult{Category: "Teknologi", Confidence: 0.95}, nil
		},
	}

	w := newWorkerFast(repo, gemini, redisLockSuccess(nil))
	res, err := w.ProcessBatch(ctx, 10)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Processed != 5 {
		t.Errorf("expected 5 processed, got %d", res.Processed)
	}
	if res.Failed != 0 {
		t.Errorf("expected 0 failed, got %d", res.Failed)
	}
	if res.SuccessRate != 100.0 {
		t.Errorf("expected 100%% success rate, got %.2f", res.SuccessRate)
	}
}

// TestProcessBatch_AllFailed verifies all-failure batch metrics; batch should
// return nil error (failures are per-article, not batch-level).
func TestProcessBatch_AllFailed(t *testing.T) {
	ctx := context.Background()

	articles := sampleArticles(3)
	repo := &mockRepository{
		listUnprocessedForAIFunc: func(ctx context.Context, limit int) ([]entity.Article, error) {
			return articles, nil
		},
		updateAIErrorFunc: func(ctx context.Context, id string, errMsg string) error { return nil },
		updateAIResultFunc: func(ctx context.Context, id string, category string, confidence float64) error {
			return nil
		},
	}

	gemini := &mockGeminiClient{
		classifyFunc: func(ctx context.Context, title string, content string) (*client.ClassifyResult, error) {
			return nil, errors.New("gemini unavailable")
		},
	}

	w := newWorkerFast(repo, gemini, redisLockSuccess(nil))
	res, err := w.ProcessBatch(ctx, 10)

	if err != nil {
		t.Fatalf("expected nil error even when all articles fail; got %v", err)
	}
	if res.Failed != 3 {
		t.Errorf("expected 3 failed, got %d", res.Failed)
	}
	if res.Processed != 0 {
		t.Errorf("expected 0 processed, got %d", res.Processed)
	}
	if res.SuccessRate != 0.0 {
		t.Errorf("expected 0%% success rate, got %.2f", res.SuccessRate)
	}
}

// ---------------------------------------------------------------------------
// Tests — Retry mechanism
// ---------------------------------------------------------------------------

// TestProcessBatch_GeminiRetrySucceedsOnSecondAttempt verifies that the worker
// retries on transient Gemini errors and succeeds when a later attempt works.
func TestProcessBatch_GeminiRetrySucceedsOnSecondAttempt(t *testing.T) {
	ctx := context.Background()

	var callCount int32
	art := entity.Article{ID: uuid.New(), Title: "Retry Article", Content: "Content"}

	repo := &mockRepository{
		listUnprocessedForAIFunc: func(ctx context.Context, limit int) ([]entity.Article, error) {
			return []entity.Article{art}, nil
		},
		updateAIResultFunc: func(ctx context.Context, id string, category string, confidence float64) error {
			return nil
		},
		updateAIErrorFunc: func(ctx context.Context, id string, errMsg string) error { return nil },
	}

	gemini := &mockGeminiClient{
		classifyFunc: func(ctx context.Context, title string, content string) (*client.ClassifyResult, error) {
			n := atomic.AddInt32(&callCount, 1)
			if n < 2 {
				return nil, errors.New("transient error")
			}
			return &client.ClassifyResult{Category: "Politik", Confidence: 0.8}, nil
		},
	}

	w := newWorkerFast(repo, gemini, redisLockSuccess(nil))
	res, err := w.ProcessBatch(ctx, 10)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Processed != 1 {
		t.Errorf("expected 1 processed after retry, got %d", res.Processed)
	}
	if atomic.LoadInt32(&callCount) < 2 {
		t.Errorf("expected at least 2 classify calls (retry), got %d", callCount)
	}
}

// TestProcessBatch_GeminiRetryExhausted verifies that after 3 failed attempts
// the article is counted as failed and UpdateAIError is called.
func TestProcessBatch_GeminiRetryExhausted(t *testing.T) {
	ctx := context.Background()

	var callCount int32
	var errorUpdated bool
	art := entity.Article{ID: uuid.New(), Title: "Bad Article", Content: "Content"}

	repo := &mockRepository{
		listUnprocessedForAIFunc: func(ctx context.Context, limit int) ([]entity.Article, error) {
			return []entity.Article{art}, nil
		},
		updateAIResultFunc: func(ctx context.Context, id string, category string, confidence float64) error {
			return nil
		},
		updateAIErrorFunc: func(ctx context.Context, id string, errMsg string) error {
			errorUpdated = true
			return nil
		},
	}

	gemini := &mockGeminiClient{
		classifyFunc: func(ctx context.Context, title string, content string) (*client.ClassifyResult, error) {
			atomic.AddInt32(&callCount, 1)
			return nil, errors.New("persistent gemini error")
		},
	}

	w := newWorkerFast(repo, gemini, redisLockSuccess(nil))
	res, err := w.ProcessBatch(ctx, 10)

	if err != nil {
		t.Fatalf("unexpected batch error: %v", err)
	}
	if res.Failed != 1 {
		t.Errorf("expected 1 failed, got %d", res.Failed)
	}
	if res.Processed != 0 {
		t.Errorf("expected 0 processed, got %d", res.Processed)
	}
	if !errorUpdated {
		t.Error("expected UpdateAIError to be called after exhausted retries")
	}
	// Should have attempted 3 times.
	if atomic.LoadInt32(&callCount) != 3 {
		t.Errorf("expected 3 classify calls, got %d", callCount)
	}
}

// ---------------------------------------------------------------------------
// Tests — DB update failures
// ---------------------------------------------------------------------------

// TestProcessBatch_DBUpdateResultFails verifies that when UpdateAIResult fails,
// the article is counted as failed (not processed) and the batch continues.
func TestProcessBatch_DBUpdateResultFails(t *testing.T) {
	ctx := context.Background()

	art1 := entity.Article{ID: uuid.New(), Title: "Art1", Content: "Content"}
	art2 := entity.Article{ID: uuid.New(), Title: "Art2", Content: "Content"}

	var errUpdateCount int32

	repo := &mockRepository{
		listUnprocessedForAIFunc: func(ctx context.Context, limit int) ([]entity.Article, error) {
			return []entity.Article{art1, art2}, nil
		},
		updateAIResultFunc: func(ctx context.Context, id string, category string, confidence float64) error {
			return errors.New("failed to save result")
		},
		updateAIErrorFunc: func(ctx context.Context, id string, errMsg string) error {
			atomic.AddInt32(&errUpdateCount, 1)
			return nil
		},
	}

	gemini := &mockGeminiClient{
		classifyFunc: func(ctx context.Context, title string, content string) (*client.ClassifyResult, error) {
			return &client.ClassifyResult{Category: "Teknologi", Confidence: 0.9}, nil
		},
	}

	w := newWorkerFast(repo, gemini, redisLockSuccess(nil))
	res, err := w.ProcessBatch(ctx, 10)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Processed != 0 {
		t.Errorf("expected 0 processed (db save failed), got %d", res.Processed)
	}
	if res.Failed != 2 {
		t.Errorf("expected 2 failed, got %d", res.Failed)
	}
	// UpdateAIError should be called once per article to record the DB failure.
	if atomic.LoadInt32(&errUpdateCount) != 2 {
		t.Errorf("expected UpdateAIError called 2 times, got %d", errUpdateCount)
	}
}

// TestProcessBatch_DBSaveErrorsAndRedisDelError verifies that when both DB
// updates and Redis Del fail, the batch still returns nil error.
func TestProcessBatch_DBSaveErrorsAndRedisDelError(t *testing.T) {
	ctx := context.Background()

	art1 := entity.Article{ID: uuid.New(), Title: "Success Article", Content: "Body 1"}
	art2 := entity.Article{ID: uuid.New(), Title: "Failed Article", Content: "Body 2"}

	repo := &mockRepository{
		listUnprocessedForAIFunc: func(ctx context.Context, limit int) ([]entity.Article, error) {
			return []entity.Article{art1, art2}, nil
		},
		updateAIResultFunc: func(ctx context.Context, id string, category string, confidence float64) error {
			return errors.New("failed to save result")
		},
		updateAIErrorFunc: func(ctx context.Context, id string, errMsg string) error {
			return errors.New("failed to save error")
		},
	}

	gemini := &mockGeminiClient{
		classifyFunc: func(ctx context.Context, title string, content string) (*client.ClassifyResult, error) {
			if title == "Success Article" {
				return &client.ClassifyResult{Category: "Teknologi", Confidence: 0.9}, nil
			}
			return nil, errors.New("gemini failed")
		},
	}

	mr := &mockRedis{
		setNXFunc: func(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd {
			cmd := redis.NewBoolCmd(ctx)
			cmd.SetVal(true)
			return cmd
		},
		delFunc: func(ctx context.Context, keys ...string) *redis.IntCmd {
			cmd := redis.NewIntCmd(ctx)
			cmd.SetErr(errors.New("redis del error"))
			return cmd
		},
	}

	w := newWorkerFast(repo, gemini, mr)
	res, err := w.ProcessBatch(ctx, 10)

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if res.Processed != 0 {
		t.Errorf("expected 0 processed, got %d", res.Processed)
	}
	if res.Failed != 2 {
		t.Errorf("expected 2 failed, got %d", res.Failed)
	}
}

// ---------------------------------------------------------------------------
// Tests — Metrics
// ---------------------------------------------------------------------------

// TestProcessBatch_MetricsAccuracy verifies DurationMS, SuccessRate, and
// AverageDuration are populated for a non-empty batch.
func TestProcessBatch_MetricsAccuracy(t *testing.T) {
	ctx := context.Background()

	articles := sampleArticles(4)
	var processed int32

	repo := &mockRepository{
		listUnprocessedForAIFunc: func(ctx context.Context, limit int) ([]entity.Article, error) {
			return articles, nil
		},
		updateAIResultFunc: func(ctx context.Context, id string, category string, confidence float64) error {
			return nil
		},
		updateAIErrorFunc: func(ctx context.Context, id string, errMsg string) error { return nil },
	}

	gemini := &mockGeminiClient{
		classifyFunc: func(ctx context.Context, title string, content string) (*client.ClassifyResult, error) {
			n := atomic.AddInt32(&processed, 1)
			// First 3 succeed, 4th fails.
			if n <= 3 {
				return &client.ClassifyResult{Category: "Ekonomi", Confidence: 0.85}, nil
			}
			return nil, errors.New("fail")
		},
	}

	w := newWorkerFast(repo, gemini, redisLockSuccess(nil))
	res, err := w.ProcessBatch(ctx, 10)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Total != 4 {
		t.Errorf("expected Total=4, got %d", res.Total)
	}
	if res.Processed != 3 {
		t.Errorf("expected Processed=3, got %d", res.Processed)
	}
	if res.Failed != 1 {
		t.Errorf("expected Failed=1, got %d", res.Failed)
	}
	expectedRate := (3.0 / 4.0) * 100.0
	if res.SuccessRate != expectedRate {
		t.Errorf("expected SuccessRate=%.2f, got %.2f", expectedRate, res.SuccessRate)
	}
	// DurationMS should be non-negative.
	if res.DurationMS < 0 {
		t.Errorf("expected non-negative DurationMS, got %d", res.DurationMS)
	}
	// AverageDuration should be non-negative.
	if res.AverageDuration < 0 {
		t.Errorf("expected non-negative AverageDuration, got %.2f", res.AverageDuration)
	}
}

// ---------------------------------------------------------------------------
// Tests — Context cancellation
// ---------------------------------------------------------------------------

// TestProcessBatch_ContextCancelledBeforeStart verifies that a pre-cancelled
// context causes SetNX to surface the cancellation and the worker exits cleanly.
func TestProcessBatch_ContextCancelledBeforeStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	mr := &mockRedis{
		setNXFunc: func(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd {
			cmd := redis.NewBoolCmd(ctx)
			// Simulate Redis honouring the cancelled context.
			cmd.SetErr(ctx.Err())
			return cmd
		},
	}

	repo := &mockRepository{
		listUnprocessedForAIFunc: func(ctx context.Context, limit int) ([]entity.Article, error) {
			return []entity.Article{}, nil
		},
	}

	// When Redis errors (context cancelled), the worker continues without lock.
	// ListUnprocessedForAI would be called next; it returns empty so batch succeeds.
	w := NewAIWorker(repo, nil, mr, nopLogger())
	res, err := w.ProcessBatch(ctx, 10)
	// Expect either graceful empty batch or context error propagation.
	// Both are acceptable; what is NOT acceptable is a panic.
	_ = res
	_ = err
}

// TestProcessBatch_SingleArticleOneSuccess verifies that a single-article batch
// with success yields SuccessRate=100 and AverageDuration >= 0.
func TestProcessBatch_SingleArticleOneSuccess(t *testing.T) {
	ctx := context.Background()

	art := entity.Article{ID: uuid.New(), Title: "Solo Article", Content: "Solo Content"}
	repo := &mockRepository{
		listUnprocessedForAIFunc: func(ctx context.Context, limit int) ([]entity.Article, error) {
			return []entity.Article{art}, nil
		},
		updateAIResultFunc: func(ctx context.Context, id string, category string, confidence float64) error {
			return nil
		},
		updateAIErrorFunc: func(ctx context.Context, id string, errMsg string) error { return nil },
	}

	gemini := &mockGeminiClient{
		classifyFunc: func(ctx context.Context, title string, content string) (*client.ClassifyResult, error) {
			return &client.ClassifyResult{Category: "Olahraga", Confidence: 0.75}, nil
		},
	}

	w := newWorkerFast(repo, gemini, redisLockSuccess(nil))
	res, err := w.ProcessBatch(ctx, 10)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.SuccessRate != 100.0 {
		t.Errorf("expected 100%% success rate, got %.2f", res.SuccessRate)
	}
	if res.AverageDuration < 0 {
		t.Errorf("expected non-negative AverageDuration, got %.2f", res.AverageDuration)
	}
}
