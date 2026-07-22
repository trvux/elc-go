-- search_vector previously only covered name + variant_mpns (000015), so
-- chat search / regular full-text search couldn't find products by
-- marketing copy — e.g. "Humi Comfort chống khô da" (a feature only
-- mentioned in the rich-text `description` doc, never in the product
-- name). Extends search_vector to also index description's plain text
-- (extracted from its ProseMirror-style jsonb via a recursive jsonpath,
-- $.**.text pulls every "text" leaf regardless of nesting depth) and
-- `highlights` (Đặc điểm nổi bật, added by 000017).
--
-- Postgres generated columns require an IMMUTABLE expression; wrapping the
-- extraction in its own IMMUTABLE SQL function keeps the column definition
-- readable and mirrors immutable_unaccent's existing shape. jsonb_path_query_array
-- is itself IMMUTABLE (verified: pg_proc.provolatile = 'i'), so this is safe
-- to use in a STORED generated column.
--
-- Generated columns can't have their expression altered in place — must
-- drop and re-add (this also drops the dependent index, recreated after).
CREATE OR REPLACE FUNCTION product_description_text(doc jsonb) RETURNS text AS $$
    SELECT COALESCE(string_agg(DISTINCT value, ' '), '')
    FROM jsonb_array_elements_text(jsonb_path_query_array(doc, '$.**.text')) AS value
$$ LANGUAGE sql IMMUTABLE PARALLEL SAFE STRICT;

-- Built-in array_to_string is only STABLE (not IMMUTABLE, per pg_proc),
-- so it's rejected inside a generated column expression on its own —
-- same wrapper technique as product_description_text/immutable_unaccent.
CREATE OR REPLACE FUNCTION immutable_array_to_string(arr text[], sep text) RETURNS text AS $$
    SELECT array_to_string(arr, sep)
$$ LANGUAGE sql IMMUTABLE PARALLEL SAFE STRICT;

DROP INDEX IF EXISTS products_search_vector_idx;
ALTER TABLE products DROP COLUMN IF EXISTS search_vector;

ALTER TABLE products ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        to_tsvector('simple', immutable_unaccent(
            COALESCE(name, '') || ' ' ||
            COALESCE(variant_mpns, '') || ' ' ||
            COALESCE(immutable_array_to_string(highlights, ' '), '') || ' ' ||
            product_description_text(description)
        ))
    ) STORED;

CREATE INDEX products_search_vector_idx ON products USING gin (search_vector);
