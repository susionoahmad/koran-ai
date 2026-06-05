-- ============================================================================
-- MIGRATION UP: Koran AI Indonesia Database Schema
-- Target: PostgreSQL 18.x
-- ============================================================================

-- 0. Custom ENUM Types
CREATE TYPE edition_type AS ENUM ('PAGI', 'SIANG', 'MALAM');
CREATE TYPE edition_status AS ENUM ('DRAFT', 'PUBLISHED');
CREATE TYPE edition_section AS ENUM ('FRONT_PAGE', 'NASIONAL', 'EKONOMI', 'TEKNOLOGI', 'UMKM');
CREATE TYPE ai_job_type AS ENUM ('CLASSIFY', 'CLUSTER', 'SUMMARY', 'EDITION');
CREATE TYPE ai_job_status AS ENUM ('PENDING', 'PROCESSING', 'SUCCESS', 'FAILED');

-- 1. Table: sources
CREATE TABLE sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) UNIQUE NOT NULL,
    base_url VARCHAR(255) NOT NULL,
    rss_url VARCHAR(255) NOT NULL,
    source_type VARCHAR(50) NOT NULL CHECK (source_type IN ('rss', 'sitemap')),
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Index for searching active sources
CREATE INDEX idx_sources_active ON sources (is_active);

-- 2. Table: articles (PARTITIONED by RANGE on published_at)
CREATE TABLE articles (
    id UUID NOT NULL,
    source_id UUID NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL,
    url VARCHAR(512) NOT NULL,
    author VARCHAR(100),
    content TEXT NOT NULL,
    published_at TIMESTAMPTZ NOT NULL,
    scraped_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    hash_content VARCHAR(64) NOT NULL,
    image_url VARCHAR(512),
    processed BOOLEAN NOT NULL DEFAULT false,
    PRIMARY KEY (id, published_at),
    CONSTRAINT uniq_articles_url UNIQUE (url, published_at),
    CONSTRAINT uniq_articles_hash UNIQUE (hash_content, published_at)
) PARTITION BY RANGE (published_at);

-- Indeks pada parent table (PostgreSQL otomatis membuat indeks ini di setiap partisi baru)
CREATE INDEX idx_articles_published_at ON articles (published_at DESC);
CREATE INDEX idx_articles_unprocessed ON articles (processed) WHERE processed = false;

-- Pembuatan partisi untuk rentang waktu tahun 2026 (Contoh 3 bulan awal)
CREATE TABLE articles_y2026m06 PARTITION OF articles
    FOR VALUES FROM ('2026-06-01 00:00:00+00') TO ('2026-07-01 00:00:00+00');

CREATE TABLE articles_y2026m07 PARTITION OF articles
    FOR VALUES FROM ('2026-07-01 00:00:00+00') TO ('2026-08-01 00:00:00+00');

CREATE TABLE articles_y2026m08 PARTITION OF articles
    FOR VALUES FROM ('2026-08-01 00:00:00+00') TO ('2026-09-01 00:00:00+00');

-- Partisi Default untuk menangani data di luar range (PostgreSQL 10+)
CREATE TABLE articles_default PARTITION OF articles DEFAULT;


-- 3. Table: categories
CREATE TABLE categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) UNIQUE NOT NULL,
    slug VARCHAR(50) UNIQUE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Seed Data: Kategori Default
INSERT INTO categories (name, slug) VALUES
('Nasional', 'nasional'),
('Ekonomi', 'ekonomi'),
('Teknologi', 'teknologi'),
('UMKM', 'umkm'),
('Pendidikan', 'pendidikan'),
('Politik', 'politik'),
('Olahraga', 'olahraga'),
('Internasional', 'internasional');


-- 4. Table: clusters
CREATE TABLE clusters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category_id UUID NOT NULL REFERENCES categories(id) ON DELETE RESTRICT,
    title VARCHAR(255) NOT NULL,
    summary_short TEXT NOT NULL,
    article_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_clusters_category ON clusters (category_id);
CREATE INDEX idx_clusters_created_at ON clusters (created_at DESC);


