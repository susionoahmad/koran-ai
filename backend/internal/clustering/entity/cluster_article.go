package entity

import "github.com/google/uuid"

// ClusterArticle links an article to a news cluster.
type ClusterArticle struct {
	ClusterID uuid.UUID `json:"cluster_id"`
	ArticleID uuid.UUID `json:"article_id"`
}
