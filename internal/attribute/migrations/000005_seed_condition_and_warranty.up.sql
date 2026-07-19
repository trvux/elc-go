-- Seeds two definitions absorbed from Product's own hardcoded columns
-- (condition, warranty_months) as part of the product-core redesign — see
-- internal/product/migrations/000014_drop_condition_and_warranty.up.sql,
-- which backfills product_attribute_values from the old columns before
-- dropping them.
--
-- tinh_trang_san_pham: no category_attribute_definitions row is inserted,
-- which per this module's own convention (see 000003) means it's global —
-- applies to every category, same as every other spec.
--
-- ON CONFLICT DO NOTHING (2026-07-19, found via cutover rehearsal against a
-- production backup): both codes already exist on production as real,
-- pre-redesign attribute_definitions rows (same pre-existing-data situation
-- as 000004's dedup) — a plain INSERT fails outright there. dev never hit
-- this because its attribute_definitions was rebuilt from scratch.
INSERT INTO attribute_definitions (code, name, data_type, options)
VALUES ('tinh_trang_san_pham', 'Tình trạng sản phẩm', 'select', ARRAY['Mới', 'Cũ'])
ON CONFLICT (code) WHERE deleted_at IS NULL DO NOTHING;

-- bao_hanh_tong: grouped with the existing bao_hanh_may_nen definition so
-- both show together in the admin "Thông số kỹ thuật" / storefront specs —
-- it's a manufacturer spec like any other, not something elc (a reseller)
-- sets itself. Falls back to no group if bao_hanh_may_nen doesn't exist in
-- this environment's seed data.
INSERT INTO attribute_definitions (code, name, data_type, unit, group_label)
VALUES (
    'bao_hanh_tong', 'Bảo hành tổng', 'number', 'tháng',
    (SELECT group_label FROM attribute_definitions WHERE code = 'bao_hanh_may_nen')
)
ON CONFLICT (code) WHERE deleted_at IS NULL DO NOTHING;
