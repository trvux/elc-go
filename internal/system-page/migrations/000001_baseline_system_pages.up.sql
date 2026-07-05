-- Baseline for a table that already exists in Supabase — IF NOT EXISTS so
-- this is a no-op against the real DB. Columns confirmed via `\d system_pages`
-- on the live DB before writing this migration.
--
-- slug is a PLAIN UNIQUE constraint (system_pages_slug_key). There is no
-- deleted_at column and no trigger on this table — rows are a fixed,
-- admin-seeded set of hub pages (home, tin-tuc, du-an, ...) that this module
-- only reads and edits SEO meta fields on; it never creates or deletes rows.
CREATE TABLE IF NOT EXISTS system_pages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    meta_title TEXT,
    meta_description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT timezone('utc'::text, now()),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT timezone('utc'::text, now())
);
