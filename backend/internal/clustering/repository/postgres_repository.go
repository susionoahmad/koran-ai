package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"koran-ai-backend/internal/clustering/entity"
)

type dbRunner interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Begin(ctx context.Context) (txRunner, error)
}

type txRunner interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
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
	return p.db.Begin(ctx)
}

type postgresRepository struct {
	db dbRunner
}

// NewPostgresRepository creates a PostgreSQL-backed clustering repository.
func NewPostgresRepository(db *pgxpool.Pool) Repository {
	return &postgresRepository{db: poolRunner{db: db}}
}

func (r *postgresRepository) CreateCluster(ctx context.Context, cluster *entity.Cluster) error {
	query := `
		INSERT INTO news_clusters (id, title, category, article_count, confidence, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, COALESCE(NULLIF($6::timestamptz, '0001-01-01'::timestamptz), NOW()), COALESCE(NULLIF($7::timestamptz, '0001-01-01'::timestamptz), NOW()))
	`
	_, err := r.db.Exec(ctx, query,
		cluster.ID,
		cluster.Title,
		nullableString(cluster.Category),
		cluster.ArticleCount,
		cluster.Confidence,
		cluster.CreatedAt,
		cluster.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create cluster: %w", err)
	}
	return nil
}

func (r *postgresRepository) AddArticleToCluster(ctx context.Context, clusterID string, articleID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin add article transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	_, err = tx.Exec(ctx, `
		INSERT INTO cluster_articles (cluster_id, article_id)
		VALUES ($1, $2)
		ON CONFLICT (cluster_id, article_id) DO NOTHING
	`, clusterID, articleID)
	if err != nil {
		return fmt.Errorf("failed to add article to cluster: %w", err)
	}

	tag, err := tx.Exec(ctx, `
		UPDATE articles
		SET clustered = TRUE,
			cluster_id = $1,
			updated_at = NOW()
		WHERE id = $2
	`, clusterID, articleID)
	if err != nil {
		return fmt.Errorf("failed to mark article clustered: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("article %s not found", articleID)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit add article transaction: %w", err)
	}
	return nil
}

func (r *postgresRepository) ListClusters(ctx context.Context, page int, limit int) ([]entity.Cluster, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit

	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM news_clusters`).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count clusters: %w", err)
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, title, COALESCE(category, ''), article_count, confidence, created_at, updated_at
		FROM news_clusters
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list clusters: %w", err)
	}
	defer rows.Close()

	clusters, err := scanClusters(rows)
	if err != nil {
		return nil, 0, err
	}
	return clusters, total, nil
}

func (r *postgresRepository) GetClusterByID(ctx context.Context, id string) (*entity.Cluster, error) {
	var cluster entity.Cluster
	err := r.db.QueryRow(ctx, `
		SELECT id, title, COALESCE(category, ''), article_count, confidence, created_at, updated_at
		FROM news_clusters
		WHERE id = $1
	`, id).Scan(
		&cluster.ID,
		&cluster.Title,
		&cluster.Category,
		&cluster.ArticleCount,
		&cluster.Confidence,
		&cluster.CreatedAt,
		&cluster.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get cluster by id: %w", err)
	}
	return &cluster, nil
}

func (r *postgresRepository) UpdateClusterStats(ctx context.Context, clusterID string, articleCount int, confidence float64) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE news_clusters
		SET article_count = $2,
			confidence = $3,
			updated_at = NOW()
		WHERE id = $1
	`, clusterID, articleCount, confidence)
	if err != nil {
		return fmt.Errorf("failed to update cluster stats: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *postgresRepository) CountClusteredArticles(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM articles WHERE clustered = TRUE`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count clustered articles: %w", err)
	}
	return count, nil
}

func scanClusters(rows pgx.Rows) ([]entity.Cluster, error) {
	var clusters []entity.Cluster
	for rows.Next() {
		var cluster entity.Cluster
		if err := rows.Scan(
			&cluster.ID,
			&cluster.Title,
			&cluster.Category,
			&cluster.ArticleCount,
			&cluster.Confidence,
			&cluster.CreatedAt,
			&cluster.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan cluster: %w", err)
		}
		clusters = append(clusters, cluster)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("cluster rows error: %w", err)
	}
	return clusters, nil
}

func nullableString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