-- 5. Table: cluster_articles (Junction Table referencing partitioned articles table)
CREATE TABLE cluster_articles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cluster_id UUID NOT NULL REFERENCES clusters(id) ON DELETE CASCADE,
    article_id UUID NOT NULL,
    article_published_at TIMESTAMPTZ NOT NULL,
    -- PostgreSQL mewajibkan foreign key ke tabel partisi merujuk ke UNIQUE composite key
    FOREIGN KEY (article_id, article_published_at) REFERENCES articles(id, published_at) ON DELETE CASCADE,
    CONSTRAINT uniq_cluster_articles_composite UNIQUE (cluster_id, article_id, article_published_at)
);

CREATE INDEX idx_cluster_articles_ref ON cluster_articles (article_id, article_published_at);


-- 6. Table: summaries
CREATE TABLE summaries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cluster_id UUID UNIQUE NOT NULL REFERENCES clusters(id) ON DELETE CASCADE,
    headline VARCHAR(255) NOT NULL,
    subheadline VARCHAR(255) NOT NULL,
    summary TEXT NOT NULL,
    impact TEXT NOT NULL,
    context TEXT NOT NULL,
    meta_description VARCHAR(255),
    search_vector tsvector,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indeks GIN untuk Full-Text Search pada judul dan konten ditulis ulang
CREATE INDEX idx_summaries_search_vector ON summaries USING gin (search_vector);

-- Trigger untuk update tsvector FTS secara otomatis
-- Menggunakan konfigurasi 'simple' (atau 'indonesian' jika modul kamus indonesia terinstall)
CREATE OR REPLACE FUNCTION summaries_search_vector_trigger() 
RETURNS trigger AS $$
BEGIN
  NEW.search_vector :=
    setweight(to_tsvector('simple', COALESCE(NEW.headline, '')), 'A') ||
    setweight(to_tsvector('simple', COALESCE(NEW.summary, '')), 'B');
  RETURN NEW;
END
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_summaries_search_vector_update
BEFORE INSERT OR UPDATE ON summaries
FOR EACH ROW EXECUTE FUNCTION summaries_search_vector_trigger();


-- 7. Table: editions
CREATE TABLE editions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    edition_date DATE NOT NULL,
    edition_type edition_type NOT NULL,
    title VARCHAR(150) NOT NULL,
    status edition_status NOT NULL DEFAULT 'DRAFT',
    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uniq_edition_date_type UNIQUE (edition_date, edition_type)
);

CREATE INDEX idx_editions_date_status ON editions (edition_date DESC, status);


-- 8. Table: edition_articles (Junction table)
CREATE TABLE edition_articles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    edition_id UUID NOT NULL REFERENCES editions(id) ON DELETE CASCADE,
    cluster_id UUID NOT NULL REFERENCES clusters(id) ON DELETE CASCADE,
    section edition_section NOT NULL,
    priority INTEGER NOT NULL DEFAULT 3,
    position INTEGER NOT NULL DEFAULT 1,
    CONSTRAINT uniq_edition_cluster UNIQUE (edition_id, cluster_id)
);

-- Indeks komposit untuk mempercepat sorting tata letak koran per edisi
CREATE INDEX idx_edition_articles_layout ON edition_articles (edition_id, section, position ASC);


-- 9. Table: crawl_logs
CREATE TABLE crawl_logs (
    id BIGSERIAL PRIMARY KEY,
    source_id UUID NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
    started_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ,
    status VARCHAR(20) NOT NULL CHECK (status IN ('RUNNING', 'SUCCESS', 'FAILED')),
    articles_found INTEGER NOT NULL DEFAULT 0,
    articles_saved INTEGER NOT NULL DEFAULT 0,
    error_message TEXT
);

CREATE INDEX idx_crawl_logs_source_status ON crawl_logs (source_id, status);
CREATE INDEX idx_crawl_logs_started ON crawl_logs (started_at DESC);


-- 10. Table: ai_jobs
CREATE TABLE ai_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_type ai_job_type NOT NULL,
    status ai_job_status NOT NULL DEFAULT 'PENDING',
    payload JSONB,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_ai_jobs_status_type ON ai_jobs (status, job_type);
CREATE INDEX idx_ai_jobs_created ON ai_jobs (created_at DESC);
