package service

import (
	"context"
	"errors"
	"testing"

	"koran-ai-backend/internal/ai/worker"
	"koran-ai-backend/internal/article/repository"
)

type mockRepositoryForService struct {
	repository.Repository
	countAIProcessedFunc func(ctx context.Context) (int64, error)
	countAIPendingFunc   func(ctx context.Context) (int64, error)
	countAIFailedFunc    func(ctx context.Context) (int64, error)
}

func (m *mockRepositoryForService) CountAIProcessed(ctx context.Context) (int64, error) {
	return m.countAIProcessedFunc(ctx)
}
func (m *mockRepositoryForService) CountAIPending(ctx context.Context) (int64, error) {
	return m.countAIPendingFunc(ctx)
}
func (m *mockRepositoryForService) CountAIFailed(ctx context.Context) (int64, error) {
	return m.countAIFailedFunc(ctx)
}

type mockWorker struct {
	processFunc func(ctx context.Context, limit int) (*worker.BatchResult, error)
}

func (m *mockWorker) ProcessBatch(ctx context.Context, limit int) (*worker.BatchResult, error) {
	return m.processFunc(ctx, limit)
}

func TestProcess_Success(t *testing.T) {
	ctx := context.Background()

	w := &mockWorker{
		processFunc: func(ctx context.Context, limit int) (*worker.BatchResult, error) {
			return &worker.BatchResult{Processed: 5, Failed: 2}, nil
		},
	}

	s := NewAIService(nil, w)
	res, err := s.Process(ctx, 10)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if res.Processed != 5 || res.Failed != 2 {
		t.Errorf("expected Processed=5, Failed=2, got Processed=%d, Failed=%d", res.Processed, res.Failed)
	}
}

func TestProcess_Error(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("worker failed")

	w := &mockWorker{
		processFunc: func(ctx context.Context, limit int) (*worker.BatchResult, error) {
			return nil, expectedErr
		},
	}

	s := NewAIService(nil, w)
	_, err := s.Process(ctx, 10)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestGetStats_Success(t *testing.T) {
	ctx := context.Background()

	repo := &mockRepositoryForService{
		countAIProcessedFunc: func(ctx context.Context) (int64, error) {
			return 100, nil
		},
		countAIPendingFunc: func(ctx context.Context) (int64, error) {
			return 20, nil
		},
		countAIFailedFunc: func(ctx context.Context) (int64, error) {
			return 5, nil
		},
	}

	s := NewAIService(repo, nil)
	stats, err := s.GetStats(ctx)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if stats.Processed != 100 || stats.Pending != 20 || stats.Failed != 5 {
		t.Errorf("expected stats {Processed:100, Pending:20, Failed:5}, got %+v", stats)
	}
}

func TestGetStats_Error(t *testing.T) {
	ctx := context.Background()
	repoErr := errors.New("db down")

	repo := &mockRepositoryForService{
		countAIProcessedFunc: func(ctx context.Context) (int64, error) {
			return 0, repoErr
		},
	}

	s := NewAIService(repo, nil)
	_, err := s.GetStats(ctx)
	if err == nil || !errors.Is(err, repoErr) {
		t.Fatalf("expected db error, got %v", err)
	}
}
