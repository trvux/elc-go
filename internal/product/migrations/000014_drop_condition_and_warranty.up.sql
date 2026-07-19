-- Moves condition and warranty duration into the attribute system (seeded
-- by internal/attribute/migrations/000005_seed_condition_and_warranty.up.sql,
-- which must run first) — they're manufacturer specs like any other, no
-- different from xuat_xu/bao_hanh_may_nen. warranty_terms (per-product free
-- text) is dropped with no replacement: elc is a reseller pass-through, not
-- the manufacturer, so warranty policy belongs on Brand, not Product — see
-- internal/brand/migrations for warranty_policy.
--
-- Backfill condition -> product_attribute_values before dropping the
-- column.
INSERT INTO product_attribute_values (product_id, attribute_definition_id, value_options)
SELECT p.id, ad.id, ARRAY[CASE WHEN p.condition = 'used' THEN 'Cũ' ELSE 'Mới' END]
FROM products p, attribute_definitions ad
WHERE ad.code = 'tinh_trang_san_pham';

-- warranty_months/warranty_terms are dropped with no backfill — both are
-- 100% empty in the live dataset at migration time (0/197 products had
-- either set), so there is nothing to carry into bao_hanh_tong.
ALTER TABLE products DROP COLUMN condition;
ALTER TABLE products DROP COLUMN warranty_months;
ALTER TABLE products DROP COLUMN warranty_terms;

-- Do NOT drop the product_condition enum type here — it's shared with
-- projects.condition (internal/project/migrations/000001_baseline_projects.up.sql),
-- an unrelated module that still uses it.
