-- WARNING: never run `migrate-down` for this migration against the shared
-- Supabase DATABASE_URL — it will drop the real production system_pages table.
DROP TABLE IF EXISTS system_pages;
