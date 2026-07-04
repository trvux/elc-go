-- WARNING: never run `migrate-down` for this migration against the shared
-- Supabase DATABASE_URL — it will drop the real production brands table
-- (and, via the products FK, orphan/null out any products.brand_id rows).
-- This exists only for tearing down a local/scratch Postgres used for testing.
DROP TABLE IF EXISTS brands;
