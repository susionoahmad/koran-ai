package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"koran-ai-backend/internal/shared/config"
	"koran-ai-backend/internal/shared/database"
	"koran-ai-backend/internal/source/entity"
)

func TestPostgresRepository_CRUD(t *testing.T) {
	// Attempt to load dev/test config
	cfg := config.InitConfig()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	dbPool, err := database.ConnectPostgres(ctx, cfg)
	if err != nil {
		t.Skip("PostgreSQL database is offline or not configured. Skipping integration repository tests.")
		return
	}
	defer dbPool.Close()

	repo := NewPostgresRepository(dbPool)

	// Clean up after test
	testName := "Test Integration Source " + uuid.New().String()[:8]
	testBaseURL := "https://test-integration-" + uuid.New().String()[:8] + ".com"
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		_, _ = dbPool.Exec(cleanupCtx, "DELETE FROM sources WHERE name = $1 OR base_url = $2", testName, testBaseURL)
	}()

	// 1. Create Test
	src := &entity.Source{
		ID:         uuid.New(),
		Name:       testName,
		BaseURL:    testBaseURL,
		RSSURL:     testBaseURL + "/rss",
		SourceType: "RSS",
		IsActive:   true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	err = repo.Create(ctx, src)
	if err != nil {
		t.Fatalf("Failed to create source: %v", err)
	}

	// 2. GetByID Test
	fetched, err := repo.GetByID(ctx, src.ID.String())
	if err != nil {
		t.Fatalf("Failed to get source by ID: %v", err)
	}
	if fetched.Name != src.Name {
		t.Errorf("Expected name %s, got %s", src.Name, fetched.Name)
	}

	// 3. GetByName Test
	fetchedByName, err := repo.GetByName(ctx, src.Name)
	if err != nil {
		t.Fatalf("Failed to get source by name: %v", err)
	}
	if fetchedByName.ID != src.ID {
		t.Errorf("Expected ID %s, got %s", src.ID, fetchedByName.ID)
	}

	// 4. GetByBaseURL Test
	fetchedByBaseURL, err := repo.GetByBaseURL(ctx, src.BaseURL)
	if err != nil {
		t.Fatalf("Failed to get source by base URL: %v", err)
	}
	if fetchedByBaseURL.ID != src.ID {
		t.Errorf("Expected ID %s, got %s", src.ID, fetchedByBaseURL.ID)
	}

	// 5. Update Test
	src.Name = testName + " Updated"
	err = repo.Update(ctx, src)
	if err != nil {
		t.Fatalf("Failed to update source: %v", err)
	}

	fetchedUpdated, err := repo.GetByID(ctx, src.ID.String())
	if err != nil {
		t.Fatalf("Failed to get updated source: %v", err)
	}
	if fetchedUpdated.Name != testName+" Updated" {
		t.Errorf("Expected name '%s Updated', got %s", testName, fetchedUpdated.Name)
	}

	// 6. List Test
	sources, total, err := repo.List(ctx, 1, 10)
	if err != nil {
		t.Fatalf("Failed to list sources: %v", err)
	}
	if total < 1 {
		t.Errorf("Expected total sources >= 1, got %d", total)
	}
	found := false
	for _, s := range sources {
		if s.ID == src.ID {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find created source in listed sources")
	}

	// 7. Delete Test (Soft Delete)
	err = repo.Delete(ctx, src.ID.String())
	if err != nil {
		t.Fatalf("Failed to delete source: %v", err)
	}

	fetchedDeleted, err := repo.GetByID(ctx, src.ID.String())
	if err != nil {
		t.Fatalf("Failed to get source after deletion: %v", err)
	}
	if fetchedDeleted.IsActive {
		t.Error("Expected source to be inactive (soft deleted)")
	}

	// 8. Get non-existent
	_, err = repo.GetByID(ctx, uuid.New().String())
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("Expected ErrNotFound for invalid ID, got %v", err)
	}
}
