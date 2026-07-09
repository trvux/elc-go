-- Full reconciliation: every product's old flat sku/mpn/gtin/price/stock
-- (from the pre-incident backup, loaded into a scratch table by the
-- deploy-time recovery step) vs the new product_variants default row.
-- Read-only — reports mismatches, does not fix anything.
--
-- Requires products_recovery_scratch to already be loaded (see
-- scripts/deploy-attribute-migration.sh's sibling recovery step, or
-- re-extract manually: pg_restore -t products --data-only from a backup
-- dump into this same scratch shape before running this file).

-- 1) Row count parity.
SELECT
  (SELECT count(*) FROM products_recovery_scratch WHERE deleted_at IS NULL) AS old_count,
  (SELECT count(*) FROM products WHERE deleted_at IS NULL) AS live_count;

-- 2) Old products with NO matching live product at all (would mean a hard
--    loss, not expected).
SELECT s.id, s.name
FROM products_recovery_scratch s
WHERE s.deleted_at IS NULL
  AND NOT EXISTS (SELECT 1 FROM products p WHERE p.id = s.id AND p.deleted_at IS NULL);

-- 3) Live products with zero variants (broken state — every product must
--    have >= 1).
SELECT p.id, p.name
FROM products p
WHERE p.deleted_at IS NULL
  AND NOT EXISTS (SELECT 1 FROM product_variants v WHERE v.product_id = p.id AND v.deleted_at IS NULL);

-- 4) Field-by-field mismatch between old flat data and the new default
--    variant. mpn is compared with the known "-dupN" suffix stripped
--    (pre-existing duplicate-mpn products, see the 2026-07-09 recovery —
--    not a migration bug). sale_price/original_price both go through
--    NULLIF(x,0) since the app treats 0 and NULL as equivalent "no sale
--    price"/defaults.
WITH old_data AS (
  SELECT
    id AS product_id,
    COALESCE(NULLIF(mpn, ''), sku) AS expected_mpn,
    sku AS expected_sku,
    NULLIF(gtin, '') AS expected_gtin,
    COALESCE(original_price, 0) AS expected_original_price,
    NULLIF(sale_price, 0) AS expected_sale_price,
    COALESCE(discount_percent, 0) AS expected_discount_percent,
    stock_status AS expected_stock_status
  FROM products_recovery_scratch
  WHERE deleted_at IS NULL
), new_data AS (
  SELECT product_id, mpn, sku, gtin, original_price, sale_price, discount_percent, stock_status::text AS stock_status
  FROM product_variants
  WHERE deleted_at IS NULL AND is_default = true
)
SELECT
  o.product_id,
  p.name,
  CASE WHEN regexp_replace(n.mpn, '-dup[0-9]+$', '') <> o.expected_mpn THEN
    format('mpn: expected %s, got %s', o.expected_mpn, n.mpn) END AS mpn_diff,
  CASE WHEN n.sku IS DISTINCT FROM o.expected_sku THEN
    format('sku: expected %s, got %s', o.expected_sku, n.sku) END AS sku_diff,
  CASE WHEN n.gtin IS DISTINCT FROM o.expected_gtin THEN
    format('gtin: expected %s, got %s', o.expected_gtin, n.gtin) END AS gtin_diff,
  CASE WHEN n.original_price IS DISTINCT FROM o.expected_original_price THEN
    format('original_price: expected %s, got %s', o.expected_original_price, n.original_price) END AS price_diff,
  CASE WHEN n.sale_price IS DISTINCT FROM o.expected_sale_price THEN
    format('sale_price: expected %s, got %s', o.expected_sale_price, n.sale_price) END AS sale_diff,
  CASE WHEN n.discount_percent IS DISTINCT FROM o.expected_discount_percent THEN
    format('discount_percent: expected %s, got %s', o.expected_discount_percent, n.discount_percent) END AS discount_diff,
  CASE WHEN n.stock_status IS DISTINCT FROM o.expected_stock_status THEN
    format('stock_status: expected %s, got %s', o.expected_stock_status, n.stock_status) END AS stock_diff
FROM old_data o
JOIN products p ON p.id = o.product_id
LEFT JOIN new_data n ON n.product_id = o.product_id
WHERE n.product_id IS NULL
   OR regexp_replace(n.mpn, '-dup[0-9]+$', '') <> o.expected_mpn
   OR n.sku IS DISTINCT FROM o.expected_sku
   OR n.gtin IS DISTINCT FROM o.expected_gtin
   OR n.original_price IS DISTINCT FROM o.expected_original_price
   OR n.sale_price IS DISTINCT FROM o.expected_sale_price
   OR n.discount_percent IS DISTINCT FROM o.expected_discount_percent
   OR n.stock_status IS DISTINCT FROM o.expected_stock_status
ORDER BY p.name;

-- 5) specs preservation — the legacy jsonb column must be byte-identical
--    to the pre-migration backup, since nothing in this migration ever
--    writes to products.specs, only reads it.
SELECT count(*) AS specs_mismatch_count
FROM products_recovery_scratch s
JOIN products p ON p.id = s.id AND p.deleted_at IS NULL
WHERE s.deleted_at IS NULL AND s.specs IS DISTINCT FROM p.specs;
