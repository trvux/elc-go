-- The original seed only listed whole/half-HP options up to 6 in steps
-- that skipped 3.5/4.5/5.5 — a real audit of every product's raw
-- "Công suất làm lạnh"/"Công suất sưởi" HP sub-item (2026-07-18) found
-- these values genuinely occur (3.5HP/4.5HP/5.5HP cassette/duct units are
-- real Daikin SKUs in this catalog), so the select field silently rejected
-- them (no exact option match) rather than guessing.
--
--   psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f scripts/fix-phan-khuc-hp-options.sql

UPDATE attribute_definitions
SET options = ARRAY['1 HP','1.5 HP','2 HP','2.5 HP','3 HP','3.5 HP','4 HP','4.5 HP','5 HP','5.5 HP','6 HP','10 HP'],
    updated_at = now()
WHERE code = 'phan_khuc_hp';
