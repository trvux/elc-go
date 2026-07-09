-- Removes Product's own sku/mpn/gtin/price/stock columns — every product now
-- has exactly the same shape as Shopify/Medusa's model: Product is a pure
-- container, and EVERY sellable identity (sku/mpn/gtin/price/stock) lives
-- exclusively on product_variants, never duplicated onto products. Before
-- this migration, a product with zero variant rows fell back to reading
-- these columns directly (see the old ProductGeneralTab.tsx "hasVariants"
-- branch) — a "boss variant on Product + optional follower variants"
-- hybrid, inconsistent with the variant model built in migration 000005.
-- Safe: 179/179 current products already have a real default variant (see
-- cmd/backfill-product-variants), so no data migration is needed beyond the
-- variant_mpns backfill below.
--
-- search_vector previously indexed name+sku. sku is internal-only and not
-- what customers search (see docs/product-v2-design.md) — replaced with
-- variant_mpns, a denormalized cache of the product's active variants' MPNs
-- (the manufacturer codes customers actually search), maintained by
-- RecomputeDisplayCache alongside display_price/display_stock_status —
-- same "compute once at write time, read via a plain column" pattern
-- already used there, avoiding a join from products to product_variants on
-- every search query.

ALTER TABLE products ADD COLUMN variant_mpns text NOT NULL DEFAULT '';

UPDATE products p
SET variant_mpns = sub.mpns
FROM (
    SELECT product_id, string_agg(mpn, ' ' ORDER BY mpn) AS mpns
    FROM product_variants
    WHERE deleted_at IS NULL AND is_active = true
    GROUP BY product_id
) sub
WHERE sub.product_id = p.id;

DROP INDEX IF EXISTS products_search_vector_idx;
ALTER TABLE products DROP COLUMN search_vector;
ALTER TABLE products ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        to_tsvector('simple', immutable_unaccent(COALESCE(name, '') || ' ' || COALESCE(variant_mpns, '')))
    ) STORED;
CREATE INDEX products_search_vector_idx ON products USING gin (search_vector);

ALTER TABLE products DROP CONSTRAINT IF EXISTS products_sku_key;
ALTER TABLE products DROP COLUMN sku;
ALTER TABLE products DROP COLUMN mpn;
ALTER TABLE products DROP COLUMN gtin;
ALTER TABLE products DROP COLUMN original_price;
ALTER TABLE products DROP COLUMN sale_price;
ALTER TABLE products DROP COLUMN discount_percent;
ALTER TABLE products DROP COLUMN stock_status;
