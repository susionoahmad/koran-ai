package worker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"koran-ai-backend/internal/ai/client"
	"koran-ai-backend/internal/article/repository"
	appLogger "koran-ai-backend/internal/shared/logger"
)

// BatchResult holds metrics for the batch run.
type BatchResult struct {
	Processed int `json:"processed"`
	Failed    int `json:"failed"`
}

// Worker defines the contract for processing articles with AI.
type Worker interface {
	ProcessBatch(ctx context.Context, limit int) (*BatchResult, error)
}

type aiWorker struct {
	repo   repository.Repository
	client client.GeminiClient
	rdb    redis.Cmdable
	logger appLogger.Logger
}

// NewAIWorker creates a new AI worker instance.
func NewAIWorker(repo repository.Repository, client client.GeminiClient, rdb redis.Cmdable, logger appLogger.Logger) Worker {
	return &aiWorker{
		repo:   repo,
		client: client,
		rdb:    rdb,
		logger: logger,
	}
}

// ProcessBatch runs AI categorization on a batch of unprocessed articles.
func (w *aiWorker) ProcessBatch(ctx context.Context, limit int) (*BatchResult, error) {
	lockKey := "ai_worker_lock"
	lockTTL := 5 * time.Minute

	// Try to acquire distributed lock to prevent concurrent workers
	acquired, err := w.rdb.SetNX(ctx, lockKey, "running", lockTTL).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to acquire redis lock: %w", err)
	}
	if !acquired {
		return nil, errors.New("worker already running")
	}

	// Defer lock release
	defer func() {
		_, releaseErr := w.rdb.Del(ctx, lockKey).Result()
		if releaseErr != nil {
			w.logger.Error("failed to release redis lock", zap.Error(releaseErr))
		}
	}()

	// Query articles needing processing
	articles, err := w.repo.ListUnprocessedForAI(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve articles for AI processing: %w", err)
	}

	res := &BatchResult{}

	for _, art := range articles {
		startTime := time.Now()

		classifyRes, err := w.client.ClassifyArticle(ctx, art.Title, art.Content)
		duration := time.Since(startTime)

		if err != nil {
			// Increment retry count and log error details
			dbErr := w.repo.UpdateAIError(ctx, art.ID.String(), err.Error())
			if dbErr != nil {
				w.logger.Error("failed to update AI error in database",
					zap.String("article_id", art.ID.String()),
					zap.Error(dbErr),
				)
			}

			w.logger.Warn("AI categorization failed for article",
				zap.String("article_id", art.ID.String()),
				zap.String("category", ""),
				zap.Float64("confidence", 0.0),
				zap.Int64("duration_ms", duration.Milliseconds()),
				zap.String("status", "failed"),
				zap.Error(err),
			)
			res.Failed++
		} else {
			// Update article category and confidence
			dbErr := w.repo.UpdateAIResult(ctx, art.ID.String(), classifyRes.Category, classifyRes.Confidence)
			if dbErr != nil {
				w.logger.Error("failed to update AI result in database",
					zap.String("article_id", art.ID.String()),
					zap.Error(dbErr),
				)
				res.Failed++
			} else {
				w.logger.Info("AI categorization succeeded for article",
					zap.String("article_id", art.ID.String()),
					zap.String("category", classifyRes.Category),
					zap.Float64("confidence", classifyRes.Confidence),
					zap.Int64("duration_ms", duration.Milliseconds()),
					zap.String("status", "success"),
				)
				res.Processed++
			}
		}
	}

	return res, nil
}
