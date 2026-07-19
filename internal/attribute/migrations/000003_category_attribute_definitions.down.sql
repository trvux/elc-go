-- Lossy: a definition attached to multiple categories can only keep one
-- (the lowest id) when collapsing back to a single column — acceptable for
-- a local-dev rollback path, same tradeoff other down migrations in this
-- project already accept.
ALTER TABLE attribute_definitions ADD COLUMN category_id uuid REFERENCES categories(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS idx_attribute_definitions_category_id ON attribute_definitions (category_id);

UPDATE attribute_definitions ad
SET category_id = sub.category_id
FROM (
    SELECT DISTINCT ON (attribute_definition_id) attribute_definition_id, category_id
    FROM category_attribute_definitions
    ORDER BY attribute_definition_id, category_id
) sub
WHERE sub.attribute_definition_id = ad.id;

DROP TABLE IF EXISTS category_attribute_definitions;
