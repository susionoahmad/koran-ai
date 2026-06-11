package entity

import (
	"time"

	"github.com/google/uuid"
)

// Edition represents a daily newspaper edition.
type Edition struct {
	ID                uuid.UUID  `json:"id"`
	EditionDate       time.Time  `json:"edition_date"`
	Title             string     `json:"title"`
	HeadlineClusterID *uuid.UUID `json:"headline_cluster_id,omitempty"`
	TotalArticles     int        `json:"total_articles"`
	Status            string     `json:"status"`
	GeneratedAt       time.Time  `json:"generated_at"`
	PublishedAt       *time.Time `json:"published_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// EditionArticle links a cluster to a specific edition and section.
type EditionArticle struct {
	ID           uuid.UUID `json:"id"`
	EditionID    uuid.UUID `json:"edition_id"`
	ClusterID    uuid.UUID `json:"cluster_id"`
	Section      string    `json:"section"`
	DisplayOrder int       `json:"display_order"`
	CreatedAt    time.Time `json:"created_at"`
}

// EditionArticleSummary represents summarized details of a cluster for an edition.
type EditionArticleSummary struct {
	ClusterID     uuid.UUID `json:"cluster_id"`
	Title         string    `json:"title"`
	SummaryShort  string    `json:"summary_short"`
	SummaryMedium string    `json:"summary_medium,omitempty"`
	SummaryLong   string    `json:"summary_long,omitempty"`
	KeyPoints     []string  `json:"key_points,omitempty"`
	Category      string    `json:"category,omitempty"`
	ArticleCount  int       `json:"article_count"`
	Confidence    float64   `json:"confidence"`
	AIModel       string    `json:"ai_model,omitempty"`
	AIConfidence  float64   `json:"ai_confidence"`
	GeneratedAt   time.Time `json:"generated_at"`
}

// SectionDetail groups articles under a section name.
type SectionDetail struct {
	Name     string                  `json:"name"`
	Articles []EditionArticleSummary `json:"articles"`
}

// EditionDetailResponse is the detailed JSON response shape for public APIs.
type EditionDetailResponse struct {
	ID                uuid.UUID              `json:"id"`
	EditionDate       string                 `json:"edition_date"`
	Title             string                 `json:"title"`
	Status            string                 `json:"status"`
	HeadlineClusterID *uuid.UUID             `json:"headline_cluster_id,omitempty"`
	Headline          *EditionArticleSummary `json:"headline,omitempty"`
	Sections          []SectionDetail        `json:"sections"`
}
