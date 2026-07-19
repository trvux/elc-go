-- Replaces attribute_definitions.category_id (1-1) with a real many-to-many
-- join, so one definition (e.g. "Công suất HP") can be shared across sibling
-- categories (treo tường/âm trần/giấu trần) without duplicating the row —
-- required for facet/filter to aggregate correctly across a whole category
-- tree. "Global" (applies everywhere) is now expressed as zero rows in this
-- table, not a NULL category_id. Same cross-module join-table ownership
-- pattern as product_tags/product_attribute_values.
CREATE TABLE IF NOT EXISTS category_attribute_definitions (
    category_id             uuid NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    attribute_definition_id uuid NOT NULL REFERENCES attribute_definitions(id) ON DELETE CASCADE,
    created_at              timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (category_id, attribute_definition_id)
);
CREATE INDEX IF NOT EXISTS idx_category_attribute_definitions_definition_id
    ON category_attribute_definitions (attribute_definition_id);

INSERT INTO category_attribute_definitions (category_id, attribute_definition_id)
SELECT category_id, id FROM attribute_definitions WHERE category_id IS NOT NULL
ON CONFLICT DO NOTHING;

DROP INDEX IF EXISTS idx_attribute_definitions_category_id;
ALTER TABLE attribute_definitions DROP COLUMN category_id;
