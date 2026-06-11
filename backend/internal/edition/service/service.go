package service

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	appLogger "koran-ai-backend/internal/shared/logger"
	"koran-ai-backend/internal/edition/entity"
	"koran-ai-backend/internal/edition/repository"
)

var (
	ErrWorkerAlreadyRunning = errors.New("edition worker already running")
	ErrNoSummariesFound     = errors.New("no summaries found for this date")
)

type Service interface {
	GenerateEdition(ctx context.Context, dateStr string) (*entity.Edition, error)
	GetStats(ctx context.Context) (int64, string, error)
	ListEditions(ctx context.Context, page, limit int) ([]entity.Edition, int64, error)
	GetEditionByID(ctx context.Context, id string) (*entity.EditionDetailResponse, error)
	GetEditionByDate(ctx context.Context, dateStr string) (*entity.EditionDetailResponse, error)
}

type editionService struct {
	repo   repository.Repository
	rdb    redis.Cmdable
	logger appLogger.Logger
}

func NewService(repo repository.Repository, rdb redis.Cmdable, logger appLogger.Logger) Service {
	return &editionService{repo: repo, rdb: rdb, logger: logger}
}

func (s *editionService) GenerateEdition(ctx context.Context, dateStr string) (*entity.Edition, error) {
	parsedDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil, fmt.Errorf("invalid date format: %w", err)
	}

	// 1. Acquire Redis Lock
	lockAcquired, err := s.acquireLock(ctx)
	if err != nil {
		return nil, err
	}
	if lockAcquired {
		defer s.releaseLock(ctx)
	}

	// 2. Check if edition already exists
	exists, err := s.repo.EditionExists(ctx, parsedDate)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, repository.ErrAlreadyExists
	}

	// 3. Load summaries for date
	summaries, err := s.repo.LoadSummariesForDate(ctx, parsedDate)
	if err != nil {
		return nil, err
	}
	if len(summaries) == 0 {
		return nil, ErrNoSummariesFound
	}

	// 4. Headline Selection: sort by article_count desc, then confidence desc, then generated_at desc
	sortSummaries(summaries)

	headlineClusterID := summaries[0].ClusterID

	// Group and build edition articles
	var editionArticles []entity.EditionArticle
	now := time.Now()
	editionID := uuid.New()

	// The first summary in sorted list is chosen as the headline
	headlineArt := entity.EditionArticle{
		ID:           uuid.New(),
		EditionID:    editionID,
		ClusterID:    headlineClusterID,
		Section:      "Headline News",
		DisplayOrder: 1,
		CreatedAt:    now,
	}
	editionArticles = append(editionArticles, headlineArt)

	// Group remaining summaries by section category
	sectionsMap := make(map[string][]entity.EditionArticleSummary)
	for i := 1; i < len(summaries); i++ {
		sec := MapCategory(summaries[i].Category)
		sectionsMap[sec] = append(sectionsMap[sec], summaries[i])
	}

	// For each section, sort and assign display orders
	for secName, arts := range sectionsMap {
		sortSummaries(arts)
		for order, art := range arts {
			editionArticles = append(editionArticles, entity.EditionArticle{
				ID:           uuid.New(),
				EditionID:    editionID,
				ClusterID:    art.ClusterID,
				Section:      secName,
				DisplayOrder: order + 1,
				CreatedAt:    now,
			})
		}
	}

	// 5. Create Edition Entity
	editionTitle := fmt.Sprintf("Koran AI - Edisi %s", dateStr)
	edition := &entity.Edition{
		ID:                editionID,
		EditionDate:       parsedDate,
		Title:             editionTitle,
		HeadlineClusterID: &headlineClusterID,
		TotalArticles:     len(editionArticles),
		Status:            "DRAFT",
		GeneratedAt:       now,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	// 6. Save using Repository transaction
	if err := s.repo.CreateEdition(ctx, edition, editionArticles); err != nil {
		return nil, err
	}

	return edition, nil
}

func (s *editionService) GetStats(ctx context.Context) (int64, string, error) {
	return s.repo.GetStats(ctx)
}

func (s *editionService) ListEditions(ctx context.Context, page, limit int) ([]entity.Edition, int64, error) {
	return s.repo.ListEditions(ctx, page, limit)
}

func (s *editionService) GetEditionByID(ctx context.Context, id string) (*entity.EditionDetailResponse, error) {
	return s.repo.GetEditionDetails(ctx, id)
}

func (s *editionService) GetEditionByDate(ctx context.Context, dateStr string) (*entity.EditionDetailResponse, error) {
	// Parse the date to ensure it is valid
	if _, err := time.Parse("2006-01-02", dateStr); err != nil {
		return nil, fmt.Errorf("invalid date format: %w", err)
	}

	ed, err := s.repo.GetEditionByDate(ctx, dateStr)
	if err != nil {
		return nil, err
	}

	return s.repo.GetEditionDetails(ctx, ed.ID.String())
}

func (s *editionService) acquireLock(ctx context.Context) (bool, error) {
	if isNilRedis(s.rdb) {
		s.warn("redis client is nil, running edition generator without lock")
		return false, nil
	}
	acquired, err := s.rdb.SetNX(ctx, "edition_generator_lock", "running", 10*time.Minute).Result()
	if err != nil {
		s.warn("redis lock acquisition failed, running without lock", zap.Error(err))
		return false, nil
	}
	if !acquired {
		return false, ErrWorkerAlreadyRunning
	}
	return true, nil
}

func (s *editionService) releaseLock(ctx context.Context) {
	if isNilRedis(s.rdb) {
		return
	}
	_, _ = s.rdb.Del(ctx, "edition_generator_lock").Result()
}

func (s *editionService) warn(msg string, fields ...zap.Field) {
	if s.logger != nil {
		s.logger.Warn(msg, fields...)
	}
}

func isNilRedis(rdb redis.Cmdable) bool {
	if rdb == nil {
		return true
	}
	value := reflect.ValueOf(rdb)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

// MapCategory assigns Indonesian categories to English layout sections.
func MapCategory(cat string) string {
	switch strings.ToLower(strings.TrimSpace(cat)) {
	case "politik":
		return "Politics"
	case "nasional":
		return "National"
	case "bisnis", "ekonomi":
		return "Economy"
	case "internasional":
		return "International"
	case "teknologi":
		return "Technology"
	case "olahraga":
		return "Sports"
	case "kesehatan":
		return "Health"
	case "pendidikan":
		return "Education"
	case "travel":
		return "Lifestyle"
	default:
		return "General"
	}
}

// sortSummaries orders summaries by article_count desc, then confidence desc, then generated_at desc
func sortSummaries(summaries []entity.EditionArticleSummary) {
	sort.Slice(summaries, func(i, j int) bool {
		if summaries[i].ArticleCount != summaries[j].ArticleCount {
			return summaries[i].ArticleCount > summaries[j].ArticleCount
		}
		if summaries[i].Confidence != summaries[j].Confidence {
			return summaries[i].Confidence > summaries[j].Confidence
		}
		return summaries[i].GeneratedAt.After(summaries[j].GeneratedAt)
	})
}
