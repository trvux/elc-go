-- WARNING: never run `migrate-down` for this migration against the shared
-- Supabase DATABASE_URL — it will drop the real production service_groups
-- table (and, via ON DELETE CASCADE-less FK, orphan any services rows).
-- This exists only for tearing down a local/scratch Postgres used for testing.
DROP TABLE IF EXISTS service_groups;
