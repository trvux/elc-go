-- authors is the byline entity for news/blog content — separate from the
-- auth module's User (admin login accounts), see docs/SEO-REDESIGN audit:
-- NewsArticle.author needs a real person, not the Organization hardcode.
CREATE TABLE IF NOT EXISTS authors (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name        text NOT NULL,
    slug        text NOT NULL,
    avatar_url  text NOT NULL DEFAULT '',
    bio         text NOT NULL DEFAULT '',
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    deleted_at  timestamptz,
    CONSTRAINT authors_slug_unique_active UNIQUE (slug)
);

CREATE INDEX IF NOT EXISTS idx_authors_deleted_at ON authors (deleted_at);

-- update_updated_at_column() already exists (shared across every module's
-- baseline migration) — reuse it instead of defining a duplicate function.
CREATE TRIGGER update_authors_modtime BEFORE UPDATE ON authors
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

ALTER TABLE news ADD COLUMN IF NOT EXISTS author_id uuid REFERENCES authors(id) ON DELETE SET NULL;
