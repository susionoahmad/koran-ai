package worker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"koran-ai-backend/internal/ai/client"
	"koran-ai-backend/internal/article/repository"
	appLogger "koran-ai-backend/internal/shared/logger"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// BatchResult holds metrics for the batch run.
type BatchResult struct {
	Total             int     `json:"total"`
	Processed         int     `json:"processed"`
	Failed            int     `json:"failed"`
	DurationMS        int64   `json:"duration_ms"`
	TotalArticles     int     `json:"total_articles"`
	ProcessedArticles int     `json:"processed_articles"`
	FailedArticles    int     `json:"failed_articles"`
	SuccessRate       float64 `json:"success_rate"`
	AverageDuration   float64 `json:"average_duration"`
}

// Worker defines the contract for processing articles with AI.
type Worker interface {
	ProcessBatch(ctx context.Context, limit int) (*BatchResult, error)
}

type aiWorker struct {
	repo         repository.Repository
	client       client.GeminiClient
	rdb          redis.Cmdable
	logger       appLogger.Logger
	backoffScale time.Duration
}

// NewAIWorker creates a new AI worker instance.
func NewAIWorker(repo repository.Repository, client client.GeminiClient, rdb redis.Cmdable, logger appLogger.Logger) Worker {
	return &aiWorker{
		repo:         repo,
		client:       client,
		rdb:          rdb,
		logger:       logger,
		backoffScale: time.Second,
	}
}

// ProcessBatch runs AI categorization on a batch of unprocessed articles.
func (w *aiWorker) ProcessBatch(ctx context.Context, limit int) (*BatchResult, error) {
	startTime := time.Now()
	lockKey := "ai_worker_lock"
	lockTTL := 5 * time.Minute

	// Try to acquire distributed lock to prevent concurrent workers
	var lockAcquired bool
	if w.rdb == nil {
		w.logger.Warn("Redis client is nil, continuing without lock")
	} else {
		acquired, err := w.rdb.SetNX(ctx, lockKey, "running", lockTTL).Result()
		if err != nil {
			w.logger.Warn("Redis unavailable, continuing without lock", zap.Error(err))
		} else if !acquired {
			return nil, errors.New("worker already running")
		} else {
			lockAcquired = true
		}
	}

	// Defer lock release
	defer func() {
		if lockAcquired && w.rdb != nil {
			_, releaseErr := w.rdb.Del(ctx, lockKey).Result()
			if releaseErr != nil {
				w.logger.Error("failed to release redis lock", zap.Error(releaseErr))
			}
		}
	}()

	// Query articles needing processing
	articles, err := w.repo.ListUnprocessedForAI(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve articles for AI processing: %w", err)
	}

	// Diagnostic/debug logging
	w.logger.Debug("articles found for processing",
		zap.Int("count", len(articles)),
		zap.Int("limit", limit),
	)

	// Log Batch Start
	w.logger.Info("AI batch started",
		zap.Int("limit", limit),
		zap.Int("article_count", len(articles)),
	)

	res := &BatchResult{
		Total:         len(articles),
		TotalArticles: len(articles),
	}

	if len(articles) == 0 {
		batchDuration := time.Since(startTime)
		res.DurationMS = batchDuration.Milliseconds()
		w.logger.Info("AI batch completed",
			zap.Int("processed", 0),
			zap.Int("failed", 0),
			zap.Int("total", 0),
			zap.Int64("duration_ms", res.DurationMS),
		)
		return res, nil
	}

	var totalArticlesDuration time.Duration

	for i, art := range articles {
		currentIdx := i + 1
		w.logger.Info("processing article",
			zap.String("article_id", art.ID.String()),
			zap.String("title", art.Title),
			zap.Int("current", currentIdx),
			zap.Int("total", res.Total),
		)

		articleStart := time.Now()

		// Run in an inline function to manage per-article context timeout and cancellation
		classifyRes, err := func() (*client.ClassifyResult, error) {
			articleCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			var innerRes *client.ClassifyResult
			var innerErr error

			for attempt := 1; attempt <= 3; attempt++ {
				innerRes, innerErr = w.client.ClassifyArticle(articleCtx, art.Title, art.Content)
				if innerErr == nil {
					return innerRes, nil
				}

				if attempt < 3 {
					scale := w.backoffScale
					backoff := time.Duration(1<<(attempt-1)) * scale
					select {
					case <-articleCtx.Done():
						return nil, articleCtx.Err()
					case <-time.After(backoff):
					}
				}
			}
			return nil, innerErr
		}()

		articleDuration := time.Since(articleStart)
		totalArticlesDuration += articleDuration

		if err != nil {
			// Update AI error column
			dbErr := w.repo.UpdateAIError(ctx, art.ID.String(), err.Error())
			if dbErr != nil {
				w.logger.Error("failed to update AI error in database",
					zap.String("article_id", art.ID.String()),
					zap.Error(dbErr),
				)
			}

			w.logger.Error("AI categorization failed",
				zap.String("article_id", art.ID.String()),
				zap.String("error", err.Error()),
				zap.Int64("duration_ms", articleDuration.Milliseconds()),
			)
			res.Failed++
			res.FailedArticles++
		} else {
			// Update AI result in database
			dbErr := w.repo.UpdateAIResult(ctx, art.ID.String(), classifyRes.Category, classifyRes.Confidence)
			if dbErr != nil {
				w.logger.Error("failed to update AI result in database",
					zap.String("article_id", art.ID.String()),
					zap.Error(dbErr),
				)
				// Update AI error in database to record the DB failure
				_ = w.repo.UpdateAIError(ctx, art.ID.String(), fmt.Sprintf("database update failed: %v", dbErr))
				res.Failed++
				res.FailedArticles++
			} else {
				w.logger.Info("AI categorization succeeded",
					zap.String("category", classifyRes.Category),
					zap.Float64("confidence", classifyRes.Confidence),
					zap.Int64("duration_ms", articleDuration.Milliseconds()),
				)
				res.Processed++
				res.ProcessedArticles++
			}
		}
	}

	batchDuration := time.Since(startTime)
	res.DurationMS = batchDuration.Milliseconds()
	if res.Total > 0 {
		res.SuccessRate = (float64(res.Processed) / float64(res.Total)) * 100.0
		res.AverageDuration = float64(totalArticlesDuration.Milliseconds()) / float64(res.Total)
	}

	w.logger.Info("AI batch completed",
		zap.Int("processed", res.Processed),
		zap.Int("failed", res.Failed),
		zap.Int("total", res.Total),
		zap.Int64("duration_ms", res.DurationMS),
	)

	return res, nil
}

