-- MENRED's "NET" heat-recovery ventilator series (NET.800 through NET.8000)
-- is a genuine capacity-only progression confirmed against menred.com and
-- reseller spec sheets 2026-07-09 (NET.800 = 600-800 m3/h, NET.1000 =
-- 600-1000 m3/h, etc.) — same shape as Daikin's capacity-only siblings
-- (see backfill-daikin-product-lines.sql), just a different brand/category.
-- Every other Menred fresh-air model (P5, S5, R150/250/350, N5.150A/250A,
-- G2/G5/G7, O2 S1/G3, Q6, HGS-90) was deliberately left OUT — quick web
-- research turned up conflicting evidence (e.g. G2 is a dehumidifier, G4 is
-- an unrelated ERV — not a consistent numbered series), not enough
-- confidence to safely group without risking a misleading cross-link.
--
-- Run: psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f scripts/backfill-menred-net-product-line.sql

BEGIN;

WITH menred AS (
    SELECT id FROM brands WHERE name = 'Menred' LIMIT 1
)
INSERT INTO product_lines (brand_id, code, name, tier_rank, description, mpn_prefixes)
SELECT menred.id, 'NET', 'Dòng NET (Heat Recovery Ventilator)', 0,
       'Máy cấp khí tươi/lọc không khí thu hồi nhiệt, chỉ khác nhau lưu lượng gió (m3/h) — NET.800 đến NET.8000.',
       ARRAY['NET']
FROM menred
ON CONFLICT DO NOTHING;

UPDATE products p
SET product_line_id = pl.id
FROM brands b, product_lines pl, product_variants v
WHERE p.brand_id = b.id
  AND b.name = 'Menred'
  AND p.product_line_id IS NULL
  AND p.deleted_at IS NULL
  AND pl.brand_id = b.id
  AND pl.code = 'NET'
  AND pl.deleted_at IS NULL
  AND v.product_id = p.id AND v.is_default = true AND v.deleted_at IS NULL
  AND v.mpn ILIKE 'NET%';

SELECT pl.code, pl.name, count(p.id) AS assigned_products
FROM product_lines pl
LEFT JOIN products p ON p.product_line_id = pl.id AND p.deleted_at IS NULL
WHERE pl.code = 'NET'
GROUP BY pl.id, pl.code, pl.name;

COMMIT;
