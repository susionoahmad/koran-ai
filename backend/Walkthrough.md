# Phase 5 News Clustering Engine

## Overview

Phase 5 adds a lightweight, non-AI clustering engine for grouping articles that discuss the same event or topic. It uses normalized title tokens, Jaccard similarity, and token overlap with a `0.70` threshold.

## Database

Migration `000005_news_clustering.up.sql` creates:

- `news_clusters`: cluster metadata, category, article count, confidence, timestamps.
- `cluster_articles`: article-to-cluster relation with indexes for cluster and article lookup.

The project currently uses a partitioned `articles` table whose primary key includes `published_at`. Because PostgreSQL cannot enforce a plain `FOREIGN KEY(article_id) REFERENCES articles(id)` against that shape, this migration enforces the cluster foreign key and indexes `article_id`; the repository updates the article row transactionally when adding a relation.

## Architecture

New module:

- `internal/clustering/entity`: `Cluster` and `ClusterArticle`.
- `internal/clustering/repository`: repository contract and PostgreSQL implementation.
- `internal/clustering/service`: clustering workflow, stats, and similarity engine.
- `internal/clustering/handler`: internal and public Fiber handlers.

Article repository extension:

- `ListClusterCandidates(ctx, limit)` returns `ai_processed = true` and `clustered = false`.
- `CountClusterCandidates(ctx)` supports clustering stats.

## Algorithm

1. Normalize title: lowercase, remove punctuation, strip Indonesian stopwords, collapse spaces.
2. Tokenize normalized title.
3. Compare candidate article title to existing cluster titles.
4. Score:

```text
score = (jaccard * 0.7) + (overlap * 0.3)
```

5. If `score >= 0.70`, join the best matching cluster.
6. Otherwise create a new cluster.

Cluster confidence is maintained as a rolling average of match scores. New single-article clusters start with confidence `1.0`.

## Locking

The service attempts to acquire Redis lock:

```text
clustering_worker_lock
```

TTL is five minutes. If Redis is nil or unavailable, clustering continues without a lock. If the lock already exists, the run returns a conflict through the internal endpoint.

## Endpoints

Internal:

- `POST /internal/clustering/run`
- `GET /internal/clustering/stats`

Public:

- `GET /api/v1/clusters`
- `GET /api/v1/clusters/:id`

## Validation

Executed from `backend`:

```bash
go build ./...
go test ./internal/clustering/... -cover
```

Coverage result:

```text
handler:    96.9%
repository: 90.3%
service:    91.7%
```

Additional `go test ./...` was run as a sanity check. It found an unrelated pre-existing crawler service test mismatch in `TestCrawlerService_QualityFilter`; Phase 5 packages passed.
