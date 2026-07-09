-- Mirrors scripts/backfill-daikin-product-lines.sql for LG's 13 wall-mounted
-- units — confirmed 4-tier pattern via consistent price ordering at every
-- capacity (docs/catalog-audit-2026-07-08.md finding 3): IEC < IDC < IDH < IPC.
-- Requires migration 000006 (product_lines.mpn_prefixes).
-- Safe to re-run (idempotent insert, UPDATE only touches NULL product_line_id).
--
--   psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f scripts/backfill-lg-product-lines.sql

BEGIN;

WITH lg AS (
    SELECT id FROM brands WHERE name = 'LG' LIMIT 1
)
INSERT INTO product_lines (brand_id, code, name, tier_rank, description, mpn_prefixes)
SELECT lg.id, v.code, v.name, v.tier_rank, v.description, v.mpn_prefixes
FROM lg, (VALUES
    ('IEC', 'Dòng Tiết Kiệm (Economy)', 1, 'Giá rẻ nhất trong các dòng máy lạnh treo tường LG.', ARRAY['IEC']),
    ('IDC', 'Dòng Tiêu Chuẩn (Standard)', 2, 'Inverter tiêu chuẩn.', ARRAY['IDC']),
    ('IDH', 'Dòng Nâng Cao (Higher)', 3, 'Cấp cao hơn dòng Tiêu Chuẩn.', ARRAY['IDH']),
    ('IPC', 'Dòng Cao Cấp (Premium)', 4, 'Cao cấp nhất trong các dòng máy lạnh treo tường LG.', ARRAY['IPC'])
) AS v(code, name, tier_rank, description, mpn_prefixes)
ON CONFLICT DO NOTHING;

UPDATE products p
SET product_line_id = pl.id
FROM brands b, product_lines pl
WHERE p.brand_id = b.id
  AND b.name = 'LG'
  AND p.product_line_id IS NULL
  AND p.deleted_at IS NULL
  AND pl.brand_id = b.id
  AND pl.deleted_at IS NULL
  AND EXISTS (
    SELECT 1 FROM product_variants v, unnest(pl.mpn_prefixes) prefix
    WHERE v.product_id = p.id AND v.is_default = true AND v.deleted_at IS NULL
      AND v.mpn LIKE prefix || '%'
  );

SELECT pl.code, pl.name, count(p.id) AS assigned_products
FROM product_lines pl
JOIN brands b ON b.id = pl.brand_id AND b.name = 'LG'
LEFT JOIN products p ON p.product_line_id = pl.id AND p.deleted_at IS NULL
GROUP BY pl.id, pl.code, pl.name, pl.tier_rank
ORDER BY pl.tier_rank;

COMMIT;
