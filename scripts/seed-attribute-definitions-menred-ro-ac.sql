-- Extends attribute_definitions to close the remaining real coverage gaps
-- found via a full products.specs label audit (2026-07-17) against
-- elc_prod_ref: Menred fresh-air (19 new fields, ~6 already existed from
-- the original seed and are reused via label synonyms in the migration
-- script, not duplicated here), Máy lọc nước RO (15 new fields), and 3
-- shared AC fields (dàn lạnh/dàn nóng/mặt nạ model codes — these were
-- previously section-header markers the old migration script had no way to
-- capture; see the section-state-tracking rewrite of
-- cmd/migrate-specs-to-attributes).
--
--   psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f scripts/seed-attribute-definitions-menred-ro-ac.sql

BEGIN;

INSERT INTO attribute_definitions (code, name, group_label, data_type, unit, options, order_index, is_required)
VALUES
    -- Menred fresh-air / dehumidifier cluster
    ('hong_gio', 'Họng gió', NULL, 'number', 'mm', ARRAY[]::text[], 20, false),
    ('cong_suat_dinh_danh', 'Công suất định danh', NULL, 'number', 'W', ARRAY[]::text[], 21, false),
    ('luu_luong_cap_gio_sach_cao_nhat', 'Lưu lượng cấp gió sạch cao nhất', NULL, 'number', 'm³/h', ARRAY[]::text[], 22, false),
    ('quat_gio', 'Quạt gió', NULL, 'text', NULL, ARRAY[]::text[], 23, false),
    ('cong_suat_da_toc_do', 'Công suất (đa tốc độ)', NULL, 'text', NULL, ARRAY[]::text[], 24, false),
    ('luu_luong_cap_gio_tuoi', 'Lưu lượng cấp gió tươi', NULL, 'text', NULL, ARRAY[]::text[], 25, false),
    ('so_luong_bo_loc', 'Số lượng bộ lọc', NULL, 'number', NULL, ARRAY[]::text[], 26, false),
    ('luu_luong_cap_gio_sach', 'Lưu lượng cấp gió sạch', NULL, 'number', 'm³/h', ARRAY[]::text[], 27, false),
    ('model_bo_loc', 'Model bộ lọc', NULL, 'text', NULL, ARRAY[]::text[], 28, false),
    ('hieu_suat_thu_hoi_nhiet', 'Hiệu suất thu hồi nhiệt', NULL, 'text', NULL, ARRAY[]::text[], 29, false),
    ('ap_suat_khi_hoi', 'Áp suất khí hồi', NULL, 'number', 'Pa', ARRAY[]::text[], 30, false),
    ('ap_suat_duong_khi_cap', 'Áp suất đường khí cấp', NULL, 'number', 'Pa', ARRAY[]::text[], 31, false),
    ('su_dung_may_nen', 'Sử dụng máy nén', NULL, 'text', NULL, ARRAY[]::text[], 32, false),
    ('tinh_nang_kem_theo', 'Tính năng kèm theo', NULL, 'text', NULL, ARRAY[]::text[], 33, false),
    ('bo_loc_khi', 'Bộ lọc khí', NULL, 'text', NULL, ARRAY[]::text[], 34, false),
    ('cong_suat_bu_am', 'Công suất bù ẩm', NULL, 'number', 'kg/h', ARRAY[]::text[], 35, false),
    ('tro_luc_gio', 'Trở lực gió', NULL, 'number', 'Pa', ARRAY[]::text[], 36, false),
    ('do_cung_nuoc_mem', 'Độ cứng của nước mềm', NULL, 'text', NULL, ARRAY[]::text[], 37, false),
    ('kha_nang_khu_am', 'Khử ẩm, kiểm soát độ ẩm', NULL, 'text', NULL, ARRAY[]::text[], 38, false),

    -- Máy lọc nước RO 3 in 1
    ('cong_suat_dinh_muc_ro', 'Công suất định mức', NULL, 'text', NULL, ARRAY[]::text[], 39, false),
    ('dien_ap_dinh_muc_ro', 'Điện áp định mức', NULL, 'text', NULL, ARRAY[]::text[], 40, false),
    ('nguon_nuoc_ap_dung', 'Nguồn nước áp dụng', NULL, 'text', NULL, ARRAY[]::text[], 41, false),
    ('nhiet_do_nuoc_ap_dung', 'Nhiệt độ nước áp dụng', NULL, 'text', NULL, ARRAY[]::text[], 42, false),
    ('model_ro', 'Model', NULL, 'text', NULL, ARRAY[]::text[], 43, false),
    ('mang_ro_hieu_suat', 'Màng RO hiệu suất lọc', NULL, 'text', NULL, ARRAY[]::text[], 44, false),
    ('tong_dung_tich_loc_dinh_muc', 'Tổng dung tích nước lọc định mức', NULL, 'text', NULL, ARRAY[]::text[], 45, false),
    ('luu_luong_nuoc_ro', 'Lưu lượng nước', NULL, 'text', NULL, ARRAY[]::text[], 46, false),
    ('bo_loc_thay_the', 'Bộ lọc / tuổi thọ bộ lọc', NULL, 'text', NULL, ARRAY[]::text[], 47, false),
    ('cap_hieu_suat_nuoc', 'Cấp hiệu suất sử dụng nước', NULL, 'text', NULL, ARRAY[]::text[], 48, false),
    ('ap_suat_nuoc_dau_vao', 'Áp suất nước đầu vào', NULL, 'text', NULL, ARRAY[]::text[], 49, false),
    ('luu_luong_nuoc_nong', 'Lưu lượng nước nóng', NULL, 'text', NULL, ARRAY[]::text[], 50, false),
    ('cong_suat_lam_nong_nuoc', 'Công suất làm nóng nước', NULL, 'number', 'W', ARRAY[]::text[], 51, false),
    ('be_chua_nuoc', 'Bể chứa nước tích hợp', NULL, 'boolean', NULL, ARRAY[]::text[], 52, false),

    -- Shared across AC categories — section-header model codes, captured by
    -- the section-state-tracking migration script, one per indoor/outdoor
    -- unit/cassette-panel section.
    ('ma_dan_lanh', 'Mã model dàn lạnh', 'Dàn lạnh', 'text', NULL, ARRAY[]::text[], 90, false),
    ('ma_dan_nong', 'Mã model dàn nóng', 'Dàn nóng', 'text', NULL, ARRAY[]::text[], 91, false),
    ('ma_mat_na', 'Mã model mặt nạ', NULL, 'text', NULL, ARRAY[]::text[], 92, false)
