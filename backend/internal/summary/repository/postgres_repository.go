package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	articleEntity "koran-ai-backend/internal/article/entity"
	"koran-ai-backend/internal/summary/entity"
)

type dbRunner interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type poolRunner struct {
	db *pgxpool.Pool
}

func (p poolRunner) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	return p.db.Exec(ctx, sql, arguments...)
}

func (p poolRunner) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return p.db.Query(ctx, sql, args...)
}

func (p poolRunner) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return p.db.QueryRow(ctx, sql, args...)
}

type postgresRepository struct {
	db dbRunner
}

// NewPostgresRepository creates a PostgreSQL-backed summary repository.
func NewPostgresRepository(db *pgxpool.Pool) Repository {
	return &postgresRepository{db: poolRunner{db: db}}
}

func (r *postgresRepository) CreateSummary(ctx context.Context, summary *entity.Summary) error {
	if summary.ID == uuid.Nil {
		summary.ID = uuid.New()
	}
	keyPoints, err := json.Marshal(summary.KeyPoints)
	if err != nil {
		return fmt.Errorf("failed to marshal key points: %w", err)
	}

	query := `
		INSERT INTO summaries (
			id, cluster_id, headline, summary_short, summary_medium, summary_long,
			key_points, ai_model, ai_confidence, generated_at, created_at, updated_at
		)
		VALUES (
			$1, $2, $3, $4, NULLIF($5, ''), NULLIF($6, ''), $7::jsonb,
			NULLIF($8, ''), $9,
			COALESCE(NULLIF($10::timestamptz, '0001-01-01'::timestamptz), NOW()),
			COALESCE(NULLIF($11::timestamptz, '0001-01-01'::timestamptz), NOW()),
			COALESCE(NULLIF($12::timestamptz, '0001-01-01'::timestamptz), NOW())
		)
	`
	_, err = r.db.Exec(ctx, query,
		summary.ID,
		summary.ClusterID,
		summary.Headline,
		summary.SummaryShort,
		summary.SummaryMedium,
		summary.SummaryLong,
		string(keyPoints),
		summary.AIModel,
		summary.AIConfidence,
		summary.GeneratedAt,
		summary.CreatedAt,
		summary.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create summary: %w", err)
	}
	return nil
}

func (r *postgresRepository) GetSummaryByID(ctx context.Context, id string) (*entity.Summary, error) {
	summary, err := scanSummary(r.db.QueryRow(ctx, summarySelectSQL()+` WHERE id = $1`, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get summary by id: %w", err)
	}
	return summary, nil
}

func (r *postgresRepository) GetSummaryByClusterID(ctx context.Context, clusterID string) (*entity.Summary, error) {
	summary, err := scanSummary(r.db.QueryRow(ctx, summarySelectSQL()+` WHERE cluster_id = $1`, clusterID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get summary by cluster id: %w", err)
	}
	return summary, nil
}

func (r *postgresRepository) ListSummaries(ctx context.Context, page int, limit int) ([]entity.Summary, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit

	total, err := r.CountSummaries(ctx)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx, summarySelectSQL()+`
		ORDER BY generated_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list summaries: %w", err)
	}
	defer rows.Close()

	summaries, err := scanSummaries(rows)
	if err != nil {
		return nil, 0, err
	}
	return summaries, total, nil
}

func (r *postgresRepository) CountSummaries(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM summaries`).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count summaries: %w", err)
	}
	return count, nil
}

