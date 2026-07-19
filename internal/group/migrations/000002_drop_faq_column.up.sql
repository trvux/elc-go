-- The group-category-page FAQ feature was removed entirely (Go module and
-- frontend both stopped reading/writing this column — see
-- internal/group/domain for the entity, which no longer carries a FAQ field).
ALTER TABLE group_categories DROP COLUMN IF EXISTS faq;
