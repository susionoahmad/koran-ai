package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"koran-ai-backend/internal/article/entity"
)

var (
	ErrNotFound      = errors.New("article not found")
	ErrDuplicateURL  = errors.New("article url already exists")
	ErrDuplicateHash = errors.New("article hash already exists")
)

type postgresRepository struct {
	db *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL-backed article repository.
func NewPostgresRepository(db *pgxpool.Pool) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) Create(ctx context.Context, a *entity.Article) error {
	query := `
		INSERT INTO articles
			(id, source_id, title, slug, url, author, content, published_at,
			 scraped_at, hash_content, image_url, processed,
			 ai_processed, ai_category, ai_confidence, ai_processed_at,
			 clustered, cluster_id, created_at, updated_at, ai_error, ai_retry_count)
		VALUES
			($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22)
	`
	_, err := r.db.Exec(ctx, query,
		a.ID, a.SourceID, a.Title, a.Slug, a.URL, a.Author, a.Content,
		a.PublishedAt, a.ScrapedAt, a.HashContent, a.ImageURL, a.Processed,
		a.AIProcessed, a.AICategory, a.AIConfidence, a.AIProcessedAt,
		a.Clustered, a.ClusterID, a.CreatedAt, a.UpdatedAt, a.AIError, a.AIRetryCount,
	)
	if err != nil {
		errMsg := err.Error()
		if contains(errMsg, "uniq_articles_url") {
			return ErrDuplicateURL
		}
		if contains(errMsg, "uniq_articles_hash") {
			return ErrDuplicateHash
		}
		return fmt.Errorf("failed to insert article: %w", err)
	}
	return nil
}

func (r *postgresRepository) ExistsByURL(ctx context.Context, url string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM articles WHERE url = $1)`
	var exists bool
	err := r.db.QueryRow(ctx, query, url).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check url existence: %w", err)
	}
	return exists, nil
}

func (r *postgresRepository) ExistsByHash(ctx context.Context, hash string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM articles WHERE hash_content = $1)`
	var exists bool
	err := r.db.QueryRow(ctx, query, hash).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check hash existence: %w", err)
	}
	return exists, nil
}

