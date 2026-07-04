-- WARNING: never run `migrate-down` for this migration against the shared
-- Supabase DATABASE_URL — it will drop the real production contacts table.
-- This down migration exists only for tearing down a local/scratch Postgres
-- used for testing.
DROP TABLE IF EXISTS contacts;
