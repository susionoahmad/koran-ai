package entity

import (
	"time"

	"github.com/google/uuid"
)

// Article represents a news article stored in the database.
type Article struct {
	ID            uuid.UUID  `json:"id"`
	SourceID      uuid.UUID  `json:"source_id"`
	Title         string     `json:"title"`
	Slug          string     `json:"slug"`
	URL           string     `json:"url"`
	Author        *string    `json:"author,omitempty"`
	Content       string     `json:"content"`
	PublishedAt   *time.Time `json:"published_at"`
	ScrapedAt     time.Time  `json:"scraped_at"`
	HashContent   string     `json:"hash_content"`
	ImageURL      *string    `json:"image_url,omitempty"`
	Processed     bool       `json:"processed"`
	AIProcessed   bool       `json:"ai_processed"`
	AICategory    *string    `json:"ai_category,omitempty"`
	AIConfidence  float64    `json:"ai_confidence,omitempty"`
	AIProcessedAt *time.Time `json:"ai_processed_at,omitempty"`
	AIError       *string    `json:"ai_error,omitempty"`
	AIRetryCount  int        `json:"ai_retry_count"`
	Clustered     bool       `json:"clustered"`
	ClusterID     *uuid.UUID `json:"cluster_id,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}