ON CONFLICT (code) WHERE deleted_at IS NULL DO NOTHING;

-- Attach: Menred + RO fields to their one category each; AC model-code
-- fields to all 5 AC hardware-group categories (sparse fill is normal —
-- not every AC category has a "mặt nạ", e.g. wall-mount units don't).
WITH menred_cat AS (
    SELECT id FROM categories WHERE name = 'Máy cấp khí tươi, lọc không khí' AND deleted_at IS NULL
),
ro_cat AS (
    SELECT id FROM categories WHERE name = 'Máy lọc nước RO 3 in 1' AND deleted_at IS NULL
),
ac_cats AS (
    SELECT id FROM categories WHERE name IN (
        'Máy lạnh treo tường', 'Máy lạnh âm trần đa hướng thổi', 'Máy lạnh áp trần',
        'Máy lạnh giấu trần nối ống gió', 'Máy lạnh tủ đứng'
    ) AND deleted_at IS NULL
),
menred_defs AS (
    SELECT id FROM attribute_definitions WHERE code IN (
        'hong_gio','cong_suat_dinh_danh','luu_luong_cap_gio_sach_cao_nhat','quat_gio',
        'cong_suat_da_toc_do','luu_luong_cap_gio_tuoi','so_luong_bo_loc','luu_luong_cap_gio_sach',
        'model_bo_loc','hieu_suat_thu_hoi_nhiet','ap_suat_khi_hoi','ap_suat_duong_khi_cap',
        'su_dung_may_nen','tinh_nang_kem_theo','bo_loc_khi','cong_suat_bu_am','tro_luc_gio',
        'do_cung_nuoc_mem','kha_nang_khu_am'
    ) AND deleted_at IS NULL
),
ro_defs AS (
    SELECT id FROM attribute_definitions WHERE code IN (
        'cong_suat_dinh_muc_ro','dien_ap_dinh_muc_ro','nguon_nuoc_ap_dung','nhiet_do_nuoc_ap_dung',
        'model_ro','mang_ro_hieu_suat','tong_dung_tich_loc_dinh_muc','luu_luong_nuoc_ro',
        'bo_loc_thay_the','cap_hieu_suat_nuoc','ap_suat_nuoc_dau_vao','luu_luong_nuoc_nong',
        'cong_suat_lam_nong_nuoc','be_chua_nuoc'
    ) AND deleted_at IS NULL
),
ac_defs AS (
    SELECT id FROM attribute_definitions WHERE code IN ('ma_dan_lanh','ma_dan_nong','ma_mat_na') AND deleted_at IS NULL
)
INSERT INTO category_attribute_definitions (category_id, attribute_definition_id)
SELECT menred_cat.id, menred_defs.id FROM menred_cat, menred_defs
UNION ALL
SELECT ro_cat.id, ro_defs.id FROM ro_cat, ro_defs
UNION ALL
SELECT ac_cats.id, ac_defs.id FROM ac_cats, ac_defs
ON CONFLICT DO NOTHING;

COMMIT;
