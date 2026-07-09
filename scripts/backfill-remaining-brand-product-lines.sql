-- Pre-populates product_lines for brands beyond the original Daikin/LG pass,
-- researched via web search 2026-07-08 (official brand sites + authorized VN
-- dealer listings). Most of these brands (Carrier, Gree, Midea, Panasonic,
-- Samsung, Toshiba, Mitsubishi Electric, Mitsubishi Heavy Industries)
-- currently have 0 products in our catalog — this pre-populates ahead of
-- actually stocking them, per explicit request, so the admin has a "Dòng
-- sản phẩm" picker ready the moment a product is added. The UPDATE
-- assignment step below is therefore a no-op for those brands today; it
-- only does real work for Menred, which already has 33 products.
--
-- IMPORTANT CAVEAT: mpn_prefixes matching (`p.mpn LIKE prefix || '%'`)
-- assumes the tier-identifying code sits at the START of the mpn — true
-- for Daikin/LG/Midea. It is NOT reliably true for Panasonic, Toshiba,
-- Mitsubishi Electric/Heavy, Samsung, whose tier code is often embedded
-- mid-string (e.g. Toshiba "RAS-H10S4KCV2G-V" — the tier token S4KCV2G is
-- in the middle, not a leading prefix; Panasonic "CS-N9WKH-8" — CS-/CU- is
-- the indoor/outdoor marker, N is the tier letter that follows it). Since
-- these brands have no real products yet, this isn't causing wrong
-- assignments today, but don't trust auto-assignment blindly once real
-- MPNs are entered for these brands — spot check, and consider a
-- contains-match (not just prefix) if this bites later.
--
-- Confidence notes per brand (see chat/audit for full sourcing):
--   Daikin, LG, Midea, Toshiba: high confidence (official brand pages).
--   Panasonic: high confidence tier structure, prefix format uncertain.
--   Mitsubishi Heavy: moderate confidence (YXS variant left unmapped,
--     ambiguous vs YL/YZP).
--   Mitsubishi Electric, Carrier: moderate confidence — Carrier's
--     Premium<Luxury<Deluxe rank order is inferred, not found on one
--     authoritative price sheet.
--   Samsung: moderate-low confidence — Samsung does not publish a tier
--     chart; tiers here are inferred from model-code feature letters
--     across multiple dealer listings.
--   Gree: lowest confidence — no single official tier taxonomy exists,
--     Vietnamese retailers use inconsistent sub-brand names (Cozy/Connect/
--     Pular/Fairy/U-Crown) for what are largely the same GWC/GWH
--     engineering platforms differentiated by feature bundle, not a rigid
--     price ladder. The entry non-inverter tier is left with empty
--     mpn_prefixes (GWC/GWH alone would be too generic and collide with
--     every other Gree tier) — assign it manually if/when Gree stock is
--     added.
--   Menred: NOT price tiers. Menred's 33 real products are ~88% fresh-air/
--     heat-recovery ventilation units (NET/P5/P7/S5/R150-R350/G2-G7/O2/Q6
--     series — these model codes closely match what web research found
--     under the separate "Hagisu" brand, Menred's HVAC-ventilation
--     affiliate) differentiated by airflow capacity, not a marketing tier
--     — deliberately NOT modeled as product_lines here. Only the genuine
--     water-purifier sub-brands (Rhine, Alpsee, Weißwasser RO/UF) get
--     product_lines rows below, as named series (tier_rank 0, no
--     cheap->premium ordering implied). NOTE: the 4 existing Menred
--     water-purifier products all have an EMPTY mpn in the DB today —
--     mpn_prefixes matching below will not auto-assign them; that is a
--     real data gap (no fake mpn invented here), left for manual fix.
--
--   psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f scripts/backfill-remaining-brand-product-lines.sql

BEGIN;

-- Carrier
WITH b AS (SELECT id FROM brands WHERE name = 'Carrier')
INSERT INTO product_lines (brand_id, code, name, tier_rank, description, mpn_prefixes)
SELECT b.id, v.code, v.name, v.tier_rank, v.description, v.mpn_prefixes
FROM b, (VALUES
    ('CUR', 'Dòng Tiêu Chuẩn (Non-Inverter)', 1, 'Không inverter, chỉ làm lạnh, giá thấp nhất trong các dòng Carrier.', ARRAY['CUR','38CUR']),
    ('PREMIUM', 'Dòng Premium (Inverter Tiêu Chuẩn)', 2, 'Inverter 1 chiều, gas R32, lọc Ultra-Fresh.', ARRAY['GCVUE','GCVBE']),
    ('LUXURY', 'Dòng Luxury', 3, 'Cả 1 chiều và 2 chiều, gió 11 hướng, tự sấy khô dàn lạnh chống nấm mốc.', ARRAY['GHVPS','HES']),
    ('DELUXE', 'Dòng Deluxe (Cao Cấp Nhất)', 4, 'Dòng máy lạnh cao cấp nhất của Carrier, cả 1 chiều và 2 chiều.', ARRAY['XFT'])
) AS v(code, name, tier_rank, description, mpn_prefixes)
ON CONFLICT DO NOTHING;

