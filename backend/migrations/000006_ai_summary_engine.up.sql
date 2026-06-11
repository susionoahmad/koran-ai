-- Migration 000006: Phase 6 AI Summary Engine
-- Stores AI-generated summaries for news clusters.

CREATE TABLE IF NOT EXISTS summaries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cluster_id UUID NOT NULL,
    headline VARCHAR(500) NOT NULL,
    summary_short TEXT NOT NULL,
    summary_medium TEXT,
    summary_long TEXT,
    key_points JSONB,
    ai_model VARCHAR(100),
    ai_confidence NUMERIC(5,4) DEFAULT 0,
    generated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_summary_cluster
        FOREIGN KEY(cluster_id)
        REFERENCES news_clusters(id)
        ON DELETE CASCADE,

    CONSTRAINT uq_summary_cluster
    UNIQUE(cluster_id)
);

-- Ensure new columns are added if the table already existed
ALTER TABLE summaries
ADD COLUMN IF NOT EXISTS summary_short TEXT;

-- Migrate data from old "summary" column if it exists
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'summaries'
          AND column_name = 'summary'
    ) THEN
        UPDATE summaries
        SET summary_short = COALESCE(NULLIF(summary_short, ''), summary, headline)
        WHERE summary_short IS NULL OR summary_short = '';
    ELSE
        UPDATE summaries
        SET summary_short = COALESCE(NULLIF(summary_short, ''), headline)
        WHERE summary_short IS NULL OR summary_short = '';
    END IF;
END $$;

ALTER TABLE summaries
ALTER COLUMN summary_short SET NOT NULL;

ALTER TABLE summaries
ADD COLUMN IF NOT EXISTS summary_medium TEXT,
ADD COLUMN IF NOT EXISTS summary_long TEXT,
ADD COLUMN IF NOT EXISTS key_points JSONB,
ADD COLUMN IF NOT EXISTS ai_model VARCHAR(100),
ADD COLUMN IF NOT EXISTS ai_confidence NUMERIC(5,4) DEFAULT 0,
ADD COLUMN IF NOT EXISTS generated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- Drop old constraints and columns that are no longer used
-- Migration 1 had a foreign key constraint and unique index on cluster_id
ALTER TABLE summaries
DROP CONSTRAINT IF EXISTS summaries_cluster_id_fkey,
DROP CONSTRAINT IF EXISTS summaries_cluster_id_key;

ALTER TABLE summaries
DROP COLUMN IF EXISTS subheadline,
DROP COLUMN IF EXISTS summary,
DROP COLUMN IF EXISTS impact,
DROP COLUMN IF EXISTS context;

-- Update the search vector trigger function to use summary_short instead of summary
CREATE OR REPLACE FUNCTION summaries_search_vector_trigger() 
RETURNS trigger AS $$
BEGIN
  NEW.search_vector :=
    setweight(to_tsvector('simple', COALESCE(NEW.headline, '')), 'A') ||
    setweight(to_tsvector('simple', COALESCE(NEW.summary_short, '')), 'B');
  RETURN NEW;
END
$$ LANGUAGE plpgsql;

-- Ensure constraints are set up correctly on the existing table
DO $$
BEGIN
    -- Drop old fk constraint if it exists and recreate referencing news_clusters(id)
    IF EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'fk_summary_cluster'
          AND conrelid = 'summaries'::regclass
    ) THEN
        ALTER TABLE summaries DROP CONSTRAINT fk_summary_cluster;
    END IF;

    ALTER TABLE summaries
    ADD CONSTRAINT fk_summary_cluster
    FOREIGN KEY(cluster_id)
    REFERENCES news_clusters(id)
    ON DELETE CASCADE;

    -- Drop old unique constraint if it exists and recreate
    IF EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'uq_summary_cluster'
          AND conrelid = 'summaries'::regclass
    ) THEN
        ALTER TABLE summaries DROP CONSTRAINT uq_summary_cluster;
    END IF;

    ALTER TABLE summaries
    ADD CONSTRAINT uq_summary_cluster
    UNIQUE(cluster_id);
END $$;

CREATE INDEX IF NOT EXISTS idx_summaries_cluster
ON summaries(cluster_id);

CREATE INDEX IF NOT EXISTS idx_summaries_generated
ON summaries(generated_at DESC);
