-- Removes the facet/search machinery added by migration 000001 (and
-- rebuilt by 000007) — the product model is being stripped down to bare
-- CRUD ahead of a redesign around category-scoped attribute sets. Does NOT
-- touch product_attribute_values/attribute_definitions (migration 000008 /
-- internal/attribute) — that's a separate, kept CRUD system, not part of
-- this facet/search teardown. variant_mpns is intentionally left in place:
-- it's woven into RecomputeDisplayCache (internal/product/infrastructure/
-- variant_repository.go) alongside display_price/display_stock_status,
-- which stays untouched CRUD infra; revisit separately if it should go too.
DROP INDEX IF EXISTS products_name_trgm_idx;
DROP INDEX IF EXISTS products_search_vector_idx;
ALTER TABLE products DROP COLUMN IF EXISTS search_vector;
DROP FUNCTION IF EXISTS immutable_unaccent(text);
DROP INDEX IF EXISTS products_normalized_specs_idx;
ALTER TABLE products DROP COLUMN IF EXISTS normalized_specs;
