-- Best-effort revert: restores the columns and backfills them from each
-- product's default variant. Lossy for products with multiple variants
-- (only the default variant's values survive) — acceptable for a down
-- migration, which is a structural rollback path, not a guaranteed
-- data-preserving one.

DROP INDEX IF EXISTS products_search_vector_idx;
ALTER TABLE products DROP COLUMN search_vector;

ALTER TABLE products ADD COLUMN sku text;
ALTER TABLE products ADD COLUMN mpn text;
ALTER TABLE products ADD COLUMN gtin text;
ALTER TABLE products ADD COLUMN original_price numeric(15,0) DEFAULT 0;
ALTER TABLE products ADD COLUMN sale_price numeric(15,0);
ALTER TABLE products ADD COLUMN discount_percent numeric(5,2) DEFAULT 0;
ALTER TABLE products ADD COLUMN stock_status text DEFAULT 'in_stock';

UPDATE products p
SET sku = COALESCE(dv.sku, p.id::text),
    mpn = dv.mpn,
    gtin = dv.gtin,
    original_price = COALESCE(dv.original_price, 0),
    sale_price = dv.sale_price,
    stock_status = COALESCE(dv.stock_status::text, 'in_stock')
FROM product_variants dv
WHERE dv.product_id = p.id AND dv.is_default = true AND dv.deleted_at IS NULL;

ALTER TABLE products ALTER COLUMN sku SET NOT NULL;
ALTER TABLE products ADD CONSTRAINT products_sku_key UNIQUE (sku);

ALTER TABLE products ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        to_tsvector('simple', immutable_unaccent(COALESCE(name, '') || ' ' || COALESCE(sku, '')))
    ) STORED;
CREATE INDEX products_search_vector_idx ON products USING gin (search_vector);

ALTER TABLE products DROP COLUMN variant_mpns;
