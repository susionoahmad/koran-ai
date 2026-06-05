package service

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	articleEntity "koran-ai-backend/internal/article/entity"
	articleRepo "koran-ai-backend/internal/article/repository"
	"koran-ai-backend/internal/crawler/rss"
	sourceEntity "koran-ai-backend/internal/source/entity"
)

var (
	ErrSourceInactive = errors.New("source is inactive")
	ErrSourceNoRSS    = errors.New("source has no rss_url configured")
)

// spaceRe matches one or more consecutive spaces or tabs.
var spaceRe = regexp.MustCompile(`[ \t]+`)

// newlineRe matches three or more consecutive line breaks.
var newlineRe = regexp.MustCompile(`\n{3,}`)

// nonAlphaRe matches characters that are not lowercase alphanumeric, spaces, or hyphens.
var nonAlphaRe = regexp.MustCompile(`[^a-z0-9\s\-]`)

// multiHyphenRe matches multiple consecutive hyphens.
var multiHyphenRe = regexp.MustCompile(`[\s\-]+`)

// SourceRepository is the minimal interface the crawler needs from the source module.
type SourceRepository interface {
	GetByID(ctx context.Context, id string) (*sourceEntity.Source, error)
	ListActive(ctx context.Context) ([]sourceEntity.Source, error)
	List(ctx context.Context, page, limit int) ([]sourceEntity.Source, int64, error)
}

// CrawlResult summarises the outcome of one crawl run.
type CrawlResult struct {
	SourceID      string
	ArticlesFound int
	ArticlesSaved int
	Error         error
}

// CrawlerStats contains system stats for the crawler dashboard.
type CrawlerStats struct {
	Sources       int64  `json:"sources"`
	ArticlesTotal int64  `json:"articles_total"`
	ArticlesToday int64  `json:"articles_today"`
	LastCrawl     string `json:"last_crawl"`
	FailedCrawls  int64  `json:"failed_crawls"`
}

// Service defines the crawler's business-logic interface.
type Service interface {
	RunSource(ctx context.Context, sourceID string) (*CrawlResult, error)
	RunAllSources(ctx context.Context) ([]*CrawlResult, error)
	GetStats(ctx context.Context) (*CrawlerStats, error)
}

type crawlerService struct {
	sourceRepo   SourceRepository
	articleRepo  articleRepo.Repository
	crawlLogRepo CrawlLogRepository
	parser       rss.Parser
}

// NewCrawlerService constructs a CrawlerService with all dependencies injected.
func NewCrawlerService(
	srcRepo SourceRepository,
	artRepo articleRepo.Repository,
	logRepo CrawlLogRepository,
	parser rss.Parser,
) Service {
	return &crawlerService{
		sourceRepo:   srcRepo,
		articleRepo:  artRepo,
		crawlLogRepo: logRepo,
		parser:       parser,
	}
}

