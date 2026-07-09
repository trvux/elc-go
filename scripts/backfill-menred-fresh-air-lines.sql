-- Remaining Menred "Máy cấp khí tươi, lọc không khí" products, confirmed
-- via web research 2026-07-09:
--   - Smart O2: EXPLICITLY confirmed as one real Menred series spanning
--     S1 (compact wall-mount, Red Dot 2018), G3/G5/G7 (ERV, G5 is the
--     floor-standing cabinet variant) — high confidence, direct source
--     confirmation, not just naming-pattern inference.
--   - N5 (N5.150A/N5.250A), R (R150/R250/R350), P (P5/P7): same
--     "letter + capacity number" progression already established for the
--     NET line (e.g. our own site's listing shows P5=450m3/h,
--     S5=400m3/h airflow) — grouped by that pattern, same reasoning as
--     Daikin's capacity-only siblings.
-- Deliberately NOT grouped (insufficient evidence): S5 (only 1 product,
-- no sibling despite sharing the "...CLS4.0E" suffix with P/R — that
-- suffix looks like a shared *controller generation*, not a shared
-- physical product line, so grouping across P/R/S by it would be
-- misleading for the capacity-switcher), G2 (a dehumidifier per earlier
-- research, NOT part of Smart O2 despite the shared "G" letter),
-- HGS-90 PRO, Q6, NEW5.350, HUM35 (a humidifier *module*/accessory, not
-- a standalone unit).
--
-- Run: psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f scripts/backfill-menred-fresh-air-lines.sql

BEGIN;

WITH menred AS (SELECT id FROM brands WHERE name = 'Menred' LIMIT 1),
cat AS (SELECT id FROM categories WHERE name = 'Máy cấp khí tươi, lọc không khí' LIMIT 1)
INSERT INTO product_lines (brand_id, category_id, code, name, tier_rank, description, mpn_prefixes)
SELECT menred.id, cat.id, v.code, v.name, 0, v.description, v.mpn_prefixes
FROM menred, cat, (VALUES
    ('SMARTO2', 'Dòng Smart O2', 'Máy cấp khí tươi Smart O2 — gồm S1 (gắn tường, giải Red Dot 2018), G3/G5/G7 (ERV, G5 là bản tủ đứng).', ARRAY['O2 S1','Smart O2']),
    ('N5', 'Dòng N5', 'Chỉ khác nhau lưu lượng gió (150A/250A).', ARRAY['N5.']),
    ('R', 'Dòng R', 'Chỉ khác nhau lưu lượng gió (150/250/350).', ARRAY['R150','R250','R350']),
    ('P', 'Dòng P', 'Chỉ khác nhau lưu lượng gió (P5=450m3/h, P7 cao hơn).', ARRAY['P5','P7'])
) AS v(code, name, description, mpn_prefixes)
ON CONFLICT DO NOTHING;

UPDATE products p
SET product_line_id = pl.id
FROM product_lines pl, product_variants v
WHERE p.product_line_id IS NULL
  AND p.deleted_at IS NULL
  AND pl.category_id = p.category_id
  AND pl.code IN ('SMARTO2','N5','R','P')
  AND pl.deleted_at IS NULL
  AND v.product_id = p.id AND v.is_default = true AND v.deleted_at IS NULL
  AND (
    (pl.code = 'SMARTO2' AND (v.mpn ILIKE 'O2 S1' OR v.mpn ILIKE 'Smart O2%' OR p.name ILIKE '%Smart O2%'))
    OR (pl.code = 'N5' AND v.mpn ILIKE 'N5.%')
    OR (pl.code = 'R' AND v.mpn ~ '^R[0-9]')
    OR (pl.code = 'P' AND v.mpn ~ '^P[0-9]')
  );

SELECT pl.code, pl.name, count(p.id) AS assigned_products
FROM product_lines pl
JOIN brands b ON b.id = pl.brand_id
LEFT JOIN products p ON p.product_line_id = pl.id AND p.deleted_at IS NULL
WHERE b.name = 'Menred' AND pl.code IN ('SMARTO2','N5','R','P')
GROUP BY pl.id, pl.code, pl.name;

COMMIT;
