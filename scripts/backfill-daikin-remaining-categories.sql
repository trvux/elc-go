-- Daikin's remaining 5 non-wall-mounted categories, product-line tiers
-- confirmed via web research 2026-07-09 (see commit message for the
-- specific queries/findings per category — same "Không Inverter <
-- Inverter Tiêu Chuẩn < Inverter Cao Cấp" 3-tier shape repeats across
-- ducted/cassette/ceiling, tủ đứng only has 1 line):
--
-- Máy lạnh giấu trần nối ống gió (ducted): FDBNQ/FDMNQ non-inverter oldest
-- generation < FBFC newer cheaper inverter < FBA bulkier pricier inverter
-- (with drain pump).
-- Máy lạnh áp trần (low-static ceiling): FHNQ non-inverter < FHFC inverter
-- standard < FHA inverter SkyAir premium.
-- Máy lạnh âm trần đa hướng thổi (4-way cassette): FCNQ non-inverter <
-- FCFC standard inverter (18-way blow) < FCF premium inverter
-- (highest-end currently distributed in VN).
-- Máy lạnh tủ đứng (floor standing): FVA is Daikin's one standard
-- floor-standing inverter line, capacity-only siblings, no sub-tiers.
-- Máy lạnh treo tường 2 chiều (wall-mount, heat pump): FTHF standard
-- (feature-comparable to 1-way FTKC) < FTXM premium (replaces FTXV,
-- comparable to FTKZ) ~ FTXV premium (older generation being phased out) —
-- kept as 2 separate lines since sources describe them as distinct
-- generations, not simply superseding within one name.
--
-- Run: psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f scripts/backfill-daikin-remaining-categories.sql

BEGIN;

WITH daikin AS (SELECT id FROM brands WHERE name = 'Daikin' LIMIT 1),
cat AS (
    SELECT
        (SELECT id FROM categories WHERE name = 'Máy lạnh giấu trần nối ống gió' LIMIT 1) AS ducted,
        (SELECT id FROM categories WHERE name = 'Máy lạnh áp trần' LIMIT 1) AS low_ceiling,
        (SELECT id FROM categories WHERE name = 'Máy lạnh âm trần đa hướng thổi' LIMIT 1) AS cassette,
        (SELECT id FROM categories WHERE name = 'Máy lạnh tủ đứng' LIMIT 1) AS floor_standing,
        (SELECT id FROM categories WHERE name = 'Máy lạnh treo tường' LIMIT 1) AS wall_mount
)
INSERT INTO product_lines (brand_id, category_id, code, name, tier_rank, description, mpn_prefixes)
SELECT daikin.id,
       CASE v.cat_key
           WHEN 'ducted' THEN cat.ducted
           WHEN 'low_ceiling' THEN cat.low_ceiling
           WHEN 'cassette' THEN cat.cassette
           WHEN 'floor_standing' THEN cat.floor_standing
           WHEN 'wall_mount' THEN cat.wall_mount
       END,
       v.code, v.name, v.tier_rank, v.description, v.mpn_prefixes
