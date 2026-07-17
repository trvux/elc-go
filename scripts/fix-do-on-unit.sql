-- do_on/do_on_dan_lanh/do_on_dan_nong were created without a unit (text
-- type), so the FE never showed "dBA" even after fixing the display
-- logic to append units for text fields too — found 2026-07-18.
--
--   psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f scripts/fix-do-on-unit.sql

UPDATE attribute_definitions SET unit = 'dBA', updated_at = now()
WHERE code IN ('do_on', 'do_on_dan_lanh', 'do_on_dan_nong') AND (unit IS NULL OR unit = '');
