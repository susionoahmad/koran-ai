package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"koran-ai-backend/internal/edition/entity"
)

var (
	ErrNotFound      = errors.New("edition not found")
	ErrAlreadyExists = errors.New("edition already exists for this date")
)

type dbRunner interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Begin(ctx context.Context) (txRunner, error)
}

type txRunner interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
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

func (p poolRunner) Begin(ctx context.Context) (txRunner, error) {
	tx, err := p.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return txRunnerWrapper{tx: tx}, nil
}

type txRunnerWrapper struct {
	tx pgx.Tx
}

func (w txRunnerWrapper) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	return w.tx.Exec(ctx, sql, arguments...)
}

func (w txRunnerWrapper) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return w.tx.Query(ctx, sql, args...)
}

func (w txRunnerWrapper) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return w.tx.QueryRow(ctx, sql, args...)
}

func (w txRunnerWrapper) Commit(ctx context.Context) error {
	return w.tx.Commit(ctx)
}

func (w txRunnerWrapper) Rollback(ctx context.Context) error {
	return w.tx.Rollback(ctx)
}

type Repository interface {
	CreateEdition(ctx context.Context, edition *entity.Edition, articles []entity.EditionArticle) error
	EditionExists(ctx context.Context, date time.Time) (bool, error)
	GetEditionByID(ctx context.Context, id string) (*entity.Edition, error)
	GetEditionByDate(ctx context.Context, date string) (*entity.Edition, error)
	ListEditions(ctx context.Context, page, limit int) ([]entity.Edition, int64, error)
	GetEditionDetails(ctx context.Context, editionID string) (*entity.EditionDetailResponse, error)
	GetStats(ctx context.Context) (int64, string, error)
	LoadSummariesForDate(ctx context.Context, date time.Time) ([]entity.EditionArticleSummary, error)
}

type postgresRepository struct {
	db dbRunner
}

func NewPostgresRepository(db *pgxpool.Pool) Repository {
	return &postgresRepository{db: poolRunner{db: db}}
}

// Internal constructor for unit testing with mocked dbRunner
func NewTestRepository(db dbRunner) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) CreateEdition(ctx context.Context, edition *entity.Edition, articles []entity.EditionArticle) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	return r.createEditionInternal(ctx, tx, edition, articles)
}

func (r *postgresRepository) createEditionInternal(ctx context.Context, tx txRunner, edition *entity.Edition, articles []entity.EditionArticle) error {
	var exists bool
	err := tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM editions WHERE edition_date = $1)", edition.EditionDate).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check duplicates: %w", err)
	}
	if exists {
		return ErrAlreadyExists
	}

	// Insert Edition
	if edition.ID == uuid.Nil {
		edition.ID = uuid.New()
	}
	queryEdition := `
		INSERT INTO editions (
			id, edition_date, title, headline_cluster_id, total_articles,
			status, generated_at, published_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err = tx.Exec(ctx, queryEdition,
		edition.ID,
		edition.EditionDate,
		edition.Title,
		edition.HeadlineClusterID,
		edition.TotalArticles,
		edition.Status,
		edition.GeneratedAt,
		edition.PublishedAt,
		edition.CreatedAt,
		edition.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert edition: %w", err)
	}

	// Insert Edition Articles
	queryArticle := `
		INSERT INTO edition_articles (
			id, edition_id, cluster_id, section, display_order, created_at
		) VALUES ($1, $2, $3, $4, $5, $6)
	`
	for _, art := range articles {
		artID := art.ID
		if artID == uuid.Nil {
			artID = uuid.New()
		}
		_, err = tx.Exec(ctx, queryArticle,
			artID,
			edition.ID,
			art.ClusterID,
			art.Section,
			art.DisplayOrder,
			art.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to insert edition article: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

func (r *postgresRepository) EditionExists(ctx context.Context, date time.Time) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM editions WHERE edition_date = $1)", date).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check edition existence: %w", err)
	}
	return exists, nil
}

func (r *postgresRepository) GetEditionByID(ctx context.Context, id string) (*entity.Edition, error) {
	var ed entity.Edition
	query := `
		SELECT id, edition_date, title, headline_cluster_id, total_articles,
		       status, generated_at, published_at, created_at, updated_at
		FROM editions WHERE id = $1
	`
	err := r.db.QueryRow(ctx, query, id).Scan(
		&ed.ID,
		&ed.EditionDate,
		&ed.Title,
		&ed.HeadlineClusterID,
		&ed.TotalArticles,
		&ed.Status,
		&ed.GeneratedAt,
		&ed.PublishedAt,
		&ed.CreatedAt,
		&ed.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get edition by id: %w", err)
	}
	return &ed, nil
}

func (r *postgresRepository) GetEditionByDate(ctx context.Context, date string) (*entity.Edition, error) {
	var ed entity.Edition
	query := `
		SELECT id, edition_date, title, headline_cluster_id, total_articles,
		       status, generated_at, published_at, created_at, updated_at
		FROM editions WHERE edition_date = $1::date
	`
	err := r.db.QueryRow(ctx, query, date).Scan(
		&ed.ID,
		&ed.EditionDate,
		&ed.Title,
		&ed.HeadlineClusterID,
		&ed.TotalArticles,
		&ed.Status,
		&ed.GeneratedAt,
		&ed.PublishedAt,
		&ed.CreatedAt,
		&ed.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get edition by date: %w", err)
	}
	return &ed, nil
}

func (r *postgresRepository) ListEditions(ctx context.Context, page, limit int) ([]entity.Edition, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit

	var total int64
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM editions").Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count editions: %w", err)
	}

	query := `
		SELECT id, edition_date, title, headline_cluster_id, total_articles,
		       status, generated_at, published_at, created_at, updated_at
		FROM editions
		ORDER BY edition_date DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query editions: %w", err)
	}
	defer rows.Close()

	var eds []entity.Edition
	for rows.Next() {
		var ed entity.Edition
		err := rows.Scan(
			&ed.ID,
			&ed.EditionDate,
			&ed.Title,
			&ed.HeadlineClusterID,
			&ed.TotalArticles,
			&ed.Status,
			&ed.GeneratedAt,
			&ed.PublishedAt,
			&ed.CreatedAt,
			&ed.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan edition: %w", err)
		}
		eds = append(eds, ed)
	}

	return eds, total, nil
}

