-- Seeds attribute_definitions for all 12 real categories — replaces the
-- free-text spec label the admin used to hand-type per product (see
-- ProductSpecsTab.tsx before this pass, and the label-drift audit in
-- docs/product-v2-design.md: "Độ ồn" existed as 3 different label strings).
-- Idempotent: ON CONFLICT DO NOTHING against the partial unique indexes
-- from migration 000001.
--
--   psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f scripts/seed-attribute-definitions.sql

BEGIN;

-- Global attribute (category_id NULL) — applies to every category.
-- warranty_months/warranty_terms are NOT duplicated here — those are
-- already dedicated Product columns from the variant-refactor pass.
INSERT INTO attribute_definitions (category_id, code, name, group_label, data_type, unit, options, order_index, is_required)
VALUES (NULL, 'xuat_xu', 'Xuất xứ', NULL, 'select', NULL, ARRAY['Thái Lan','Nhật Bản','Trung Quốc','Việt Nam','Malaysia','Indonesia'], 0, false)
ON CONFLICT DO NOTHING;

-- AC hardware group (Máy lạnh treo tường / âm trần đa hướng thổi / áp trần /
-- giấu trần nối ống gió / tủ đứng) — ~85% of the catalog, share one
-- attribute set (confirmed via the real label audit against all 5
-- categories in docs/catalog-audit-2026-07-08.md).
WITH ac_categories AS (
    SELECT id FROM categories WHERE name IN (
        'Máy lạnh treo tường',
        'Máy lạnh âm trần đa hướng thổi',
        'Máy lạnh áp trần',
        'Máy lạnh giấu trần nối ống gió',
        'Máy lạnh tủ đứng'
    ) AND deleted_at IS NULL
)
INSERT INTO attribute_definitions (category_id, code, name, group_label, data_type, unit, options, order_index, is_required)
SELECT ac.id, v.code, v.name, v.group_label, v.data_type, v.unit, v.options, v.order_index, v.is_required
FROM ac_categories ac, (VALUES
    -- Chung
    ('loai_may', 'Loại máy', NULL::text, 'text', NULL::text, ARRAY[]::text[], 1, false),
    ('cong_nghe_inverter', 'Công nghệ Inverter', NULL, 'boolean', NULL, ARRAY[]::text[], 2, false),
    ('loai_gas_lanh', 'Loại Gas lạnh', NULL, 'select', NULL, ARRAY['R32','R410A','R22','R290'], 3, false),
    -- BTU is the canonical/manufacturer-sheet value; HP is a separate
    -- admin-picked marketing bucket (VN retail "ngựa" convention isn't a
    -- fixed ratio of BTU — see docs/product-v2-design.md), kW is computed
    -- for display from BTU on the frontend, never stored.
    ('cong_suat_lam_lanh_btu', 'Công suất làm lạnh', NULL, 'number', 'BTU/h', ARRAY[]::text[], 4, true),
    ('phan_khuc_hp', 'Phân khúc công suất (HP)', NULL, 'select', NULL, ARRAY['1 HP','1.5 HP','2 HP','2.5 HP','3 HP','4 HP','5 HP','6 HP','10 HP'], 5, true),
    ('cong_suat_suoi_btu', 'Công suất sưởi', NULL, 'number', 'BTU/h', ARRAY[]::text[], 6, false),
    ('pham_vi_lam_lanh', 'Phạm vi làm lạnh hiệu quả', NULL, 'number', 'm²', ARRAY[]::text[], 7, false),
    ('dien_nang_tieu_thu', 'Điện năng tiêu thụ', NULL, 'number', 'W', ARRAY[]::text[], 8, false),
    ('hieu_suat_cspf', 'Hiệu suất năng lượng (CSPF)', NULL, 'number', NULL, ARRAY[]::text[], 9, false),
    ('nhan_nang_luong', 'Nhãn năng lượng tiết kiệm điện', NULL, 'select', NULL, ARRAY['1 sao','2 sao','3 sao','4 sao','5 sao'], 10, false),
    ('nguon_dien', 'Nguồn điện', NULL, 'text', NULL, ARRAY[]::text[], 11, false),
    ('chieu_dai_ong_gas_toi_da', 'Chiều dài ống gas tối đa', NULL, 'number', 'm', ARRAY[]::text[], 12, false),
    ('chenh_lech_do_cao_toi_da', 'Chênh lệch độ cao tối đa', NULL, 'number', 'm', ARRAY[]::text[], 13, false),
    ('do_on', 'Độ ồn', NULL, 'text', NULL, ARRAY[]::text[], 14, false),
    -- Dàn lạnh
    ('kich_thuoc_dan_lanh', 'Kích thước', 'Dàn lạnh', 'text', 'mm', ARRAY[]::text[], 1, false),
    ('trong_luong_dan_lanh', 'Trọng lượng', 'Dàn lạnh', 'number', 'kg', ARRAY[]::text[], 2, false),
    -- Dàn nóng
    ('kich_thuoc_dan_nong', 'Kích thước', 'Dàn nóng', 'text', 'mm', ARRAY[]::text[], 1, false),
    ('trong_luong_dan_nong', 'Trọng lượng', 'Dàn nóng', 'number', 'kg', ARRAY[]::text[], 2, false)
) AS v(code, name, group_label, data_type, unit, options, order_index, is_required)
ON CONFLICT DO NOTHING;

