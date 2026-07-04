-- WARNING: never run `migrate-down` for this migration against the shared
-- Supabase DATABASE_URL in a way that assumes it drops the products table —
-- it deliberately does NOT. products is a shared, pre-existing table (also
-- referenced by brand/category), so this only reverses what THIS migration
-- actually added: the search_vector/normalized_specs columns and their
-- indexes. It exists only for tearing down a local/scratch Postgres used for
-- testing (where dropping the whole table would also be reasonable, but
-- staying symmetrical with the .up.sql's additive-only real-DB behavior is
-- simpler to reason about).
DROP INDEX IF EXISTS products_name_trgm_idx;
DROP INDEX IF EXISTS products_search_vector_idx;
ALTER TABLE products DROP COLUMN IF EXISTS search_vector;
DROP FUNCTION IF EXISTS immutable_unaccent(text);
DROP INDEX IF EXISTS products_normalized_specs_idx;
ALTER TABLE products DROP COLUMN IF EXISTS normalized_specs;
