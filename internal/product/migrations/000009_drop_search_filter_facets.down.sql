-- Reverses 000009: re-adds normalized_specs + search_vector (in the
-- post-000007 shape, indexing name + variant_mpns since sku no longer
-- exists on products) + their indexes, and the immutable_unaccent()
-- wrapper function.
ALTER TABLE products ADD COLUMN IF NOT EXISTS normalized_specs TEXT[] DEFAULT '{}';
CREATE INDEX IF NOT EXISTS products_normalized_specs_idx ON products USING gin (normalized_specs);

CREATE OR REPLACE FUNCTION immutable_unaccent(text) RETURNS text AS $$
    SELECT unaccent('unaccent', $1)
$$ LANGUAGE sql IMMUTABLE PARALLEL SAFE STRICT;

ALTER TABLE products ADD COLUMN IF NOT EXISTS search_vector tsvector
    GENERATED ALWAYS AS (
        to_tsvector('simple', immutable_unaccent(COALESCE(name, '') || ' ' || COALESCE(variant_mpns, '')))
    ) STORED;

CREATE INDEX IF NOT EXISTS products_search_vector_idx ON products USING gin (search_vector);
CREATE INDEX IF NOT EXISTS products_name_trgm_idx ON products USING gin (name gin_trgm_ops);
