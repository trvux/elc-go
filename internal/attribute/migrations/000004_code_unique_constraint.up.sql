-- Migration 000003 dropped attribute_definitions.category_id, which
-- silently dropped the two partial unique indexes that depended on it
-- (attribute_definitions_category_code_unique_active/global_code_unique_active)
-- — Postgres auto-drops indexes/constraints referencing a dropped column.
-- That left `code` with no uniqueness guarantee at all. Now that category
-- scoping lives in category_attribute_definitions (many-to-many), `code`
-- itself is the one stable identifier and should just be globally unique
-- while active, same "unique while active, soft-deleted rows can be
-- reused" pattern as product_lines_brand_code_unique_active.
CREATE UNIQUE INDEX IF NOT EXISTS attribute_definitions_code_unique_active
    ON attribute_definitions (code)
    WHERE deleted_at IS NULL;