// RunSource executes a full crawl cycle for one source identified by sourceID.
func (s *crawlerService) RunSource(ctx context.Context, sourceID string) (*CrawlResult, error) {
	startTime := time.Now()

	// 1. Fetch source
	src, err := s.sourceRepo.GetByID(ctx, sourceID)
	if err != nil {
		return nil, fmt.Errorf("source lookup failed: %w", err)
	}

	// 2. Validate source
	if !src.IsActive {
		return nil, ErrSourceInactive
	}
	if src.RSSURL == "" {
		return nil, ErrSourceNoRSS
	}

	// 3. Create crawl log
	crawlLog := &CrawlLog{
		SourceID:   sourceID,
		SourceName: src.Name,
		StartedAt:  startTime,
	}
	logID, err := s.crawlLogRepo.Create(ctx, crawlLog)
	if err != nil {
		// non-fatal: continue crawling even if log creation fails
		logID = 0
	}

	result := &CrawlResult{SourceID: sourceID}
	crawlStatus := "SUCCESS"
	errMsg := ""

	// 4. Fetch & parse RSS
	items, err := s.parser.Parse(ctx, src.RSSURL)
	if err != nil {
		crawlStatus = "FAILED"
		errMsg = err.Error()
		result.Error = err
		if logID > 0 {
			durationMs := time.Since(startTime).Milliseconds()
			_ = s.crawlLogRepo.Finish(ctx, logID, crawlStatus, 0, 0, 0, durationMs, errMsg)
		}
		return result, err
	}
	result.ArticlesFound = len(items)

	// 5. For each item: deduplicate then save
	saved := 0
	skipped := 0
	for _, item := range items {
		// Normalization
		normTitle := NormalizeText(item.Title)
		normContent := NormalizeText(item.Content)

		// Quality Filter Checks
		if normTitle == "" || len(normContent) < 100 || item.PublishedAt == nil {
			skipped++
			continue
		}

		// Check URL duplicate
		exists, err := s.articleRepo.ExistsByURL(ctx, item.URL)
		if err != nil {
			continue
		}
		if exists {
			continue
		}

		// Compute SHA-256 hash of title + content
		hash := ComputeHash(normTitle + normContent)

		// Check hash duplicate
		exists, err = s.articleRepo.ExistsByHash(ctx, hash)
		if err != nil {
			continue
		}
		if exists {
			continue
		}

		// Build article entity
		now := time.Now()
		article := &articleEntity.Article{
			ID:          uuid.New(),
			SourceID:    src.ID,
			Title:       normTitle,
			Slug:        ToSlug(normTitle),
			URL:         item.URL,
			Author:      item.Author,
			Content:     normContent,
			PublishedAt: item.PublishedAt,
			ScrapedAt:   now,
			HashContent: hash,
			ImageURL:    item.ImageURL,
			Processed:   false,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		// Save article
		if err := s.articleRepo.Create(ctx, article); err != nil {
			if errors.Is(err, articleRepo.ErrDuplicateURL) || errors.Is(err, articleRepo.ErrDuplicateHash) {
				continue // race condition – skip silently
			}
			// log but continue processing other articles
			continue
		}
		saved++
	}
	result.ArticlesSaved = saved

	// 6. Finish crawl log
	durationMs := time.Since(startTime).Milliseconds()
	if logID > 0 {
		_ = s.crawlLogRepo.Finish(ctx, logID, crawlStatus, result.ArticlesFound, saved, skipped, durationMs, errMsg)
	}

	return result, nil
}

// RunAllSources iterates over all active sources and crawls each one.
func (s *crawlerService) RunAllSources(ctx context.Context) ([]*CrawlResult, error) {
	sources, err := s.sourceRepo.ListActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list active sources: %w", err)
	}

	results := make([]*CrawlResult, 0, len(sources))
	for _, src := range sources {
		res, err := s.RunSource(ctx, src.ID.String())
		if err != nil {
			results = append(results, &CrawlResult{
				SourceID: src.ID.String(),
				Error:    err,
			})
			continue
		}
		results = append(results, res)
	}
	return results, nil
}

// GetStats returns system stats for crawler dashboard.
func (s *crawlerService) GetStats(ctx context.Context) (*CrawlerStats, error) {
	_, totalSources, err := s.sourceRepo.List(ctx, 1, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to query sources total: %w", err)
	}

	totalArticles, err := s.articleRepo.CountTotal(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query articles total: %w", err)
	}

	articlesToday, err := s.articleRepo.CountToday(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query articles today: %w", err)
	}

	lastCrawl, failedCrawls, err := s.crawlLogRepo.GetStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query crawl logs stats: %w", err)
	}

	lastCrawlStr := ""
	if lastCrawl != nil {
		lastCrawlStr = lastCrawl.Format(time.RFC3339)
	}

	return &CrawlerStats{
		Sources:       totalSources,
		ArticlesTotal: totalArticles,
		ArticlesToday: articlesToday,
		LastCrawl:     lastCrawlStr,
		FailedCrawls:  failedCrawls,
	}, nil
}

// NormalizeText normalizes whitespace, spaces, and line breaks.
func NormalizeText(txt string) string {
	txt = strings.ReplaceAll(txt, "\r\n", "\n")
	txt = strings.ReplaceAll(txt, "\r", "\n")
	txt = spaceRe.ReplaceAllString(txt, " ")
	// Trim leading/trailing spaces from each line
	lines := strings.Split(txt, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimSpace(l)
	}
	txt = strings.Join(lines, "\n")
	txt = strings.TrimSpace(txt)
	txt = newlineRe.ReplaceAllString(txt, "\n\n")
	return txt
}

// ComputeHash returns the SHA-256 hex digest of the input string.
func ComputeHash(input string) string {
	h := sha256.Sum256([]byte(input))
	return fmt.Sprintf("%x", h)
}

// ToSlug converts a title to a URL-safe slug (lowercase, hyphens).
func ToSlug(title string) string {
	s := strings.ToLower(title)
	s = nonAlphaRe.ReplaceAllString(s, "")
	s = multiHyphenRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if len(s) > 500 {
		s = s[:500]
	}
	return s
}
