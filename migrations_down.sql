-- ============================================================================
-- MIGRATION DOWN: Koran AI Indonesia Database Schema Cleanup
-- Target: PostgreSQL 18.x
-- ============================================================================

-- 1. Drop Triggers and Functions
DROP TRIGGER IF EXISTS trg_summaries_search_vector_update ON summaries;
DROP FUNCTION IF EXISTS summaries_search_vector_trigger();

-- 2. Drop Tables (In reverse order of creation to satisfy foreign key dependencies)
DROP TABLE IF EXISTS ai_jobs;
DROP TABLE IF EXISTS crawl_logs;
DROP TABLE IF EXISTS edition_articles;
DROP TABLE IF EXISTS editions;
DROP TABLE IF EXISTS summaries;
DROP TABLE IF EXISTS cluster_articles;
DROP TABLE IF EXISTS clusters;
DROP TABLE IF EXISTS categories;
DROP TABLE IF EXISTS articles; -- Menghapus table partitioned ini otomatis menghapus seluruh partisinya.
DROP TABLE IF EXISTS sources;

-- 3. Drop Custom ENUM Types
DROP TYPE IF EXISTS ai_job_status;
DROP TYPE IF EXISTS ai_job_type;
DROP TYPE IF EXISTS edition_section;
DROP TYPE IF EXISTS edition_status;
DROP TYPE IF EXISTS edition_type;
