package service

import (
	"context"
	"fmt"

	"koran-ai-backend/internal/ai/worker"
	"koran-ai-backend/internal/article/repository"
)

// ProcessResult holds the results of the AI run.
type ProcessResult struct {
	Processed int `json:"processed"`
	Failed    int `json:"failed"`
}

// AIStats contains high-level metrics for the AI categorization status.
type AIStats struct {
	Processed int64 `json:"processed"`
	Pending   int64 `json:"pending"`
	Failed    int64 `json:"failed"`
}

// Service is the interface defining AI management services.
type Service interface {
	Process(ctx context.Context, limit int) (*ProcessResult, error)
	GetStats(ctx context.Context) (*AIStats, error)
}

type aiService struct {
	repo   repository.Repository
	worker worker.Worker
}

// NewAIService constructs a new AI service instance.
func NewAIService(repo repository.Repository, worker worker.Worker) Service {
	return &aiService{
		repo:   repo,
		worker: worker,
	}
}

// Process invokes the batch worker process.
func (s *aiService) Process(ctx context.Context, limit int) (*ProcessResult, error) {
	res, err := s.worker.ProcessBatch(ctx, limit)
	if err != nil {
		return nil, err
	}
	return &ProcessResult{
		Processed: res.Processed,
		Failed:    res.Failed,
	}, nil
}

// GetStats compiles current statistics on AI processing from the repository.
func (s *aiService) GetStats(ctx context.Context) (*AIStats, error) {
	processed, err := s.repo.CountAIProcessed(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count processed articles: %w", err)
	}

	pending, err := s.repo.CountAIPending(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count pending articles: %w", err)
	}

	failed, err := s.repo.CountAIFailed(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count failed articles: %w", err)
	}

	return &AIStats{
		Processed: processed,
		Pending:   pending,
		Failed:    failed,
	}, nil
}
