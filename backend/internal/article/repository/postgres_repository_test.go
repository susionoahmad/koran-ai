package repository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	articleEntity "koran-ai-backend/internal/article/entity"
	articleRepo "koran-ai-backend/internal/article/repository"
	"koran-ai-backend/internal/shared/config"
	"koran-ai-backend/internal/shared/database"
	sourceEntity "koran-ai-backend/internal/source/entity"
	sourceRepo "koran-ai-backend/internal/source/repository"
)

func TestPostgresRepository_Articles(t *testing.T) {
	// Attempt to load dev/test config
	cfg := config.InitConfig()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dbPool, err := database.ConnectPostgres(ctx, cfg)
	if err != nil {
		t.Skip("PostgreSQL database is offline or not configured. Skipping integration repository tests.")
		return
	}
	defer dbPool.Close()

	srcRepo := sourceRepo.NewPostgresRepository(dbPool)
	artRepo := articleRepo.NewPostgresRepository(dbPool)

	// Clean up at start and end
	testURL := "https://test-article-" + uuid.New().String()[:8] + ".com"
	testHash := "hash-" + uuid.New().String()[:8]
	sourceName := "Source Article Integration " + uuid.New().String()[:8]

	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		_, _ = dbPool.Exec(cleanupCtx, "DELETE FROM articles WHERE url = $1", testURL)
		_, _ = dbPool.Exec(cleanupCtx, "DELETE FROM sources WHERE name = $1", sourceName)
	}()

	// Create a dummy source first
	src := &sourceEntity.Source{
		ID:         uuid.New(),
		Name:       sourceName,
		BaseURL:    "https://base-article.com",
		RSSURL:     "https://base-article.com/rss",
		SourceType: "rss",
		IsActive:   true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	err = srcRepo.Create(ctx, src)
	if err != nil {
		t.Fatalf("Failed to create dummy source: %v", err)
	}

	// 1. Check ExistsByURL before insertion
	exists, err := artRepo.ExistsByURL(ctx, testURL)
	if err != nil {
		t.Fatalf("Failed to check url existence: %v", err)
	}
	if exists {
		t.Error("Expected URL to not exist yet")
	}

	// 2. Create Article Test
	pubTime := time.Now().Truncate(time.Second)
	art := &articleEntity.Article{
		ID:          uuid.New(),
		SourceID:    src.ID,
		Title:       "Test Article Integration",
		Slug:        "test-article-integration",
		URL:         testURL,
		Author:      "Author",
		Content:     "Hello world test article content",
		PublishedAt: &pubTime,
		ScrapedAt:   time.Now(),
		HashContent: testHash,
		ImageURL:    "https://example.com/img.jpg",
		Processed:   false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err = artRepo.Create(ctx, art)
	if err != nil {
		t.Fatalf("Failed to create article: %v", err)
	}

	// Try inserting same URL to trigger duplicate error
	err = artRepo.Create(ctx, art)
	if !errors.Is(err, articleRepo.ErrDuplicateURL) {
		t.Errorf("Expected ErrDuplicateURL, got %v", err)
	}

	// Try inserting same Hash but different URL to trigger duplicate hash error
	art2 := *art
	art2.ID = uuid.New()
	art2.URL = testURL + "/different"
	err = artRepo.Create(ctx, &art2)
	if !errors.Is(err, articleRepo.ErrDuplicateHash) {
		t.Errorf("Expected ErrDuplicateHash, got %v", err)
	}

	// 3. ExistsByURL and ExistsByHash Tests
	exists, err = artRepo.ExistsByURL(ctx, testURL)
	if err != nil {
		t.Fatalf("ExistsByURL failed: %v", err)
	}
	if !exists {
		t.Error("Expected ExistsByURL to be true")
	}

	exists, err = artRepo.ExistsByHash(ctx, testHash)
	if err != nil {
		t.Fatalf("ExistsByHash failed: %v", err)
	}
	if !exists {
		t.Error("Expected ExistsByHash to be true")
	}

	// 4. GetByID Test
	fetched, err := artRepo.GetByID(ctx, art.ID.String())
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if fetched.Title != art.Title {
		t.Errorf("Expected title %s, got %s", art.Title, fetched.Title)
	}

	// 5. ListUnprocessed Test
	unprocessed, err := artRepo.ListUnprocessed(ctx, 10)
	if err != nil {
		t.Fatalf("ListUnprocessed failed: %v", err)
	}
	found := false
	for _, a := range unprocessed {
		if a.ID == art.ID {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find created article in unprocessed list")
	}

	// 6. Get non-existent
	_, err = artRepo.GetByID(ctx, uuid.New().String())
	if !errors.Is(err, articleRepo.ErrNotFound) {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}
