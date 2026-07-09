-- Acis's "LUX" line is named directly in these 8 products' own `name` field
-- ("... DÒNG LUX") — no external research needed to establish the line is
-- real, it's self-declared in our own data. Confirmed via web research
-- 2026-07-09 that Acis (a Vietnamese smart-home brand, EASYCONTROL
-- platform) is real, but could NOT confirm a reliable mapping from these
-- Vietnamese product names to any external manufacturer model code (their
-- public site lists English model designations like "REC OLED"/"SSW REC"
-- with no clear 1:1 match to our catalog's Vietnamese names) — so mpn
-- stays as-is (currently falls back to sku, see the 2026-07-09 recovery),
-- NOT guessed here. Spans 3 categories (Bảng điều khiển/Công tắc/Cảm
-- biến), so category_id is left NULL like the AC hardware-group lines.
--
-- Run: psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f scripts/backfill-acis-lux-product-line.sql

BEGIN;

WITH acis AS (SELECT id FROM brands WHERE name = 'Acis' LIMIT 1)
INSERT INTO product_lines (brand_id, code, name, tier_rank, description, mpn_prefixes)
SELECT acis.id, 'LUX', 'Dòng LUX', 0,
       'Dòng thiết bị nhà thông minh có dây/tích hợp của Acis — tên "LUX" lấy trực tiếp từ tên sản phẩm gốc.',
       ARRAY[]::text[]
FROM acis
ON CONFLICT DO NOTHING;

UPDATE products p
SET product_line_id = pl.id
FROM brands b, product_lines pl
WHERE p.brand_id = b.id
  AND b.name = 'Acis'
  AND p.product_line_id IS NULL
  AND p.deleted_at IS NULL
  AND pl.brand_id = b.id
  AND pl.code = 'LUX'
  AND pl.deleted_at IS NULL
  AND p.name ILIKE '%DÒNG LUX%';

SELECT pl.code, pl.name, count(p.id) AS assigned_products
FROM product_lines pl
LEFT JOIN products p ON p.product_line_id = pl.id AND p.deleted_at IS NULL
WHERE pl.code = 'LUX'
GROUP BY pl.id, pl.code, pl.name;

COMMIT;
