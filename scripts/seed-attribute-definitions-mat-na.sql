-- Máy lạnh âm trần đa hướng thổi (and any other AC category with a cassette
-- panel) has a third "Mặt nạ" section alongside dàn lạnh/dàn nóng — found
-- via the 2026-07-17 audit's section-header trace. Only size/weight are
-- added here; ma_mat_na (model code) already exists from
-- scripts/seed-attribute-definitions-menred-ro-ac.sql.
--
--   psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f scripts/seed-attribute-definitions-mat-na.sql

BEGIN;

INSERT INTO attribute_definitions (code, name, group_label, data_type, unit, options, order_index, is_required)
VALUES
    ('kich_thuoc_mat_na', 'Kích thước', 'Mặt nạ', 'text', 'mm', ARRAY[]::text[], 4, false),
    ('trong_luong_mat_na', 'Trọng lượng', 'Mặt nạ', 'number', 'kg', ARRAY[]::text[], 5, false)
ON CONFLICT (code) WHERE deleted_at IS NULL DO NOTHING;

WITH ac_cats AS (
    SELECT id FROM categories WHERE name IN (
        'Máy lạnh treo tường', 'Máy lạnh âm trần đa hướng thổi', 'Máy lạnh áp trần',
        'Máy lạnh giấu trần nối ống gió', 'Máy lạnh tủ đứng'
    ) AND deleted_at IS NULL
),
new_defs AS (
    SELECT id FROM attribute_definitions WHERE code IN ('kich_thuoc_mat_na', 'trong_luong_mat_na') AND deleted_at IS NULL
)
INSERT INTO category_attribute_definitions (category_id, attribute_definition_id)
SELECT ac_cats.id, new_defs.id FROM ac_cats, new_defs
ON CONFLICT DO NOTHING;

COMMIT;
