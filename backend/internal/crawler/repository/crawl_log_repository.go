package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	crawlerService "koran-ai-backend/internal/crawler/service"
)

type pgCrawlLogRepository struct {
	db *pgxpool.Pool
}

// NewCrawlLogRepository creates a new PostgreSQL-backed crawl log repository.
func NewCrawlLogRepository(db *pgxpool.Pool) crawlerService.CrawlLogRepository {
	return &pgCrawlLogRepository{db: db}
}

func (r *pgCrawlLogRepository) Create(ctx context.Context, log *crawlerService.CrawlLog) (int64, error) {
	query := `
		INSERT INTO crawl_logs (source_id, source_name, started_at, status, articles_found, articles_saved, articles_skipped)
		VALUES ($1, $2, $3, 'RUNNING', 0, 0, 0)
		RETURNING id
	`
	var id int64
	err := r.db.QueryRow(ctx, query, log.SourceID, log.SourceName, log.StartedAt).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to create crawl log: %w", err)
	}
	return id, nil
}

func (r *pgCrawlLogRepository) Finish(ctx context.Context, id int64, status string, found, saved, skipped int, durationMs int64, errMsg string) error {
	now := time.Now()
	query := `
		UPDATE crawl_logs
		SET finished_at = $1, status = $2, articles_found = $3, articles_saved = $4, articles_skipped = $5, duration_ms = $6, error_message = $7
		WHERE id = $8
	`
	_, err := r.db.Exec(ctx, query, now, status, found, saved, skipped, durationMs, errMsg, id)
	if err != nil {
		return fmt.Errorf("failed to update crawl log: %w", err)
	}
	return nil
}

func (r *pgCrawlLogRepository) GetStats(ctx context.Context) (lastCrawl *time.Time, failedCrawls int64, err error) {
	// 1. Get last successful crawl timestamp
	lastQuery := `SELECT finished_at FROM crawl_logs WHERE status = 'SUCCESS' ORDER BY finished_at DESC LIMIT 1`
	var last time.Time
	err = r.db.QueryRow(ctx, lastQuery).Scan(&last)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			lastCrawl = nil
		} else {
			return nil, 0, fmt.Errorf("failed to get last successful crawl: %w", err)
		}
	} else {
		lastCrawl = &last
	}

	// 2. Count total failed crawls
	failedQuery := `SELECT COUNT(*) FROM crawl_logs WHERE status = 'FAILED'`
	err = r.db.QueryRow(ctx, failedQuery).Scan(&failedCrawls)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count failed crawls: %w", err)
	}

	return lastCrawl, failedCrawls, nil
}