func (r *postgresRepository) GetEditionDetails(ctx context.Context, editionID string) (*entity.EditionDetailResponse, error) {
	ed, err := r.GetEditionByID(ctx, editionID)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT ea.cluster_id, ea.section,
		       s.headline, s.summary_short, COALESCE(s.summary_medium, ''), COALESCE(s.summary_long, ''), COALESCE(s.key_points, '[]'::jsonb),
		       COALESCE(s.ai_model, ''), s.ai_confidence,
		       c.category, c.article_count, c.confidence, s.generated_at
		FROM edition_articles ea
		JOIN summaries s ON ea.cluster_id = s.cluster_id
		JOIN news_clusters c ON ea.cluster_id = c.id
		WHERE ea.edition_id = $1
		ORDER BY ea.display_order ASC
	`
	rows, err := r.db.Query(ctx, query, editionID)
	if err != nil {
		return nil, fmt.Errorf("failed to query edition articles: %w", err)
	}
	defer rows.Close()

	sectionsMap := make(map[string][]entity.EditionArticleSummary)

	var headline *entity.EditionArticleSummary

	for rows.Next() {
		var art entity.EditionArticleSummary
		var section string
		var keyPointsBytes []byte

		err := rows.Scan(
			&art.ClusterID,
			&section,
			&art.Title,
			&art.SummaryShort,
			&art.SummaryMedium,
			&art.SummaryLong,
			&keyPointsBytes,
			&art.AIModel,
			&art.AIConfidence,
			&art.Category,
			&art.ArticleCount,
			&art.Confidence,
			&art.GeneratedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan edition article details: %w", err)
		}

		if len(keyPointsBytes) > 0 {
			_ = json.Unmarshal(keyPointsBytes, &art.KeyPoints)
		}

		if section == "Headline News" {
			headline = &art
		} else {
			sectionsMap[section] = append(sectionsMap[section], art)
		}
	}

	// Format sections as array of SectionDetail
	sections := make([]entity.SectionDetail, 0, len(sectionsMap))
	for secName, arts := range sectionsMap {
		sections = append(sections, entity.SectionDetail{
			Name:     secName,
			Articles: arts,
		})
	}

	return &entity.EditionDetailResponse{
		ID:                ed.ID,
		EditionDate:       ed.EditionDate.Format("2006-01-02"),
		Title:             ed.Title,
		Status:            ed.Status,
		HeadlineClusterID: ed.HeadlineClusterID,
		Headline:          headline,
		Sections:          sections,
	}, nil
}

func (r *postgresRepository) GetStats(ctx context.Context) (int64, string, error) {
	var total int64
	var latestDate time.Time

	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM editions").Scan(&total)
	if err != nil {
		return 0, "", fmt.Errorf("failed to count editions: %w", err)
	}

	if total == 0 {
		return 0, "", nil
	}

	err = r.db.QueryRow(ctx, "SELECT MAX(edition_date) FROM editions").Scan(&latestDate)
	if err != nil {
		return 0, "", fmt.Errorf("failed to get latest edition date: %w", err)
	}

	return total, latestDate.Format("2006-01-02"), nil
}

func (r *postgresRepository) LoadSummariesForDate(ctx context.Context, date time.Time) ([]entity.EditionArticleSummary, error) {
	query := `
		SELECT s.cluster_id, s.headline, s.summary_short, COALESCE(s.summary_medium, ''), COALESCE(s.summary_long, ''), COALESCE(s.key_points, '[]'::jsonb),
		       COALESCE(s.ai_model, ''), s.ai_confidence, c.category, c.article_count, c.confidence, s.generated_at
		FROM summaries s
		JOIN news_clusters c ON s.cluster_id = c.id
		WHERE s.generated_at::date = $1::date
	`
	rows, err := r.db.Query(ctx, query, date)
	if err != nil {
		return nil, fmt.Errorf("failed to query summaries: %w", err)
	}
	defer rows.Close()

	var summaries []entity.EditionArticleSummary
	for rows.Next() {
		var s entity.EditionArticleSummary
		var keyPointsBytes []byte
		err := rows.Scan(
			&s.ClusterID,
			&s.Title,
			&s.SummaryShort,
			&s.SummaryMedium,
			&s.SummaryLong,
			&keyPointsBytes,
			&s.AIModel,
			&s.AIConfidence,
			&s.Category,
			&s.ArticleCount,
			&s.Confidence,
			&s.GeneratedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan summary for date: %w", err)
		}
		if len(keyPointsBytes) > 0 {
			_ = json.Unmarshal(keyPointsBytes, &s.KeyPoints)
		}
		summaries = append(summaries, s)
	}

	return summaries, nil
}
