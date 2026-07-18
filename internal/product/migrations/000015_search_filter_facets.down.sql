-- Reverses 000015. specs is re-added empty (its data is not recoverable —
-- same "down doesn't restore dropped data" convention as every other
-- destructive migration in this module, e.g. 000009/000010/000014).
DROP INDEX IF EXISTS products_facet_tokens_idx;
ALTER TABLE products DROP COLUMN IF EXISTS facet_tokens;

DROP INDEX IF EXISTS products_name_trgm_idx;
DROP INDEX IF EXISTS products_search_vector_idx;
ALTER TABLE products DROP COLUMN IF EXISTS search_vector;
DROP FUNCTION IF EXISTS immutable_unaccent(text);

ALTER TABLE products ADD COLUMN IF NOT EXISTS specs jsonb NOT NULL DEFAULT '{}';
