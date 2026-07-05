-- WARNING: never run `migrate-down` for this migration against the shared
-- Supabase DATABASE_URL — it will drop the real production categories table.
DROP TABLE IF EXISTS categories;
