package repository

import (
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
)

type fakeArticleScanner struct {
	values []any
}

func (f fakeArticleScanner) Scan(dest ...any) error {
	for i := range dest {
		switch d := dest[i].(type) {
		case *uuid.UUID:
			*d = f.values[i].(uuid.UUID)
		case *string:
			*d = f.values[i].(string)
		case *sql.NullString:
			if f.values[i] == nil {
				*d = sql.NullString{}
			} else {
				*d = sql.NullString{String: f.values[i].(string), Valid: true}
			}
		case *time.Time:
			*d = f.values[i].(time.Time)
		case **time.Time:
			if f.values[i] == nil {
				*d = nil
			} else {
				value := f.values[i].(time.Time)
				*d = &value
			}
		case *bool:
			*d = f.values[i].(bool)
		case *float64:
			*d = f.values[i].(float64)
		case *int:
			*d = f.values[i].(int)
		}
	}
	return nil
}

func TestScanArticle_NullableStrings(t *testing.T) {
	now := time.Now()
	articleID := uuid.New()
	sourceID := uuid.New()

	article, err := scanArticle(fakeArticleScanner{values: []any{
		articleID,
		sourceID,
		"Bank Indonesia Pertahankan BI Rate",
		"bank-indonesia-pertahankan-bi-rate",
		"https://example.com/article",
		nil, // author
		"content",
		now,
		now,
		"hash",
		nil, // image_url
		false,
		true,
		nil, // ai_category
		0.91,
		now,
		false,
		nil,
		now,
		now,
		nil, // ai_error
		0,
	}})
	if err != nil {
		t.Fatalf("scanArticle returned error: %v", err)
	}

	if article.Author != nil {
		t.Fatalf("expected nil author, got %q", *article.Author)
	}
	if article.ImageURL != nil {
		t.Fatalf("expected nil image_url, got %q", *article.ImageURL)
	}
	if article.AICategory != nil {
		t.Fatalf("expected nil ai_category, got %q", *article.AICategory)
	}
	if article.AIError != nil {
		t.Fatalf("expected nil ai_error, got %q", *article.AIError)
	}
}

func TestScanArticle_NonNullNullableStrings(t *testing.T) {
	now := time.Now()

	article, err := scanArticle(fakeArticleScanner{values: []any{
		uuid.New(),
		uuid.New(),
		"Title",
		"slug",
		"https://example.com/article",
		"Reporter",
		"content",
		now,
		now,
		"hash",
		"https://example.com/image.jpg",
		false,
		true,
		"ekonomi",
		0.91,
		now,
		false,
		nil,
		now,
		now,
		"previous error",
		1,
	}})
	if err != nil {
		t.Fatalf("scanArticle returned error: %v", err)
	}

	if article.Author == nil || *article.Author != "Reporter" {
		t.Fatalf("expected author Reporter, got %#v", article.Author)
	}
	if article.ImageURL == nil || *article.ImageURL != "https://example.com/image.jpg" {
		t.Fatalf("expected image url, got %#v", article.ImageURL)
	}
	if article.AICategory == nil || *article.AICategory != "ekonomi" {
		t.Fatalf("expected category ekonomi, got %#v", article.AICategory)
	}
	if article.AIError == nil || *article.AIError != "previous error" {
		t.Fatalf("expected ai error, got %#v", article.AIError)
	}
}
