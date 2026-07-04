-- WARNING: never run `migrate-down` for this migration against the shared
-- Supabase DATABASE_URL — it will drop the real production group_categories table.
DROP TABLE IF EXISTS group_categories;
