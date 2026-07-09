-- One-off content-classification script (not app code) — creates Daikin's
-- 5 confirmed wall-mounted ("treo tường", 1-way/"một chiều") product tiers
-- and assigns product_line_id by MPN prefix, for products whose MPN clearly
-- matches a confirmed tier. Confirmed against real catalog data + web
-- research 2026-07-08 (see docs/catalog-audit-2026-07-08.md) — FTHB/FTHF/
-- FTXM/FTXV (2-way "hai chiều" models) are deliberately left unmapped, not
-- guessed. Requires migration 000006 (product_lines.mpn_prefixes).
-- Safe to re-run: product_lines insert is idempotent (ON CONFLICT DO
-- NOTHING via the partial unique index), and the UPDATE only ever sets
-- product_line_id for rows currently NULL.
--
-- Run against a DB with the product v2 schema already migrated:
--   psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f scripts/backfill-daikin-product-lines.sql

BEGIN;

-- code is a short, single, human-assigned identifier for the line — NOT a
-- concatenation of the prefixes it covers (that was an earlier mistake,
-- fixed by migration 000006: mpn_prefixes is the real array used both for
-- matching below and for the admin UI to show/edit each prefix as its own
-- chip, e.g. "FTF" and "FTC" separately rather than one "FTF-FTC" string).
WITH daikin AS (
    SELECT id FROM brands WHERE name = 'Daikin' LIMIT 1
)
INSERT INTO product_lines (brand_id, code, name, tier_rank, description, mpn_prefixes)
SELECT daikin.id, v.code, v.name, v.tier_rank, v.description, v.mpn_prefixes
FROM daikin, (VALUES
    ('FTF', 'Dòng Tiêu Chuẩn (Non-Inverter)', 1, 'Không Inverter, giá rẻ nhất trong các dòng máy lạnh treo tường Daikin.', ARRAY['FTF','FTC']),
    ('FTKB', 'Dòng Inverter Tiêu Chuẩn', 2, 'Inverter cơ bản, vận hành êm, bền, có chức năng chống ẩm mốc.', ARRAY['FTKB','ATKB']),
    ('FTKF', 'Dòng Inverter Trung Cấp', 3, 'Thêm công nghệ lọc không khí Streamer so với dòng Tiêu Chuẩn.', ARRAY['FTKF','ATKF']),
    ('FTKC', 'Dòng Inverter Cận Cao Cấp', 4, 'Thêm cảm biến "mắt thần" phát hiện người trong phòng.', ARRAY['FTKC','FTKA']),
    ('FTKY', 'Dòng Inverter Cao Cấp & Sang Trọng', 5, 'Cao cấp nhất: cân bằng độ ẩm, kết nối WiFi, mắt thần đa vùng, siêu tiết kiệm điện.', ARRAY['FTKY','FTKM','FTKZ'])
) AS v(code, name, tier_rank, description, mpn_prefixes)
ON CONFLICT DO NOTHING;

-- Assign product_line_id by MPN prefix — wall-mounted category, Daikin
-- brand, only rows not already assigned. Prefix match is case-sensitive on
-- purpose (real MPNs are consistently upper-case in this catalog).
-- unnest(pl.mpn_prefixes) instead of a hardcoded OR chain: adding a prefix
-- to a line later (e.g. via the admin UI) makes this matching logic apply
-- to it automatically next run, no script edit needed.
UPDATE products p
SET product_line_id = pl.id
FROM brands b, product_lines pl
WHERE p.brand_id = b.id
  AND b.name = 'Daikin'
  AND p.product_line_id IS NULL
  AND p.deleted_at IS NULL
  AND pl.brand_id = b.id
  AND pl.deleted_at IS NULL
  AND EXISTS (
    SELECT 1 FROM product_variants v, unnest(pl.mpn_prefixes) prefix
    WHERE v.product_id = p.id AND v.is_default = true AND v.deleted_at IS NULL
      AND v.mpn LIKE prefix || '%'
  );

-- Report what happened before committing, for a human to eyeball.
SELECT pl.code, pl.name, count(p.id) AS assigned_products
FROM product_lines pl
LEFT JOIN products p ON p.product_line_id = pl.id AND p.deleted_at IS NULL
GROUP BY pl.id, pl.code, pl.name, pl.tier_rank
ORDER BY pl.tier_rank;

SELECT v.mpn, p.name
FROM products p
JOIN brands b ON b.id = p.brand_id
JOIN product_variants v ON v.product_id = p.id AND v.is_default = true AND v.deleted_at IS NULL
WHERE b.name = 'Daikin' AND p.product_line_id IS NULL AND p.deleted_at IS NULL
ORDER BY v.mpn;

COMMIT;