-- Gree (lowest confidence — see header caveat; entry tier left unmapped)
WITH b AS (SELECT id FROM brands WHERE name = 'Gree')
INSERT INTO product_lines (brand_id, code, name, tier_rank, description, mpn_prefixes)
SELECT b.id, v.code, v.name, v.tier_rank, v.description, v.mpn_prefixes
FROM b, (VALUES
    ('AMORE', 'Dòng Không Inverter (Amore)', 1, 'Không inverter, giá rẻ nhất, làm lạnh nhanh, không tiết kiệm điện. Chưa xác định được tiền tố mpn riêng biệt — gán thủ công.', ARRAY[]::text[]),
    ('FAIRY', 'Dòng Inverter Cơ Bản (Fairy)', 2, 'Real Inverter cơ bản, cảm biến iFeel, khử trùng Cold Plasma.', ARRAY['GWC-PB','GWH-PA']),
    ('COZY', 'Dòng Inverter Trung Cấp (Cozy/Connect/Pular)', 3, 'Thêm WiFi, G-Clean tự làm sạch — bán chạy nhất phân khúc.', ARRAY['GWC-CA','GWC-QB']),
    ('UCROWN', 'Dòng Cao Cấp (U-Crown)', 4, 'Flagship, vỏ siêu mỏng, G10 Inverter tần số 1Hz, tiết kiệm điện >60%, WiFi + iFeel remote.', ARRAY['GWC-UB','GWC-UC'])
) AS v(code, name, tier_rank, description, mpn_prefixes)
ON CONFLICT DO NOTHING;

-- Midea (high confidence)
WITH b AS (SELECT id FROM brands WHERE name = 'Midea')
INSERT INTO product_lines (brand_id, code, name, tier_rank, description, mpn_prefixes)
SELECT b.id, v.code, v.name, v.tier_rank, v.description, v.mpn_prefixes
FROM b, (VALUES
    ('AEPRO', 'AE Pro', 1, 'Dòng phổ thông giá rẻ nhất, không inverter, thiết kế "1 vít" dễ tháo vệ sinh.', ARRAY['MSAE']),
    ('XCOOL', 'X-Cool', 2, 'Làm lạnh nhanh (Turbo, Follow Me), có bản thường và bản inverter.', ARRAY['MSAFB','MSAFC']),
    ('XSAVE', 'Xtreme Save', 3, 'Điều hòa 1 chiều inverter, tiết kiệm điện tới 70%, dàn tản nhiệt mạ vàng chống ăn mòn.', ARRAY['MSAG']),
    ('AIRSTILL', 'Airstill', 4, 'Điều hòa 2 chiều (nóng/lạnh) inverter siêu tiết kiệm điện, vận hành êm.', ARRAY['MSMT']),
    ('CELEST', 'Celest Inverter', 5, 'Dòng cao cấp nhất: AI Inverter, Cool Flash, tự làm sạch, chống ăn mòn Hyper Grapfins.', ARRAY['MSCE','MAF'])
) AS v(code, name, tier_rank, description, mpn_prefixes)
ON CONFLICT DO NOTHING;

-- Panasonic (tier structure high confidence, prefix format uncertain — see header)
WITH b AS (SELECT id FROM brands WHERE name = 'Panasonic')
INSERT INTO product_lines (brand_id, code, name, tier_rank, description, mpn_prefixes)
SELECT b.id, v.code, v.name, v.tier_rank, v.description, v.mpn_prefixes
FROM b, (VALUES
    ('N', 'Không Inverter (Non-Inverter)', 1, 'Máy nén tốc độ cố định, không inverter, Nanoe-G lọc PM2.5 cơ bản.', ARRAY['N']),
    ('PU', 'Inverter Phổ Thông', 2, 'Inverter giá rẻ nhất, chỉ bán qua hệ thống siêu thị điện máy, tính năng tối giản.', ARRAY['PU']),
    ('RU', 'Inverter Tiêu Chuẩn', 3, 'Dòng chủ lực bán chạy nhất, tự làm sạch dàn lạnh, NanoeX thế hệ 2, wifi.', ARRAY['RU']),
    ('UBAU', 'Inverter Cao Cấp', 4, 'Thêm NanoeG mạnh hơn, cảm biến, thiết kế cao cấp hơn dòng RU.', ARRAY['U','BU','AU']),
    ('XU', 'Inverter Cao Cấp Nhất (1 chiều)', 5, 'NanoeX thế hệ mới, wifi, hiệu suất CSPF cao nhất dòng 1 chiều.', ARRAY['XU']),
    ('XZ', 'Inverter Cao Cấp Nhất (2 chiều)', 5, 'Bản 2 chiều của XU, NanoeX gen 3, cảm biến độ ẩm, cánh đảo gió AEROWINGS.', ARRAY['XZ'])
) AS v(code, name, tier_rank, description, mpn_prefixes)
ON CONFLICT DO NOTHING;

