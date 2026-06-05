package dto

import "time"

type CreateSourceRequest struct {
	Name       string `json:"name" validate:"required,max=255"`
	BaseURL    string `json:"base_url" validate:"required,url"`
	RSSURL     string `json:"rss_url" validate:"required,url"`
	SourceType string `json:"source_type" validate:"required,oneof=RSS SITEMAP SCRAPER"`
}

type UpdateSourceRequest struct {
	Name       string `json:"name" validate:"required,max=255"`
	BaseURL    string `json:"base_url" validate:"required,url"`
	RSSURL     string `json:"rss_url" validate:"required,url"`
	SourceType string `json:"source_type" validate:"required,oneof=RSS SITEMAP SCRAPER"`
	IsActive   *bool  `json:"is_active" validate:"required"`
}

type SourceResponse struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	BaseURL    string    `json:"base_url"`
	RSSURL     string    `json:"rss_url"`
	SourceType string    `json:"source_type"`
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type SourceListResponse struct {
	Sources []SourceResponse `json:"sources"`
	Total   int64            `json:"total"`
	Page    int              `json:"page"`
	Limit   int              `json:"limit"`
}
