package worker

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"koran-ai-backend/internal/ai/client"
	"koran-ai-backend/internal/article/entity"
	"koran-ai-backend/internal/article/repository"
)

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

func TestProcessBatch_LockAcquisitionFailed(t *testing.T) {
	ctx := context.Background()
	logger := &mockLogger{zapLog: zap.NewNop()}

	mr := &mockRedis{
		setNXFunc: func(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd {
			cmd := redis.NewBoolCmd(ctx)
			cmd.SetVal(false) // Lock already held
			return cmd
		},
	}

	w := NewAIWorker(nil, nil, mr, logger)
	_, err := w.ProcessBatch(ctx, 10)
	if err == nil || err.Error() != "worker already running" {
		t.Fatalf("expected error 'worker already running', got %v", err)
	}
}

func TestProcessBatch_RedisError(t *testing.T) {
	ctx := context.Background()
	logger := &mockLogger{zapLog: zap.NewNop()}

	mr := &mockRedis{
		setNXFunc: func(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd {
			cmd := redis.NewBoolCmd(ctx)
			cmd.SetErr(errors.New("redis down"))
			return cmd
		},
	}

	w := NewAIWorker(nil, nil, mr, logger)
	_, err := w.ProcessBatch(ctx, 10)
	if err == nil || !strings.Contains(err.Error(), "redis down") {
		t.Fatalf("expected redis error, got %v", err)
	}
}

func TestProcessBatch_SuccessAndFailureMix(t *testing.T) {
	ctx := context.Background()
	logger := &mockLogger{zapLog: zap.NewNop()}

	art1 := entity.Article{ID: uuid.New(), Title: "Success Article", Content: "Body text 1"}
	art2 := entity.Article{ID: uuid.New(), Title: "Failed Article", Content: "Body text 2"}

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
	mr := &mockRedis{
		setNXFunc: func(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd {
			cmd := redis.NewBoolCmd(ctx)
			cmd.SetVal(true)
			return cmd
		},
		delFunc: func(ctx context.Context, keys ...string) *redis.IntCmd {
			lockReleased = true
			cmd := redis.NewIntCmd(ctx)
			cmd.SetVal(1)
			return cmd
		},
	}

	w := NewAIWorker(repo, gemini, mr, logger)
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

func TestProcessBatch_DatabaseRetrievalError(t *testing.T) {
	ctx := context.Background()
	logger := &mockLogger{zapLog: zap.NewNop()}

	repo := &mockRepository{
		listUnprocessedForAIFunc: func(ctx context.Context, limit int) ([]entity.Article, error) {
			return nil, errors.New("db error on select")
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
			cmd.SetVal(1)
			return cmd
		},
	}

	w := NewAIWorker(repo, nil, mr, logger)
	_, err := w.ProcessBatch(ctx, 10)
	if err == nil || !strings.Contains(err.Error(), "db error on select") {
		t.Fatalf("expected select database error, got %v", err)
	}
}

func TestProcessBatch_DBSaveErrorsAndRedisDelError(t *testing.T) {
	ctx := context.Background()
	logger := &mockLogger{zapLog: zap.NewNop()}

	art1 := entity.Article{ID: uuid.New(), Title: "Success Article", Content: "Body text 1"}
	art2 := entity.Article{ID: uuid.New(), Title: "Failed Article", Content: "Body text 2"}

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

	w := NewAIWorker(repo, gemini, mr, logger)
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

