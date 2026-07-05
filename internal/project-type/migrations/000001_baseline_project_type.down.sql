-- WARNING: never run `migrate-down` for this migration against the shared
-- Supabase DATABASE_URL — it will drop the real production project_type
-- tables.
DROP TABLE IF EXISTS project_type_category;
DROP TABLE IF EXISTS project_type;
