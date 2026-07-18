-- attribute_definitions.name for 'nguon_dien' was literally stored as
-- "Nguồn điện (loại pin)" — leaked wording from a battery-powered product
-- template (remote/small appliance), nonsensical for grid-powered AC units.
-- Found 2026-07-18 via user's manufacturer-spec audit of product FCNQ30MV1.
--
--   psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f scripts/fix-nguon-dien-name.sql

UPDATE attribute_definitions SET name = 'Nguồn điện', updated_at = now()
WHERE code = 'nguon_dien';
