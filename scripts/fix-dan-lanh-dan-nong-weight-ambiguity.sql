-- Found via cmd/audit-attribute-values 2026-07-09: 8 products (Daikin
-- FTXM/FTXV/FTHF 2-way wall-mount family) have their old specs using a
-- generic "Trọng lượng" label for BOTH indoor and outdoor unit weight
-- (no "dàn lạnh"/"dàn nóng" qualifier), unlike every other product which
-- uses qualified labels. The original migration's exact-name matching
-- could only capture one of the two same-labeled values per product
-- (ON CONFLICT silently dropped the second insert) — trong_luong_dan_nong
-- never got populated for these 8.
--
-- Safe to auto-resolve (not a guess): in a split-system AC, the indoor
-- blower unit (dàn lạnh) is always physically lighter than the outdoor
-- compressor unit (dàn nóng) — same category of certain physical fact as
-- the kW/BTU conversion used elsewhere in this project, not domain
-- speculation. For each affected product, take its two raw "Trọng lượng"
-- spec values: the smaller becomes trong_luong_dan_lanh (already stored,
-- verified to already hold the smaller value in every case checked), the
-- larger becomes trong_luong_dan_nong (currently missing).
--
-- Run: psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f scripts/fix-dan-lanh-dan-nong-weight-ambiguity.sql

BEGIN;

WITH raw AS (
    SELECT
        p.id AS product_id,
        p.category_id,
        (item->>'value') AS raw_value
    FROM products p, jsonb_array_elements(p.specs) AS item
    WHERE p.deleted_at IS NULL
      AND item->>'label' = 'Trọng lượng'
      AND item->>'value' IS NOT NULL
), parsed AS (
    SELECT product_id, category_id,
           (regexp_replace(raw_value, '[^0-9.]', '', 'g'))::numeric AS kg
    FROM raw
    WHERE regexp_replace(raw_value, '[^0-9.]', '', 'g') <> ''
), ranked AS (
    SELECT product_id, category_id, kg,
           row_number() OVER (PARTITION BY product_id ORDER BY kg ASC) AS rn,
           count(*) OVER (PARTITION BY product_id) AS n
    FROM parsed
), outdoor AS (
    -- Only touch products with exactly 2 ambiguous "Trọng lượng" entries
    -- and where dàn nóng isn't already populated some other way.
    SELECT r.product_id, r.category_id, r.kg
    FROM ranked r
    WHERE r.n = 2 AND r.rn = 2
)
INSERT INTO product_attribute_values (product_id, attribute_definition_id, value_number)
SELECT o.product_id, ad.id, o.kg
FROM outdoor o
JOIN attribute_definitions ad ON ad.category_id = o.category_id AND ad.code = 'trong_luong_dan_nong' AND ad.deleted_at IS NULL
ON CONFLICT (product_id, attribute_definition_id) WHERE deleted_at IS NULL DO NOTHING;

SELECT p.name, av.value_number AS trong_luong_dan_nong_kg
FROM product_attribute_values av
JOIN attribute_definitions ad ON ad.id = av.attribute_definition_id AND ad.code = 'trong_luong_dan_nong'
JOIN products p ON p.id = av.product_id
WHERE av.created_at > now() - interval '1 minute'
ORDER BY p.name;

COMMIT;
