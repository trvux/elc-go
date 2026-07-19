-- Rebuilds search/filter/facet on top of the new structured attribute
-- system (attribute_definitions/product_attribute_values), replacing the
-- free-text mechanism torn down by 000009. search_vector/immutable_unaccent
-- are the exact same proven shape 000009 already knew how to re-add (see its
-- down.sql) — name + variant_mpns full-text, unaccent-aware. facet_tokens is
-- new: a denormalized array of "attribute_code:value" tokens for
-- select/multiselect/boolean attribute values, recomputed at write time by
-- RecomputeFacetTokens (internal/product/infrastructure) whenever a
-- product's attribute values change — mirrors the exact "compute once at
-- write time, read via a plain column" technique already used for
-- display_price/price_min/price_max/variant_mpns (RecomputeDisplayCache).
-- Number-type attributes (BTU, HP, ...) are intentionally NOT tokenized —
-- they're range-filterable, not discrete/equality-facetable, so they're
-- queried directly against product_attribute_values.value_number instead.
--
-- Also drops `specs` (jsonb, default '{}') — leftover from the original
-- 000001 baseline table, never migrated away by any later migration, and
-- confirmed dead: zero Go code in internal/product references it anymore
-- (superseded entirely by the structured attribute system). Cheap to clean
-- up while this migration is already touching the table.
CREATE EXTENSION IF NOT EXISTS unaccent;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

ALTER TABLE products DROP COLUMN IF EXISTS specs;

CREATE OR REPLACE FUNCTION immutable_unaccent(text) RETURNS text AS $$
    SELECT unaccent('unaccent', $1)
$$ LANGUAGE sql IMMUTABLE PARALLEL SAFE STRICT;

ALTER TABLE products ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        to_tsvector('simple', immutable_unaccent(COALESCE(name, '') || ' ' || COALESCE(variant_mpns, '')))
    ) STORED;

CREATE INDEX products_search_vector_idx ON products USING gin (search_vector);
CREATE INDEX products_name_trgm_idx ON products USING gin (name gin_trgm_ops);

ALTER TABLE products ADD COLUMN facet_tokens text[] NOT NULL DEFAULT '{}';

-- Backfill facet_tokens for every product that already has attribute values
-- (RecomputeFacetTokens, added alongside this migration, only fires on
-- future Create/Update — existing rows need this one-time bulk pass, same
-- CASE/unnest logic).
WITH tokens AS (
	SELECT pav.product_id, ad.code || ':' || opt AS token
	FROM product_attribute_values pav
	JOIN attribute_definitions ad ON ad.id = pav.attribute_definition_id
	CROSS JOIN LATERAL unnest(
		CASE ad.data_type
			WHEN 'select' THEN ARRAY[pav.value_text]
			WHEN 'multiselect' THEN pav.value_options
			WHEN 'boolean' THEN ARRAY[pav.value_boolean::text]
			ELSE ARRAY[]::text[]
		END
	) AS opt
	WHERE pav.deleted_at IS NULL AND opt IS NOT NULL
), agg AS (
	SELECT product_id, array_agg(token) AS tokens FROM tokens GROUP BY product_id
)
UPDATE products SET facet_tokens = agg.tokens FROM agg WHERE products.id = agg.product_id;

CREATE INDEX products_facet_tokens_idx ON products USING gin (facet_tokens);
