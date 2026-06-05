package repository

import (
	"context"
	"koran-ai-backend/internal/article/entity"
)

// Repository defines data-access operations for articles.
type Repository interface {
	// Create inserts a new article. Returns ErrDuplicateURL or ErrDuplicateHash on conflict.
	Create(ctx context.Context, a *entity.Article) error

	// ExistsByURL returns true if an article with the given URL already exists.
	ExistsByURL(ctx context.Context, url string) (bool, error)

	// ExistsByHash returns true if an article with the given hash_content already exists.
	ExistsByHash(ctx context.Context, hash string) (bool, error)

	// ListUnprocessed returns articles that have not yet been AI-processed.
	ListUnprocessed(ctx context.Context, limit int) ([]entity.Article, error)

	// GetByID returns a single article by its UUID string.
	GetByID(ctx context.Context, id string) (*entity.Article, error)

	// ListUnprocessedForAI returns articles that are not AI processed, sorted by published_at DESC.
	ListUnprocessedForAI(ctx context.Context, limit int) ([]entity.Article, error)

	// ListUnclustered returns articles that are AI processed but not clustered.
	ListUnclustered(ctx context.Context, limit int) ([]entity.Article, error)

	// CountTotal returns the total count of articles in the database.
	CountTotal(ctx context.Context) (int64, error)

	// CountToday returns the count of articles scraped today (UTC).
	CountToday(ctx context.Context) (int64, error)

	// UpdateAIResult updates article with AI categorization result (success path).
	UpdateAIResult(ctx context.Context, id string, category string, confidence float64) error

	// UpdateAIError updates article with AI failure info and increments retry count.
	UpdateAIError(ctx context.Context, id string, errMsg string) error

	// CountAIPending returns count of articles not yet AI processed with retry < 3.
	CountAIPending(ctx context.Context) (int64, error)

	// CountAIFailed returns count of articles that hit max retry (ai_retry_count >= 3).
	CountAIFailed(ctx context.Context) (int64, error)

	// CountAIProcessed returns count of articles that have been AI processed.
	CountAIProcessed(ctx context.Context) (int64, error)
}
