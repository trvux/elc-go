-- Fixes 3 pre-existing Menred water-purifier product_lines (Alpsee, Rhine,
-- Weißwasser UF) that were created by an earlier pass this session but
-- never actually got their products assigned:
--   - ALPSEE/RHINE had empty mpn_prefixes (nothing to match against).
--   - WEISSWASSER_UF's mpn_prefixes=['WU'] never matched because the real
--     mpn is "UF WU2.SH01" (prefixed with "UF ", not starting with "WU").
-- All 3 sub-brand names are stated directly in the product's own `name`
-- field, so matched by name here instead of mpn prefix — same
-- self-evident-from-our-own-data reasoning as the Acis LUX line.
-- "MÁY LỌC NƯỚC RO 3 IN 1 MENRED" (mpn M3.C8) has no stated sub-brand and
-- is deliberately left unmapped, not guessed into one of these 3.
--
-- Run: psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f scripts/backfill-menred-ro-named-lines.sql

BEGIN;

UPDATE products p
SET product_line_id = pl.id
FROM brands b, product_lines pl
WHERE p.brand_id = b.id AND b.name = 'Menred'
  AND p.product_line_id IS NULL AND p.deleted_at IS NULL
  AND pl.brand_id = b.id AND pl.deleted_at IS NULL
  AND (
    (pl.code = 'ALPSEE' AND p.name ILIKE '%Alpsee%')
    OR (pl.code = 'RHINE' AND p.name ILIKE '%Rhine%')
    OR (pl.code = 'WEISSWASSER_UF' AND p.name ILIKE '%Wei%wasser%UF%')
  );

SELECT pl.code, pl.name, count(p.id) AS assigned_products
FROM product_lines pl
JOIN brands b ON b.id = pl.brand_id
LEFT JOIN products p ON p.product_line_id = pl.id AND p.deleted_at IS NULL
WHERE b.name = 'Menred' AND pl.code IN ('ALPSEE','RHINE','WEISSWASSER_RO','WEISSWASSER_UF')
GROUP BY pl.id, pl.code, pl.name;

COMMIT;
