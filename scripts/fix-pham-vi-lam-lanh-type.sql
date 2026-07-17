-- "Phạm vi làm lạnh hiệu quả" ("Sử dụng cho phòng") is frequently a genuine
-- range ("36-40 m2") rather than a single number ("<=12") depending on the
-- product — found 2026-07-18 auditing a product whose value never reduces
-- to one number without arbitrarily picking a bound. Same trade-off
-- kích thước/độ ồn already made: text preserves the full real value,
-- losing only the ability to numerically filter by cooling area later.
--
--   psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f scripts/fix-pham-vi-lam-lanh-type.sql

UPDATE attribute_definitions SET data_type = 'text', updated_at = now() WHERE code = 'pham_vi_lam_lanh';