FROM daikin, cat, (VALUES
    -- Ducted (giấu trần nối ống gió)
    ('ducted', 'FDNQ', 'Dòng Không Inverter', 1, 'Không inverter, thế hệ cũ nhất còn bán, áp suất tĩnh thấp.', ARRAY['FDBNQ','FDMNQ']),
    ('ducted', 'FBFC', 'Dòng Inverter Tiêu Chuẩn', 2, 'Inverter, mẫu mới hơn FBA, gọn nhẹ hơn, giá rẻ hơn cùng công suất.', ARRAY['FBFC']),
    ('ducted', 'FBA', 'Dòng Inverter Cao Cấp', 3, 'Inverter, thân máy to hơn FBFC, có thêm bơm thoát nước.', ARRAY['FBA']),
    -- Low-static ceiling (áp trần)
    ('low_ceiling', 'FHNQ', 'Dòng Không Inverter', 1, 'Không inverter, thiết kế gọn.', ARRAY['FHNQ']),
    ('low_ceiling', 'FHFC', 'Dòng Inverter Tiêu Chuẩn', 2, 'Inverter, phù hợp shop/nhà hàng/văn phòng nhỏ.', ARRAY['FHFC']),
    ('low_ceiling', 'FHA', 'Dòng Inverter Cao Cấp (SkyAir)', 3, 'Inverter SkyAir, lắp đặt linh hoạt cho trần cao.', ARRAY['FHA']),
    -- 4-way cassette (âm trần đa hướng thổi)
    ('cassette', 'FCNQ', 'Dòng Không Inverter', 1, 'Không inverter, đa hướng thổi 23 kiểu.', ARRAY['FCNQ']),
    ('cassette', 'FCFC', 'Dòng Inverter Tiêu Chuẩn', 2, 'Inverter, đa hướng thổi 18 kiểu.', ARRAY['FCFC']),
    ('cassette', 'FCF', 'Dòng Inverter Cao Cấp', 3, 'Inverter cao cấp nhất đang phân phối tại VN, đa hướng thổi đều.', ARRAY['FCF']),
    -- Floor standing (tủ đứng) — single line, capacity-only siblings
    ('floor_standing', 'FVA', 'Dòng Tiêu Chuẩn', 1, 'Dòng tủ đứng inverter tiêu chuẩn duy nhất, chỉ khác công suất.', ARRAY['FVA']),
    -- Wall-mount 2-way (treo tường hai chiều)
    ('wall_mount', 'FTHF', 'Dòng Inverter 2 Chiều Tiêu Chuẩn', 1, '2 chiều (nóng/lạnh), tính năng tương đương dòng 1 chiều FTKC.', ARRAY['FTHF']),
    ('wall_mount', 'FTXM', 'Dòng Inverter 2 Chiều Cao Cấp (FTXM)', 2, '2 chiều cao cấp, thế hệ thay thế FTXV, tương đương dòng 1 chiều FTKZ.', ARRAY['FTXM']),
    ('wall_mount', 'FTXV', 'Dòng Inverter 2 Chiều Cao Cấp (FTXV)', 2, '2 chiều cao cấp thế hệ cũ hơn, đang được FTXM thay thế dần.', ARRAY['FTXV'])
) AS v(cat_key, code, name, tier_rank, description, mpn_prefixes)
ON CONFLICT DO NOTHING;

-- Assign product_line_id by mpn prefix, scoped to both brand AND category
-- (unlike the wall-mount-1-way script, these lines set category_id so this
-- join filters on it too — avoids any cross-category prefix collision).
-- Anchored to "prefix immediately followed by a digit" (~ not LIKE) —
-- plain LIKE 'FCF%' would also match FCFC-prefixed mpns (FCFC100DVM starts
-- with FCF too), silently stealing FCFC's rows into the FCF line.
UPDATE products p
SET product_line_id = pl.id
FROM product_lines pl, product_variants v
WHERE p.product_line_id IS NULL
  AND p.deleted_at IS NULL
  AND pl.category_id = p.category_id
  AND pl.deleted_at IS NULL
  AND v.product_id = p.id AND v.is_default = true AND v.deleted_at IS NULL
  AND EXISTS (SELECT 1 FROM unnest(pl.mpn_prefixes) prefix WHERE v.mpn ~ ('^' || prefix || '[0-9]'));

SELECT c.name AS category, pl.code, pl.name, count(p.id) AS assigned_products
FROM product_lines pl
JOIN categories c ON c.id = pl.category_id
LEFT JOIN products p ON p.product_line_id = pl.id AND p.deleted_at IS NULL
WHERE pl.code IN ('FDNQ','FBFC','FBA','FHNQ','FHFC','FHA','FCNQ','FCFC','FCF','FVA','FTHF','FTXM','FTXV')
GROUP BY c.name, pl.id, pl.code, pl.name, pl.tier_rank
ORDER BY c.name, pl.tier_rank;

COMMIT;
