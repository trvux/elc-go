-- Migration 000003 dropped attribute_definitions.category_id, which
-- silently dropped the two partial unique indexes that depended on it
-- (attribute_definitions_category_code_unique_active/global_code_unique_active)
-- — Postgres auto-drops indexes/constraints referencing a dropped column.
-- That left `code` with no uniqueness guarantee at all. Now that category
-- scoping lives in category_attribute_definitions (many-to-many), `code`
-- itself is the one stable identifier and should just be globally unique
-- while active, same "unique while active, soft-deleted rows can be
-- reused" pattern as product_lines_brand_code_unique_active.
--
-- 2026-07-19: found via a real cutover rehearsal against a production
-- backup (dev's attribute_definitions was rebuilt from scratch during the
-- redesign and never had this problem) — production's real, pre-redesign
-- attribute_definitions has genuine duplicate codes under the old 1-1
-- category_id model (e.g. "Công suất làm lạnh" duplicated once per each of
-- the 5 AC categories that share the same spec vocabulary). 000003 copied
-- these into category_attribute_definitions as-is without merging, so the
-- index below would fail against real data. Dedupe first: keep the
-- lowest-id row per code, repoint category_attribute_definitions/
-- product_attribute_values to it, soft-delete the rest. This is a generic
-- safety net so migrate-all.sh doesn't crash here — it is NOT meant to
-- reproduce the redesign's actual curated attribute set (which already
-- exists, fully deduped, on dev, and gets restored over this table
-- wholesale right after deploy as part of the cutover's Phase 3).
-- min()/max() have no aggregate defined for uuid in Postgres — use
-- first_value() over an ORDER BY instead (uuid supports the < operator
-- needed for ordering, just not the MIN/MAX aggregate).
CREATE TEMP TABLE _attr_code_canonical AS
SELECT id, first_value(id) OVER (PARTITION BY code ORDER BY id) AS canonical_id
FROM attribute_definitions
WHERE deleted_at IS NULL;

INSERT INTO category_attribute_definitions (category_id, attribute_definition_id)
SELECT cad.category_id, c.canonical_id
FROM category_attribute_definitions cad
JOIN _attr_code_canonical c ON c.id = cad.attribute_definition_id
WHERE c.id <> c.canonical_id
ON CONFLICT DO NOTHING;

UPDATE product_attribute_values pav
SET attribute_definition_id = c.canonical_id
FROM _attr_code_canonical c
WHERE pav.attribute_definition_id = c.id
  AND c.id <> c.canonical_id
  AND NOT EXISTS (
      SELECT 1 FROM product_attribute_values pav2
      WHERE pav2.product_id = pav.product_id
        AND pav2.attribute_definition_id = c.canonical_id
        AND pav2.deleted_at IS NULL
  );

-- Leftover values that couldn't be repointed above (the product already had
-- a value under the canonical id) would otherwise dangle once the dupe
-- definition is soft-deleted below.
UPDATE product_attribute_values pav
SET deleted_at = now()
FROM _attr_code_canonical c
WHERE pav.attribute_definition_id = c.id
  AND c.id <> c.canonical_id
  AND pav.deleted_at IS NULL;

DELETE FROM category_attribute_definitions cad
USING _attr_code_canonical c
WHERE cad.attribute_definition_id = c.id
  AND c.id <> c.canonical_id;

UPDATE attribute_definitions ad
SET deleted_at = now()
FROM _attr_code_canonical c
WHERE ad.id = c.id
  AND c.id <> c.canonical_id;

DROP TABLE _attr_code_canonical;

CREATE UNIQUE INDEX IF NOT EXISTS attribute_definitions_code_unique_active
    ON attribute_definitions (code)
    WHERE deleted_at IS NULL;
