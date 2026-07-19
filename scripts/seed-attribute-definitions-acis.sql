-- Extends attribute_definitions to cover the ACIS smart-home electronics
-- cluster (Bảng điều khiển / Công tắc thông minh / Cảm biến thông minh /
-- Remote cầm tay) — this cluster had almost zero structured coverage before
-- (confirmed via a full products.specs label audit against elc_legacy +
-- elc_prod_ref, 2026-07-17: ~40 real distinct fields per category, none
-- previously defined). Uses the new category_attribute_definitions
-- many-to-many join (see internal/attribute migration 000003) — one shared
-- definition set attached to all 4 categories, same "hardware group shares
-- one attribute set" pattern the original AC seed already used.
--
-- data_type choices are conservative: `text` unless the real sample values
-- are consistently a single clean number+unit or a clean Có/Không boolean
-- across the audited samples — compound/descriptive values (e.g. "Tiêu
-- chuẩn 500W/kênh. Dòng khởi động max = 6A.") stay `text` rather than
-- guessing a number split that would silently truncate real information.
--
--   psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f scripts/seed-attribute-definitions-acis.sql

BEGIN;

INSERT INTO attribute_definitions (code, name, group_label, data_type, unit, options, order_index, is_required)
    VALUES
        ('kich_thuoc_san_pham', 'Kích thước sản phẩm', NULL, 'text', 'mm', ARRAY[]::text[], 1, false),
        ('kich_thuoc_de_am', 'Kích thước đế âm', NULL, 'text', 'mm', ARRAY[]::text[], 2, false),
        ('so_phim_dieu_khien', 'Số phím điều khiển', NULL, 'text', NULL, ARRAY[]::text[], 3, false),
        ('man_hinh_hien_thi', 'Màn hình hiển thị', NULL, 'text', NULL, ARRAY[]::text[], 4, false),
        ('cau_chi_bao_ve', 'Cầu chì bảo vệ', NULL, 'boolean', NULL, ARRAY[]::text[], 5, false),
        ('led_bao_hong_cau_chi', 'LED báo hỏng cầu chì', NULL, 'boolean', NULL, ARRAY[]::text[], 6, false),
        ('led_bao_trang_thai', 'LED báo trạng thái', NULL, 'boolean', NULL, ARRAY[]::text[], 7, false),
        ('led_nen_ho_tro_ban_dem', 'LED nền hỗ trợ ban đêm', NULL, 'boolean', NULL, ARRAY[]::text[], 8, false),
        ('tich_hop_speaker', 'Tích hợp Speaker', NULL, 'boolean', NULL, ARRAY[]::text[], 9, false),
        ('loa_thong_bao', 'Loa thông báo', NULL, 'boolean', NULL, ARRAY[]::text[], 10, false),
        ('dieu_khien_dong_mo_tiep_diem_relay', 'Điều khiển đóng/mở tiếp điểm (Relay)', NULL, 'text', NULL, ARRAY[]::text[], 11, false),
        ('dieu_khien_ngo_ra_tiep_diem_kho', 'Điều khiển ngõ ra tiếp điểm khô', NULL, 'text', NULL, ARRAY[]::text[], 12, false),
        ('dieu_khien_hong_ngoai', 'Điều khiển hồng ngoại', NULL, 'text', NULL, ARRAY[]::text[], 13, false),
        ('dieu_khien_ngu_canh', 'Điều khiển ngữ cảnh', NULL, 'text', NULL, ARRAY[]::text[], 14, false),
        ('dien_ap_ngo_vao', 'Điện áp ngõ vào', NULL, 'text', NULL, ARRAY[]::text[], 15, false),
        ('cong_suat_ngo_ra', 'Công suất ngõ ra', NULL, 'text', NULL, ARRAY[]::text[], 16, false),
        ('cong_suat_khong_tai', 'Công suất không tải', NULL, 'number', 'W', ARRAY[]::text[], 17, false),
        ('chip_xu_ly_chinh', 'Chip xử lý chính', NULL, 'text', NULL, ARRAY[]::text[], 18, false),
        ('ram_rom_flash', 'RAM/ROM/FLASH', NULL, 'text', NULL, ARRAY[]::text[], 19, false),
        ('chip_xu_ly_cam_ung', 'Chip xử lý cảm ứng', NULL, 'text', NULL, ARRAY[]::text[], 20, false),
        ('chip_quan_ly_nguon', 'Chip quản lý nguồn', NULL, 'text', NULL, ARRAY[]::text[], 21, false),
        ('che_do_bao_ve', 'Chế độ bảo vệ', NULL, 'text', NULL, ARRAY[]::text[], 22, false),
        ('do_dai_tin_hieu_ir', 'Độ dài tín hiệu IR', NULL, 'text', NULL, ARRAY[]::text[], 23, false),
        ('dien_toan_dam_may', 'Điện toán đám mây', NULL, 'text', NULL, ARRAY[]::text[], 24, false),
        ('quan_ly_ngu_canh', 'Quản lý ngữ cảnh', NULL, 'text', NULL, ARRAY[]::text[], 25, false),
        ('giao_tiep_internet', 'Giao tiếp internet', NULL, 'text', NULL, ARRAY[]::text[], 26, false),
        ('dau_phat_tich_hop', 'Đầu phát tích hợp bên trong', NULL, 'text', NULL, ARRAY[]::text[], 27, false),
        ('giao_thuc_mang', 'Giao thức mạng', NULL, 'text', NULL, ARRAY[]::text[], 28, false),
        ('quan_ly_hen_gio', 'Quản lý hẹn giờ', NULL, 'text', NULL, ARRAY[]::text[], 29, false),
        ('quan_ly_thiet_bi', 'Quản lý thiết bị', NULL, 'text', NULL, ARRAY[]::text[], 30, false),
        ('cau_hinh_ngo_ra', 'Cấu hình ngõ ra', NULL, 'text', NULL, ARRAY[]::text[], 31, false),
        ('ngo_ra', 'Ngõ ra', NULL, 'text', NULL, ARRAY[]::text[], 32, false),
        ('goc_phat_led_ir', 'Góc phát LED IR', NULL, 'text', NULL, ARRAY[]::text[], 33, false),
        ('dau_phat_hong_ngoai_roi', 'Đầu phát hồng ngoại rời', NULL, 'text', NULL, ARRAY[]::text[], 34, false),
        ('ngo_ra_dimmable', 'Ngõ ra điều khiển tăng/giảm (dimmable)', NULL, 'text', NULL, ARRAY[]::text[], 35, false),
        ('num_xoay_tang_giam', 'Núm xoay điều chỉnh Tăng/Giảm', NULL, 'boolean', NULL, ARRAY[]::text[], 36, false),
        ('ngo_ra_relay', 'Ngõ ra Relay', NULL, 'text', NULL, ARRAY[]::text[], 37, false),
        ('tuoi_tho_pin', 'Tuổi thọ pin', NULL, 'text', NULL, ARRAY[]::text[], 38, false),
        ('kenh_dieu_khien', 'Kênh điều khiển', NULL, 'text', NULL, ARRAY[]::text[], 39, false),
        ('do_nhay_tu', 'Độ nhạy từ', NULL, 'text', NULL, ARRAY[]::text[], 40, false),
        ('dien_ap_hoat_dong', 'Điện áp hoạt động', NULL, 'text', NULL, ARRAY[]::text[], 41, false),
        ('dong_tieu_thu', 'Dòng tiêu thụ', NULL, 'text', NULL, ARRAY[]::text[], 42, false),
        ('ngo_vao', 'Ngõ vào', NULL, 'text', NULL, ARRAY[]::text[], 43, false),
        ('dieu_khien', 'Điều khiển', NULL, 'text', NULL, ARRAY[]::text[], 44, false),
        ('khoang_cach_giao_tiep', 'Khoảng cách giao tiếp', NULL, 'text', NULL, ARRAY[]::text[], 45, false)
    ON CONFLICT DO NOTHING;

-- Separate statement (not a sibling CTE) so it sees the rows the INSERT
-- above just committed-within-transaction — a WITH-clause sibling SELECT
-- would still see the pre-statement snapshot and find nothing on a first
-- run. Matched by code, so this stays idempotent/re-runnable too.
WITH acis_categories AS (
    SELECT id FROM categories WHERE name IN (
        'Bảng điều khiển', 'Công tắc thông minh', 'Cảm biến thông minh', 'Remote cầm tay'
    ) AND deleted_at IS NULL
),
all_defs AS (
    SELECT id FROM attribute_definitions WHERE code IN (
        'kich_thuoc_san_pham','kich_thuoc_de_am','so_phim_dieu_khien','man_hinh_hien_thi',
        'cau_chi_bao_ve','led_bao_hong_cau_chi','led_bao_trang_thai','led_nen_ho_tro_ban_dem',
        'tich_hop_speaker','loa_thong_bao','dieu_khien_dong_mo_tiep_diem_relay',
        'dieu_khien_ngo_ra_tiep_diem_kho','dieu_khien_hong_ngoai','dieu_khien_ngu_canh',
        'dien_ap_ngo_vao','cong_suat_ngo_ra','cong_suat_khong_tai','chip_xu_ly_chinh',
        'ram_rom_flash','chip_xu_ly_cam_ung','chip_quan_ly_nguon','che_do_bao_ve',
        'do_dai_tin_hieu_ir','dien_toan_dam_may','quan_ly_ngu_canh','giao_tiep_internet',
        'dau_phat_tich_hop','giao_thuc_mang','quan_ly_hen_gio','quan_ly_thiet_bi',
        'cau_hinh_ngo_ra','ngo_ra','goc_phat_led_ir','dau_phat_hong_ngoai_roi',
        'ngo_ra_dimmable','num_xoay_tang_giam','ngo_ra_relay','tuoi_tho_pin',
        'kenh_dieu_khien','do_nhay_tu','dien_ap_hoat_dong','dong_tieu_thu','ngo_vao',
        'dieu_khien','khoang_cach_giao_tiep'
    ) AND deleted_at IS NULL
)
INSERT INTO category_attribute_definitions (category_id, attribute_definition_id)
SELECT ac.id, ad.id FROM acis_categories ac CROSS JOIN all_defs ad
ON CONFLICT DO NOTHING;

COMMIT;
