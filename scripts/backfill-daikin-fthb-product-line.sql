-- FTHB (4 products) was missed by both earlier Daikin backfill passes:
-- the wall-mount 1-way script's FTKB line only ever used mpn_prefixes
-- ['FTKB','ATKB'], and the remaining-categories script only handled
-- FTHF/FTXM/FTXV. FTHB completes the 2-way tier ladder as its own basic
-- tier (parallel to FTHF standard / FTXM+FTXV premium) — kept separate
-- from FTKB rather than merged, since these 4 products' own name field
-- says "một chiều" (1-way) while FTHB is Daikin's real 2-way ("H" =
-- heat-pump) code, a known pre-existing name/code mismatch flagged
-- earlier this session and deliberately NOT resolved here (out of scope
-- for a product-line backfill) — giving it its own line avoids baking
-- that disputed labeling into either the 1-way or 2-way family.
--
-- Run: psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f scripts/backfill-daikin-fthb-product-line.sql

BEGIN;

WITH daikin AS (SELECT id FROM brands WHERE name = 'Daikin' LIMIT 1),
cat AS (SELECT id FROM categories WHERE name = 'Máy lạnh treo tường' LIMIT 1)
INSERT INTO product_lines (brand_id, category_id, code, name, tier_rank, description, mpn_prefixes)
SELECT daikin.id, cat.id, 'FTHB', 'Dòng Inverter 2 Chiều Cơ Bản', 0,
       '2 chiều (nóng/lạnh) cơ bản nhất — lưu ý: tên sản phẩm hiện ghi "một chiều", có thể là lỗi dữ liệu cần rà lại (mã FTHB thực tế là dòng 2 chiều của Daikin).',
       ARRAY['FTHB']
FROM daikin, cat
ON CONFLICT DO NOTHING;

UPDATE products p
SET product_line_id = pl.id
FROM product_lines pl, product_variants v
WHERE p.product_line_id IS NULL
  AND p.deleted_at IS NULL
  AND pl.category_id = p.category_id
  AND pl.code = 'FTHB'
  AND pl.deleted_at IS NULL
  AND v.product_id = p.id AND v.is_default = true AND v.deleted_at IS NULL
  AND v.mpn ~ '^FTHB[0-9]';

SELECT pl.code, pl.name, count(p.id) AS assigned_products
FROM product_lines pl
LEFT JOIN products p ON p.product_line_id = pl.id AND p.deleted_at IS NULL
WHERE pl.code = 'FTHB'
GROUP BY pl.id, pl.code, pl.name;

COMMIT;