-- Toshiba (high confidence — official site, but tier code is mid-string, see header caveat)
WITH b AS (SELECT id FROM brands WHERE name = 'Toshiba')
INSERT INTO product_lines (brand_id, code, name, tier_rank, description, mpn_prefixes)
SELECT b.id, v.code, v.name, v.tier_rank, v.description, v.mpn_prefixes
FROM b, (VALUES
    ('STANDARD', 'Dòng Tiêu Chuẩn', 1, 'Không inverter, làm lạnh nhanh, giá thấp nhất, chỉ có bản 1 chiều.', ARRAY['S3KS']),
    ('S4S5', 'Dòng Inverter Tiêu Chuẩn', 2, 'Hybrid Inverter (PAM+PWM) cơ bản, lọc bụi mịn PM2.5, tiết kiệm điện 35-50%.', ARRAY['S4KCV2G','S5KCV2G']),
    ('E2', 'Dòng Inverter Sang Trọng', 3, 'Thêm lọc Plasma Ion, thiết kế cao cấp hơn dòng tiêu chuẩn.', ARRAY['E2KCVG']),
    ('T4', 'Dòng Daiseikai Cao Cấp', 4, 'Flagship: Hybrid Inverter tiết kiệm >58%, Plasma Ion kháng khuẩn, độ ồn 22dB.', ARRAY['T4KCVRG'])
) AS v(code, name, tier_rank, description, mpn_prefixes)
ON CONFLICT DO NOTHING;

-- Samsung (moderate-low confidence — no official tier chart, inferred from feature-letter codes)
WITH b AS (SELECT id FROM brands WHERE name = 'Samsung')
INSERT INTO product_lines (brand_id, code, name, tier_rank, description, mpn_prefixes)
SELECT b.id, v.code, v.name, v.tier_rank, v.description, v.mpn_prefixes
FROM b, (VALUES
    ('STANDARD', 'Dòng Tiêu Chuẩn (Digital Inverter)', 1, 'Inverter cơ bản, không có công nghệ WindFree.', ARRAY['YHQ','DYHZ']),
    ('WINDFREE', 'Dòng WindFree', 2, 'Làm mát qua 23.000 lỗ nhỏ, không gió lùa trực tiếp.', ARRAY['YAAC','YGCD']),
    ('WINDFREE_WIFI', 'Dòng WindFree WiFi / Bespoke AI', 3, 'Thêm điều khiển wifi, AI Energy Mode.', ARRAY['CYFAA','CYFCA']),
    ('WINDFREE_PM25', 'Dòng WindFree PM2.5 / Avant', 4, 'Thêm lọc PM2.5 và/hoặc thiết kế Avant cao cấp — đắt nhất dòng treo tường.', ARRAY['CYECA','CYHAA'])
) AS v(code, name, tier_rank, description, mpn_prefixes)
ON CONFLICT DO NOTHING;

-- Mitsubishi Electric (moderate confidence)
WITH b AS (SELECT id FROM brands WHERE name = 'Mitsubishi Electric')
INSERT INTO product_lines (brand_id, code, name, tier_rank, description, mpn_prefixes)
SELECT b.id, v.code, v.name, v.tier_rank, v.description, v.mpn_prefixes
FROM b, (VALUES
    ('GHGR', 'Dòng Inverter Tiêu Chuẩn (1 chiều)', 1, 'Inverter tiêu chuẩn, giá rẻ, dàn nóng gọn nhẹ, không có tính năng nâng cao.', ARRAY['MSY-GH','MSY-GR']),
    ('JWJY', 'Dòng Inverter Nâng Cao (1 chiều)', 2, 'Thêm màng lọc diệt khuẩn, tiết kiệm điện tốt hơn dòng GH/GR.', ARRAY['MSY-JW','MSY-JY']),
    ('HL', 'Dòng Inverter 2 Chiều Tầm Trung', 2, 'Inverter 2 chiều (nóng/lạnh) tầm trung.', ARRAY['MSZ-HL']),
    ('HT', 'Dòng Inverter 2 Chiều Cao Cấp', 3, '2 chiều, lọc Nano Platinum + Enzyme diệt khuẩn 99.9%, cánh đảo gió 4 hướng.', ARRAY['MSZ-HT']),
    ('LN', 'Dòng Inverter 2 Chiều Cao Cấp Nhất', 4, 'Cao cấp nhất, thiết kế mặt gương, êm và tiết kiệm điện tối đa.', ARRAY['MSZ-LN'])
) AS v(code, name, tier_rank, description, mpn_prefixes)
ON CONFLICT DO NOTHING;

