package service

import (
	"context"
	"time"
)

// CrawlLog represents a single crawl run record.
type CrawlLog struct {
	ID              int64
	SourceID        string
	SourceName      string
	StartedAt       time.Time
	FinishedAt      *time.Time
	DurationMs      int64
	Status          string // RUNNING | SUCCESS | FAILED
	ArticlesFound   int
	ArticlesSaved   int
	ArticlesSkipped int
	ErrorMessage    string
}

// CrawlLogRepository defines persistence operations for crawl logs.
type CrawlLogRepository interface {
	Create(ctx context.Context, log *CrawlLog) (int64, error)
	Finish(ctx context.Context, id int64, status string, found, saved, skipped int, durationMs int64, errMsg string) error
	GetStats(ctx context.Context) (lastCrawl *time.Time, failedCrawls int64, err error)
}
