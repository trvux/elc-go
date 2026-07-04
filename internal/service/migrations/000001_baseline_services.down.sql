-- WARNING: never run `migrate-down` for this migration against the shared
-- Supabase DATABASE_URL — it will drop the real production services table
-- (and cascade-delete project_service rows). Local/scratch Postgres only.
DROP TABLE IF EXISTS services;
