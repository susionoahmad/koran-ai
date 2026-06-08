package entity

import (
	"time"

	"github.com/google/uuid"
)

// Cluster represents a group of articles discussing the same news event.
type Cluster struct {
	ID           uuid.UUID `json:"id"`
	Title        string    `json:"title"`
	Category     string    `json:"category,omitempty"`
	ArticleCount int       `json:"article_count"`
	Confidence   float64   `json:"confidence"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
