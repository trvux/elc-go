-- tinh_trang_san_pham is a select-type attribute (single choice), which the
-- application layer's validateAttributeValues expects stored in value_text
-- (see internal/product/application/attribute_values.go) -- same as every
-- other select-type attribute (loai_gas_lanh, phan_khuc_hp, ...). The
-- backfill in migration 000014_drop_condition_and_warranty.up.sql wrote it
-- into value_options instead (an ARRAY, the multiselect shape), leaving
-- value_text NULL on every row it touched -- effectively every product with
-- this attribute set (197/197 in dev at the time this was found).
--
-- This went unnoticed because ProductSpecsTab.tsx's admin-form padding used
-- to submit a blank "" value_text placeholder for definitions with no real
-- value, which happened to pass the Go backend's non-nil check regardless
-- of whether a real value existed -- masking the shape mismatch. Once that
-- masking bug was fixed (frontend no longer sends blank stubs), saving any
-- product started failing with "value_text is required for data_type
-- select" for this attribute specifically.
--
-- Usage: run once against a target database.
--   PGPASSWORD=... psql -h <host> -U elc -d elc -f scripts/backfill-tinh-trang-san-pham-value-text.sql

UPDATE product_attribute_values pav
SET value_text = pav.value_options[1], value_options = '{}'
FROM attribute_definitions ad
WHERE ad.id = pav.attribute_definition_id
AND ad.code = 'tinh_trang_san_pham'
AND (pav.value_text IS NULL OR pav.value_text = '')
AND array_length(pav.value_options, 1) > 0;