func (r *postgresRepository) ListUnprocessed(ctx context.Context, limit int) ([]entity.Article, error) {
	query := `
		SELECT id, source_id, title, slug, url, author, content, published_at,
		       scraped_at, hash_content, image_url, processed,
		       ai_processed, ai_category, ai_confidence, ai_processed_at,
		       clustered, cluster_id, created_at, updated_at, ai_error, ai_retry_count
		FROM articles
		WHERE processed = FALSE
		ORDER BY scraped_at ASC
		LIMIT $1
	`
	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list unprocessed articles: %w", err)
	}
	defer rows.Close()

	var articles []entity.Article
	for rows.Next() {
		var a entity.Article
		if err := rows.Scan(
			&a.ID, &a.SourceID, &a.Title, &a.Slug, &a.URL, &a.Author, &a.Content,
			&a.PublishedAt, &a.ScrapedAt, &a.HashContent, &a.ImageURL, &a.Processed,
			&a.AIProcessed, &a.AICategory, &a.AIConfidence, &a.AIProcessedAt,
			&a.Clustered, &a.ClusterID, &a.CreatedAt, &a.UpdatedAt, &a.AIError, &a.AIRetryCount,
		); err != nil {
			return nil, fmt.Errorf("failed to scan article: %w", err)
		}
		articles = append(articles, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return articles, nil
}

func (r *postgresRepository) GetByID(ctx context.Context, id string) (*entity.Article, error) {
	query := `
		SELECT id, source_id, title, slug, url, author, content, published_at,
		       scraped_at, hash_content, image_url, processed,
		       ai_processed, ai_category, ai_confidence, ai_processed_at,
		       clustered, cluster_id, created_at, updated_at, ai_error, ai_retry_count
		FROM articles
		WHERE id = $1
	`
	var a entity.Article
	err := r.db.QueryRow(ctx, query, id).Scan(
		&a.ID, &a.SourceID, &a.Title, &a.Slug, &a.URL, &a.Author, &a.Content,
		&a.PublishedAt, &a.ScrapedAt, &a.HashContent, &a.ImageURL, &a.Processed,
		&a.AIProcessed, &a.AICategory, &a.AIConfidence, &a.AIProcessedAt,
		&a.Clustered, &a.ClusterID, &a.CreatedAt, &a.UpdatedAt, &a.AIError, &a.AIRetryCount,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get article by id: %w", err)
	}
	return &a, nil
}

func (r *postgresRepository) ListUnprocessedForAI(ctx context.Context, limit int) ([]entity.Article, error) {
	query := `
		SELECT id, source_id, title, slug, url, author, content, published_at,
		       scraped_at, hash_content, image_url, processed,
		       ai_processed, ai_category, ai_confidence, ai_processed_at,
		       clustered, cluster_id, created_at, updated_at, ai_error, ai_retry_count
		FROM articles
		WHERE ai_processed = FALSE AND ai_retry_count < 3
		ORDER BY published_at DESC
		LIMIT $1
	`
	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list unprocessed for AI: %w", err)
	}
	defer rows.Close()

	var articles []entity.Article
	for rows.Next() {
		var a entity.Article
		if err := rows.Scan(
			&a.ID, &a.SourceID, &a.Title, &a.Slug, &a.URL, &a.Author, &a.Content,
			&a.PublishedAt, &a.ScrapedAt, &a.HashContent, &a.ImageURL, &a.Processed,
			&a.AIProcessed, &a.AICategory, &a.AIConfidence, &a.AIProcessedAt,
			&a.Clustered, &a.ClusterID, &a.CreatedAt, &a.UpdatedAt, &a.AIError, &a.AIRetryCount,
		); err != nil {
			return nil, fmt.Errorf("failed to scan article: %w", err)
		}
		articles = append(articles, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return articles, nil
}

func (r *postgresRepository) ListUnclustered(ctx context.Context, limit int) ([]entity.Article, error) {
	query := `
		SELECT id, source_id, title, slug, url, author, content, published_at,
		       scraped_at, hash_content, image_url, processed,
		       ai_processed, ai_category, ai_confidence, ai_processed_at,
		       clustered, cluster_id, created_at, updated_at, ai_error, ai_retry_count
		FROM articles
		WHERE clustered = FALSE AND ai_processed = TRUE
		LIMIT $1
	`
	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list unclustered: %w", err)
	}
	defer rows.Close()

	var articles []entity.Article
	for rows.Next() {
		var a entity.Article
		if err := rows.Scan(
			&a.ID, &a.SourceID, &a.Title, &a.Slug, &a.URL, &a.Author, &a.Content,
			&a.PublishedAt, &a.ScrapedAt, &a.HashContent, &a.ImageURL, &a.Processed,
			&a.AIProcessed, &a.AICategory, &a.AIConfidence, &a.AIProcessedAt,
			&a.Clustered, &a.ClusterID, &a.CreatedAt, &a.UpdatedAt, &a.AIError, &a.AIRetryCount,
		); err != nil {
			return nil, fmt.Errorf("failed to scan article: %w", err)
		}
		articles = append(articles, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return articles, nil
}

// contains is a simple substring check to avoid importing strings package inline.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func (r *postgresRepository) CountTotal(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM articles`
	var count int64
	err := r.db.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count total articles: %w", err)
	}
	return count, nil
}

func (r *postgresRepository) CountToday(ctx context.Context) (int64, error) {
	// Query articles where scraped_at is today (since last midnight UTC)
	query := `SELECT COUNT(*) FROM articles WHERE scraped_at >= date_trunc('day', NOW())`
	var count int64
	err := r.db.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count articles today: %w", err)
	}
	return count, nil
}

func (r *postgresRepository) UpdateAIResult(ctx context.Context, id string, category string, confidence float64) error {
	query := `
		UPDATE articles
		SET ai_processed = TRUE,
			ai_category = $1,
			ai_confidence = $2,
			ai_processed_at = NOW(),
			ai_error = NULL,
			updated_at = NOW()
		WHERE id = $3
	`
	res, err := r.db.Exec(ctx, query, category, confidence, id)
	if err != nil {
		return fmt.Errorf("failed to update AI result: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *postgresRepository) UpdateAIError(ctx context.Context, id string, errMsg string) error {
	query := `
		UPDATE articles
		SET ai_retry_count = ai_retry_count + 1,
			ai_error = $1,
			updated_at = NOW()
		WHERE id = $2
	`
	res, err := r.db.Exec(ctx, query, errMsg, id)
	if err != nil {
		return fmt.Errorf("failed to update AI error: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *postgresRepository) CountAIPending(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM articles WHERE ai_processed = FALSE AND ai_retry_count < 3`
	var count int64
	err := r.db.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count pending AI articles: %w", err)
	}
	return count, nil
}

func (r *postgresRepository) CountAIFailed(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM articles WHERE ai_retry_count >= 3`
	var count int64
	err := r.db.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count failed AI articles: %w", err)
	}
	return count, nil
}

func (r *postgresRepository) CountAIProcessed(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM articles WHERE ai_processed = TRUE`
	var count int64
	err := r.db.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count AI processed articles: %w", err)
	}
	return count, nil
}



