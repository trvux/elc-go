DROP INDEX IF EXISTS products_search_vector_idx;
ALTER TABLE products DROP COLUMN IF EXISTS search_vector;

ALTER TABLE products ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        to_tsvector('simple', immutable_unaccent(COALESCE(name, '') || ' ' || COALESCE(variant_mpns, '')))
    ) STORED;

CREATE INDEX products_search_vector_idx ON products USING gin (search_vector);

DROP FUNCTION IF EXISTS product_description_text(jsonb);
DROP FUNCTION IF EXISTS immutable_array_to_string(text[], text);
