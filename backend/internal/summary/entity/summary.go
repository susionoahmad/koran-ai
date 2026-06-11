package entity

import (
	"time"

	"github.com/google/uuid"
)

// Summary represents an AI-generated neutral news summary for a cluster.
type Summary struct {
	ID            uuid.UUID `json:"id"`
	ClusterID     uuid.UUID `json:"cluster_id"`
	Headline      string    `json:"headline"`
	SummaryShort  string    `json:"summary_short"`
	SummaryMedium string    `json:"summary_medium,omitempty"`
	SummaryLong   string    `json:"summary_long,omitempty"`
	KeyPoints     []string  `json:"key_points,omitempty"`
	AIModel       string    `json:"ai_model,omitempty"`
	AIConfidence  float64   `json:"ai_confidence"`
	GeneratedAt   time.Time `json:"generated_at"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