func (r *postgresRepository) ListPendingClusters(ctx context.Context, limit int) ([]string, error) {
	if limit < 1 {
		limit = 50
	}
	rows, err := r.db.Query(ctx, `
		SELECT nc.id
		FROM news_clusters nc
		LEFT JOIN summaries s ON s.cluster_id = nc.id
		WHERE s.id IS NULL
		ORDER BY nc.created_at ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list pending clusters: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan pending cluster: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("pending cluster rows error: %w", err)
	}
	return ids, nil
}

func (r *postgresRepository) ListClusterArticles(ctx context.Context, clusterID string) ([]articleEntity.Article, error) {
	rows, err := r.db.Query(ctx, `
		SELECT a.id, a.source_id, a.title, a.slug, a.url, a.author, a.content,
		       a.published_at, a.scraped_at, a.hash_content, a.image_url, a.processed,
		       a.ai_processed, a.ai_category, a.ai_confidence, a.ai_processed_at,
		       a.clustered, a.cluster_id, a.created_at, a.updated_at, a.ai_error, a.ai_retry_count
		FROM cluster_articles ca
		JOIN articles a ON a.id = ca.article_id
		WHERE ca.cluster_id = $1
		ORDER BY a.published_at DESC NULLS LAST, a.scraped_at DESC
	`, clusterID)
	if err != nil {
		return nil, fmt.Errorf("failed to list cluster articles: %w", err)
	}
	defer rows.Close()

	var articles []articleEntity.Article
	for rows.Next() {
		article, err := scanArticle(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan cluster article: %w", err)
		}
		articles = append(articles, *article)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("cluster article rows error: %w", err)
	}
	return articles, nil
}

func (r *postgresRepository) Stats(ctx context.Context) (*Stats, error) {
	var stats Stats
	err := r.db.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM summaries),
			(SELECT COUNT(*)
			 FROM news_clusters nc
			 LEFT JOIN summaries s ON s.cluster_id = nc.id
			 WHERE s.id IS NULL)
	`).Scan(&stats.TotalSummaries, &stats.PendingClusters)
	if err != nil {
		return nil, fmt.Errorf("failed to get summary stats: %w", err)
	}
	return &stats, nil
}

func summarySelectSQL() string {
	return `
		SELECT id, cluster_id, headline, summary_short, COALESCE(summary_medium, ''),
		       COALESCE(summary_long, ''), COALESCE(key_points, '[]'::jsonb),
		       COALESCE(ai_model, ''), ai_confidence, generated_at, created_at, updated_at
		FROM summaries
	`
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanSummary(scanner rowScanner) (*entity.Summary, error) {
	var summary entity.Summary
	var keyPointsBytes []byte
	err := scanner.Scan(
		&summary.ID,
		&summary.ClusterID,
		&summary.Headline,
		&summary.SummaryShort,
		&summary.SummaryMedium,
		&summary.SummaryLong,
		&keyPointsBytes,
		&summary.AIModel,
		&summary.AIConfidence,
		&summary.GeneratedAt,
		&summary.CreatedAt,
		&summary.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if len(keyPointsBytes) > 0 {
		if err := json.Unmarshal(keyPointsBytes, &summary.KeyPoints); err != nil {
			return nil, fmt.Errorf("failed to unmarshal key points: %w", err)
		}
	}
	return &summary, nil
}

func scanSummaries(rows pgx.Rows) ([]entity.Summary, error) {
	var summaries []entity.Summary
	for rows.Next() {
		summary, err := scanSummary(rows)
		if err != nil {
			return nil, fmt.Errorf("failed to scan summary: %w", err)
		}
		summaries = append(summaries, *summary)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("summary rows error: %w", err)
	}
	return summaries, nil
}

func scanArticle(scanner rowScanner) (*articleEntity.Article, error) {
	var a articleEntity.Article
	var author sql.NullString
	var imageURL sql.NullString
	var aiCategory sql.NullString
	var aiError sql.NullString

	err := scanner.Scan(
		&a.ID, &a.SourceID, &a.Title, &a.Slug, &a.URL, &author, &a.Content,
		&a.PublishedAt, &a.ScrapedAt, &a.HashContent, &imageURL, &a.Processed,
		&a.AIProcessed, &aiCategory, &a.AIConfidence, &a.AIProcessedAt,
		&a.Clustered, &a.ClusterID, &a.CreatedAt, &a.UpdatedAt, &aiError, &a.AIRetryCount,
	)
	if err != nil {
		return nil, err
	}
	a.Author = nullStringPtr(author)
	a.ImageURL = nullStringPtr(imageURL)
	a.AICategory = nullStringPtr(aiCategory)
	a.AIError = nullStringPtr(aiError)
	return &a, nil
}

func nullStringPtr(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}
