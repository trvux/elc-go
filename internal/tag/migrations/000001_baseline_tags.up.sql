-- tags is a cross-cutting taxonomy shared by news/products/projects — the
-- WordPress-standard tag pattern missing from this schema (audit finding),
-- doubling as the real relation for news<->product/project linking instead
-- of the fragile name-matching in app/(public)/tin-tuc/[slug]/page.tsx.
CREATE TABLE IF NOT EXISTS tags (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name        text NOT NULL,
    slug        text NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    deleted_at  timestamptz,
    CONSTRAINT tags_slug_unique_active UNIQUE (slug)
);

CREATE INDEX IF NOT EXISTS idx_tags_deleted_at ON tags (deleted_at);

CREATE TRIGGER update_tags_modtime BEFORE UPDATE ON tags
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
