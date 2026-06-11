-- Migration 000006: Rollback Phase 6 AI Summary Engine

-- Restore the search vector trigger function to use the old summary column
CREATE OR REPLACE FUNCTION summaries_search_vector_trigger() 
RETURNS trigger AS $$
BEGIN
  NEW.search_vector :=
    setweight(to_tsvector('simple', COALESCE(NEW.headline, '')), 'A') ||
    setweight(to_tsvector('simple', COALESCE(NEW.summary, '')), 'B');
  RETURN NEW;
END
$$ LANGUAGE plpgsql;

-- Re-add the old columns dropped during migration up
-- We add them as nullable first to avoid error on existing rows
ALTER TABLE summaries
ADD COLUMN IF NOT EXISTS subheadline VARCHAR(255),
ADD COLUMN IF NOT EXISTS summary TEXT,
ADD COLUMN IF NOT EXISTS impact TEXT,
ADD COLUMN IF NOT EXISTS context TEXT;

-- If summary is null, copy from summary_short
UPDATE summaries
SET summary = COALESCE(summary, summary_short),
    subheadline = COALESCE(subheadline, headline),
    impact = COALESCE(impact, ''),
    context = COALESCE(context, '')
WHERE summary IS NULL;

-- Remove constraints and indexes
ALTER TABLE summaries
DROP CONSTRAINT IF EXISTS fk_summary_cluster,
DROP CONSTRAINT IF EXISTS uq_summary_cluster;

DROP INDEX IF EXISTS idx_summaries_generated;
DROP INDEX IF EXISTS idx_summaries_cluster;

-- Drop the new columns added during migration up
ALTER TABLE summaries
DROP COLUMN IF EXISTS generated_at,
DROP COLUMN IF EXISTS ai_confidence,
DROP COLUMN IF EXISTS ai_model,
DROP COLUMN IF EXISTS key_points,
DROP COLUMN IF EXISTS summary_long,
DROP COLUMN IF EXISTS summary_medium,
DROP COLUMN IF EXISTS summary_short;
