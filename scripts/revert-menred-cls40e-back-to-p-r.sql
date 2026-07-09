-- Reverts fix-menred-cls40e-merge.sql. Further research showed "CLS4.0"/
-- "CLS4.0E" is Menred's shared LCD control-panel/display component name
-- (confirmed: "CLS4.0 control screen showing PM2.5 and CO2 concentration
-- charts"), used across MULTIPLE physically different Menred product
-- families — not itself a product-line identifier. Grouping P (wall/other
-- form factor?) with R (likely a different installation type) just
-- because they share a display panel risks the same mistake in the
-- opposite direction: misleadingly cross-linking two different physical
-- products in the capacity switcher. Every other brand's naming audited
-- today (Daikin FTKB/FBA/FCF/etc.) consistently uses a different leading
-- letter for a different physical design — no evidence found that Menred
-- breaks that pattern here. Reverting to separate P (P5/P7) and R
-- (R150/R250/R350) lines; S5 (single product, no same-letter sibling)
-- stays deliberately unmapped, same as before the merge.
--
-- Run: psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f scripts/revert-menred-cls40e-back-to-p-r.sql

BEGIN;

WITH menred AS (SELECT id FROM brands WHERE name = 'Menred' LIMIT 1),
cat AS (SELECT id FROM categories WHERE name = 'Máy cấp khí tươi, lọc không khí' LIMIT 1)
INSERT INTO product_lines (brand_id, category_id, code, name, tier_rank, description, mpn_prefixes)
SELECT menred.id, cat.id, v.code, v.name, 0, v.description, v.mpn_prefixes
FROM menred, cat, (VALUES
    ('P', 'Dòng P', 'Chỉ khác nhau lưu lượng gió (P5=450m3/h, P7 cao hơn).', ARRAY['P5','P7']),
    ('R', 'Dòng R', 'Chỉ khác nhau lưu lượng gió (150/250/350).', ARRAY['R150','R250','R350'])
) AS v(code, name, description, mpn_prefixes)
ON CONFLICT DO NOTHING;

UPDATE products p
SET product_line_id = new_pl.id
FROM product_lines new_pl, product_variants v
WHERE v.product_id = p.id AND v.is_default = true AND v.deleted_at IS NULL
  AND p.deleted_at IS NULL
  AND p.product_line_id IN (SELECT id FROM product_lines WHERE code = 'CLS40E')
  AND (
    (new_pl.code = 'P' AND v.mpn ~ '^P[0-9]')
    OR (new_pl.code = 'R' AND v.mpn ~ '^R[0-9]')
  );

-- S5 goes back to unmapped (was pulled into CLS40E by name match, not
-- mpn prefix, so it's not caught by the UPDATE above).
UPDATE products p
SET product_line_id = NULL
WHERE p.deleted_at IS NULL
  AND p.product_line_id IN (SELECT id FROM product_lines WHERE code = 'CLS40E');

DELETE FROM product_lines WHERE code = 'CLS40E';

SELECT pl.code, pl.name, count(p.id) AS assigned_products
FROM product_lines pl
JOIN brands b ON b.id = pl.brand_id
LEFT JOIN products p ON p.product_line_id = pl.id AND p.deleted_at IS NULL
WHERE b.name = 'Menred' AND pl.code IN ('P','R')
GROUP BY pl.id, pl.code, pl.name;

COMMIT;
