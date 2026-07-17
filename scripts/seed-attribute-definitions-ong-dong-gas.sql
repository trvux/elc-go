-- "Kích thước ống đồng Gas" (e.g. "9.5 / 15.9 mm") had no matching
-- definition at all — found 2026-07-18 comparing a product's local dev
-- display against production, missing entirely rather than mismatched.
--
--   psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f scripts/seed-attribute-definitions-ong-dong-gas.sql

BEGIN;

INSERT INTO attribute_definitions (code, name, group_label, data_type, unit, options, order_index, is_required)
VALUES ('kich_thuoc_ong_dong_gas', 'Kích thước ống đồng Gas', NULL, 'text', 'mm', ARRAY[]::text[], 15, false)
ON CONFLICT (code) WHERE deleted_at IS NULL DO NOTHING;

WITH ac_cats AS (
    SELECT id FROM categories WHERE name IN (
        'Máy lạnh treo tường', 'Máy lạnh âm trần đa hướng thổi', 'Máy lạnh áp trần',
        'Máy lạnh giấu trần nối ống gió', 'Máy lạnh tủ đứng'
    ) AND deleted_at IS NULL
),
new_def AS (
    SELECT id FROM attribute_definitions WHERE code = 'kich_thuoc_ong_dong_gas' AND deleted_at IS NULL
)
INSERT INTO category_attribute_definitions (category_id, attribute_definition_id)
SELECT ac_cats.id, new_def.id FROM ac_cats, new_def
ON CONFLICT DO NOTHING;

COMMIT;