-- Máy cấp khí tươi, lọc không khí
WITH cat AS (SELECT id FROM categories WHERE name = 'Máy cấp khí tươi, lọc không khí' AND deleted_at IS NULL)
INSERT INTO attribute_definitions (category_id, code, name, group_label, data_type, unit, options, order_index, is_required)
SELECT cat.id, v.code, v.name, NULL, v.data_type, v.unit, v.options, v.order_index, false
FROM cat, (VALUES
    ('luu_luong_khi', 'Lưu lượng khí', 'number', 'm³/h', ARRAY[]::text[], 1),
    ('hieu_suat_loc', 'Hiệu suất lọc', 'text', NULL::text, ARRAY[]::text[], 2),
    ('cong_suat_tieu_thu', 'Công suất tiêu thụ', 'number', 'W', ARRAY[]::text[], 3),
    ('kich_thuoc', 'Kích thước', 'text', 'mm', ARRAY[]::text[], 4),
    ('trong_luong', 'Trọng lượng', 'number', 'kg', ARRAY[]::text[], 5),
    ('do_on', 'Độ ồn', 'text', NULL, ARRAY[]::text[], 6)
) AS v(code, name, data_type, unit, options, order_index)
ON CONFLICT DO NOTHING;

-- Máy lọc nước RO 3 in 1
WITH cat AS (SELECT id FROM categories WHERE name = 'Máy lọc nước RO 3 in 1' AND deleted_at IS NULL)
INSERT INTO attribute_definitions (category_id, code, name, group_label, data_type, unit, options, order_index, is_required)
SELECT cat.id, v.code, v.name, NULL, v.data_type, v.unit, v.options, v.order_index, false
FROM cat, (VALUES
    ('cong_suat_loc', 'Công suất lọc', 'number', 'L/h', ARRAY[]::text[], 1),
    ('so_loi_loc', 'Số lõi lọc', 'number', NULL::text, ARRAY[]::text[], 2),
    ('ap_suat_dau_vao', 'Áp suất nước đầu vào', 'text', NULL, ARRAY[]::text[], 3),
    ('dung_tich_binh_chua', 'Dung tích bình chứa', 'number', 'L', ARRAY[]::text[], 4),
    ('nguon_dien', 'Nguồn điện', 'text', NULL, ARRAY[]::text[], 5),
    ('kich_thuoc', 'Kích thước', 'text', 'mm', ARRAY[]::text[], 6)
) AS v(code, name, data_type, unit, options, order_index)
ON CONFLICT DO NOTHING;

-- Bảng điều khiển
WITH cat AS (SELECT id FROM categories WHERE name = 'Bảng điều khiển' AND deleted_at IS NULL)
INSERT INTO attribute_definitions (category_id, code, name, group_label, data_type, unit, options, order_index, is_required)
SELECT cat.id, v.code, v.name, NULL, v.data_type, v.unit, v.options, v.order_index, false
FROM cat, (VALUES
    ('loai_ket_noi', 'Loại kết nối', 'select', NULL::text, ARRAY['Wifi','Zigbee','RF','Có dây'], 1),
    ('dien_ap', 'Điện áp', 'text', NULL, ARRAY[]::text[], 2),
    ('kich_thuoc', 'Kích thước', 'text', 'mm', ARRAY[]::text[], 3),
    ('chat_lieu', 'Chất liệu', 'text', NULL, ARRAY[]::text[], 4)
) AS v(code, name, data_type, unit, options, order_index)
ON CONFLICT DO NOTHING;

