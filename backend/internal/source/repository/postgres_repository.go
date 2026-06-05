package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"koran-ai-backend/internal/source/entity"
)

var (
	ErrNotFound = errors.New("source not found")
)

type postgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) Repository {
	return &postgresRepository{db: db}
}

func (r *postgresRepository) Create(ctx context.Context, s *entity.Source) error {
	query := `
		INSERT INTO sources (id, name, base_url, rss_url, source_type, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.Exec(ctx, query, s.ID, s.Name, s.BaseURL, s.RSSURL, s.SourceType, s.IsActive, s.CreatedAt, s.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert source: %w", err)
	}
	return nil
}

func (r *postgresRepository) GetByID(ctx context.Context, id string) (*entity.Source, error) {
	query := `
		SELECT id, name, base_url, rss_url, source_type, is_active, created_at, updated_at
		FROM sources
		WHERE id = $1
	`
	var s entity.Source
	err := r.db.QueryRow(ctx, query, id).Scan(
		&s.ID, &s.Name, &s.BaseURL, &s.RSSURL, &s.SourceType, &s.IsActive, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get source by ID: %w", err)
	}
	return &s, nil
}

func (r *postgresRepository) GetByName(ctx context.Context, name string) (*entity.Source, error) {
	query := `
		SELECT id, name, base_url, rss_url, source_type, is_active, created_at, updated_at
		FROM sources
		WHERE name = $1
	`
	var s entity.Source
	err := r.db.QueryRow(ctx, query, name).Scan(
		&s.ID, &s.Name, &s.BaseURL, &s.RSSURL, &s.SourceType, &s.IsActive, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get source by name: %w", err)
	}
	return &s, nil
}

func (r *postgresRepository) GetByBaseURL(ctx context.Context, baseURL string) (*entity.Source, error) {
	query := `
		SELECT id, name, base_url, rss_url, source_type, is_active, created_at, updated_at
		FROM sources
		WHERE base_url = $1
	`
	var s entity.Source
	err := r.db.QueryRow(ctx, query, baseURL).Scan(
		&s.ID, &s.Name, &s.BaseURL, &s.RSSURL, &s.SourceType, &s.IsActive, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get source by base URL: %w", err)
	}
	return &s, nil
}

func (r *postgresRepository) List(ctx context.Context, page, limit int) ([]entity.Source, int64, error) {
	var total int64
	countQuery := `SELECT COUNT(*) FROM sources`
	err := r.db.QueryRow(ctx, countQuery).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count sources: %w", err)
	}

	offset := (page - 1) * limit
	query := `
		SELECT id, name, base_url, rss_url, source_type, is_active, created_at, updated_at
		FROM sources
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list sources: %w", err)
	}
	defer rows.Close()

	var sources []entity.Source
	for rows.Next() {
		var s entity.Source
		err := rows.Scan(&s.ID, &s.Name, &s.BaseURL, &s.RSSURL, &s.SourceType, &s.IsActive, &s.CreatedAt, &s.UpdatedAt)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan source row: %w", err)
		}
		sources = append(sources, s)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows error: %w", err)
	}

	return sources, total, nil
}

func (r *postgresRepository) Update(ctx context.Context, s *entity.Source) error {
	query := `
		UPDATE sources
		SET name = $1, base_url = $2, rss_url = $3, source_type = $4, is_active = $5, updated_at = $6
		WHERE id = $7
	`
	cmdTag, err := r.db.Exec(ctx, query, s.Name, s.BaseURL, s.RSSURL, s.SourceType, s.IsActive, s.UpdatedAt, s.ID)
	if err != nil {
		return fmt.Errorf("failed to update source: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *postgresRepository) Delete(ctx context.Context, id string) error {
	// Soft delete: set is_active = false
	query := `
		UPDATE sources
		SET is_active = false, updated_at = $1
		WHERE id = $2
	`
	cmdTag, err := r.db.Exec(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to soft delete source: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