-- Mitsubishi Heavy Industries (moderate confidence — YXS left unmapped, ambiguous vs YL/YZP)
WITH b AS (SELECT id FROM brands WHERE name = 'Mitsubishi Heavy Industries')
INSERT INTO product_lines (brand_id, code, name, tier_rank, description, mpn_prefixes)
SELECT b.id, v.code, v.name, v.tier_rank, v.description, v.mpn_prefixes
FROM b, (VALUES
    ('CTRCS', 'Dòng Không Inverter', 1, 'Máy cơ (không inverter), giá thấp nhất, dùng cho công trình/phòng ít dùng.', ARRAY['SRK-CTR','SRK-CS']),
    ('YN', 'Dòng Inverter Tiêu Chuẩn', 2, 'Inverter DC PAM tiết kiệm ~40%, thiết kế cơ bản.', ARRAY['SRK-YN']),
    ('YT', 'Dòng Inverter Thiết Kế Châu Âu', 3, 'Thiết kế kiểu Âu sang trọng hơn (ra mắt 2018), tính năng tương đương YN.', ARRAY['SRK-YT']),
    ('YL', 'Dòng Inverter Cao Cấp', 4, 'Tích hợp mọi tính năng YN+YT, thêm đảo gió 3D Auto, tiết kiệm điện đến 70%.', ARRAY['SRK-YL']),
    ('YZP', 'Dòng Inverter Cao Cấp Nhất (2024-2025)', 5, 'Dòng mới thay thế phân khúc sang trọng, CSPF đến 5.47, thiết kế bo tròn hiện đại.', ARRAY['SRK-YZP'])
) AS v(code, name, tier_rank, description, mpn_prefixes)
ON CONFLICT DO NOTHING;

-- Menred water-purifier series only (NOT the fresh-air/ventilation majority
-- of Menred's catalog — see header). tier_rank 0 for all: these are parallel
-- named product families, not a price ladder.
WITH b AS (SELECT id FROM brands WHERE name = 'Menred')
INSERT INTO product_lines (brand_id, code, name, tier_rank, description, mpn_prefixes)
SELECT b.id, v.code, v.name, v.tier_rank, v.description, v.mpn_prefixes
FROM b, (VALUES
    ('RHINE', 'Dòng Rhine', 0, 'Máy lọc nước RO dòng Rhine.', ARRAY[]::text[]),
    ('ALPSEE', 'Dòng Alpsee', 0, 'Máy lọc nước thẩm thấu ngược Alpsee Series.', ARRAY[]::text[]),
    ('WEISSWASSER_RO', 'Weißwasser RO', 0, 'Máy lọc nước RO Weißwasser.', ARRAY['WR']),
    ('WEISSWASSER_UF', 'Weißwasser UF', 0, 'Máy lọc nước UF Weißwasser.', ARRAY['WU'])
) AS v(code, name, tier_rank, description, mpn_prefixes)
ON CONFLICT DO NOTHING;

-- Assign existing products to their line by mpn prefix (real effect only for
-- Menred today — Weißwasser rows won't match because those 4 products have
-- an empty mpn in the DB, a genuine data gap, not something this script
-- papers over).
UPDATE products p
SET product_line_id = pl.id
FROM product_lines pl
WHERE p.brand_id = pl.brand_id
  AND p.product_line_id IS NULL
  AND p.deleted_at IS NULL
  AND pl.deleted_at IS NULL
  AND EXISTS (
    SELECT 1 FROM product_variants v, unnest(pl.mpn_prefixes) prefix
    WHERE v.product_id = p.id AND v.is_default = true AND v.deleted_at IS NULL
      AND v.mpn IS NOT NULL AND v.mpn <> ''
      AND v.mpn LIKE prefix || '%'
  );

SELECT b.name AS brand, pl.code, pl.name, pl.tier_rank, count(p.id) AS assigned_products
FROM product_lines pl
JOIN brands b ON b.id = pl.brand_id
LEFT JOIN products p ON p.product_line_id = pl.id AND p.deleted_at IS NULL
GROUP BY b.name, pl.id, pl.code, pl.name, pl.tier_rank
ORDER BY b.name, pl.tier_rank;

COMMIT;