-- Công tắc thông minh
WITH cat AS (SELECT id FROM categories WHERE name = 'Công tắc thông minh' AND deleted_at IS NULL)
INSERT INTO attribute_definitions (category_id, code, name, group_label, data_type, unit, options, order_index, is_required)
SELECT cat.id, v.code, v.name, NULL, v.data_type, v.unit, v.options, v.order_index, false
FROM cat, (VALUES
    ('loai_ket_noi', 'Loại kết nối', 'select', NULL::text, ARRAY['Wifi','Zigbee','RF'], 1),
    ('so_kenh', 'Số kênh (nút bấm)', 'number', NULL, ARRAY[]::text[], 2),
    ('dien_ap', 'Điện áp', 'text', NULL, ARRAY[]::text[], 3),
    ('chat_lieu', 'Chất liệu', 'text', NULL, ARRAY[]::text[], 4),
    ('mau_sac', 'Màu sắc', 'text', NULL, ARRAY[]::text[], 5)
) AS v(code, name, data_type, unit, options, order_index)
ON CONFLICT DO NOTHING;

-- Cảm biến thông minh
WITH cat AS (SELECT id FROM categories WHERE name = 'Cảm biến thông minh' AND deleted_at IS NULL)
INSERT INTO attribute_definitions (category_id, code, name, group_label, data_type, unit, options, order_index, is_required)
SELECT cat.id, v.code, v.name, NULL, v.data_type, v.unit, v.options, v.order_index, false
FROM cat, (VALUES
    ('loai_cam_bien', 'Loại cảm biến', 'select', NULL::text, ARRAY['Chuyển động','Cửa từ','Nhiệt độ','Độ ẩm','Khói'], 1),
    ('loai_ket_noi', 'Loại kết nối', 'select', NULL, ARRAY['Wifi','Zigbee','RF'], 2),
    ('nguon_dien', 'Nguồn điện', 'text', NULL, ARRAY[]::text[], 3),
    ('kich_thuoc', 'Kích thước', 'text', 'mm', ARRAY[]::text[], 4)
) AS v(code, name, data_type, unit, options, order_index)
ON CONFLICT DO NOTHING;

-- Remote cầm tay
WITH cat AS (SELECT id FROM categories WHERE name = 'Remote cầm tay' AND deleted_at IS NULL)
INSERT INTO attribute_definitions (category_id, code, name, group_label, data_type, unit, options, order_index, is_required)
SELECT cat.id, v.code, v.name, NULL, v.data_type, v.unit, v.options, v.order_index, false
FROM cat, (VALUES
    ('loai_ket_noi', 'Loại kết nối', 'select', NULL::text, ARRAY['Hồng ngoại','RF','Zigbee'], 1),
    ('so_kenh', 'Số kênh điều khiển', 'number', NULL, ARRAY[]::text[], 2),
    ('nguon_dien', 'Nguồn điện (loại pin)', 'text', NULL, ARRAY[]::text[], 3)
) AS v(code, name, data_type, unit, options, order_index)
ON CONFLICT DO NOTHING;

-- Phụ kiện đồng bộ của hệ thống cấp gió tươi
WITH cat AS (SELECT id FROM categories WHERE name = 'Phụ kiện đồng bộ của hệ thống cấp gió tươi' AND deleted_at IS NULL)
INSERT INTO attribute_definitions (category_id, code, name, group_label, data_type, unit, options, order_index, is_required)
SELECT cat.id, v.code, v.name, NULL, v.data_type, v.unit, v.options, v.order_index, false
FROM cat, (VALUES
    ('loai_phu_kien', 'Loại phụ kiện', 'text', NULL::text, ARRAY[]::text[], 1),
    ('kich_thuoc', 'Kích thước', 'text', 'mm', ARRAY[]::text[], 2),
    ('chat_lieu', 'Chất liệu', 'text', NULL, ARRAY[]::text[], 3)
) AS v(code, name, data_type, unit, options, order_index)
ON CONFLICT DO NOTHING;

-- Report before committing.
SELECT COALESCE(c.name, '(Toàn bộ category)') AS category, count(*) AS attribute_count
FROM attribute_definitions ad
LEFT JOIN categories c ON c.id = ad.category_id
GROUP BY c.name
ORDER BY category;

COMMIT;
