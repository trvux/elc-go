-- WARNING: never run `migrate-down` for this migration against the shared
-- Supabase DATABASE_URL — it will drop the real production projects table
-- and its join tables. This exists only for tearing down a local/scratch
-- Postgres used for testing.
DROP TABLE IF EXISTS project_service;
DROP TABLE IF EXISTS project_category;
DROP TABLE IF EXISTS projects;
