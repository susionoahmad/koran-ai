package entity

import (
	"time"

	"github.com/google/uuid"
)

type Source struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	BaseURL    string    `json:"base_url"`
	RSSURL     string    `json:"rss_url"`
	SourceType string    `json:"source_type"` // RSS, SITEMAP, SCRAPER
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
