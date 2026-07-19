-- The brand-page FAQ feature was removed entirely (Go module and frontend
-- both stopped reading/writing this column — see internal/brand/domain for
-- the entity, which no longer carries a FAQ field).
ALTER TABLE brands DROP COLUMN IF EXISTS faq;
