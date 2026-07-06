-- WARNING: never run `migrate-down` for this migration against the shared
-- Supabase DATABASE_URL — it will drop the real production tracking_events
-- table.
DROP TABLE IF EXISTS tracking_events;
