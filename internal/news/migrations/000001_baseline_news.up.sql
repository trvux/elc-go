-- Baseline for the news table that already exists in Supabase — IF NOT
-- EXISTS so this is a no-op against the real DB. Column types/defaults/
-- constraints confirmed against the live DB via `\d news` before writing
-- this migration.
--
-- news.slug is a PLAIN UNIQUE constraint (named "services_slug_key" — a
-- leftover from a copy-pasted migration, not actually related to the
-- services table) — unlike brand/group/category's partial "unique among
-- non-deleted rows" index, this one applies even to soft-deleted rows.
-- That's why Create() must resurrect a soft-deleted row on slug reuse
-- rather than plain-INSERT — the constraint itself would reject the insert
-- otherwise. See docs/news.md.
--
-- The triggers `update_news_modtime` and `update_news_updated_at` (both
-- calling the same update_updated_at_column() function on every UPDATE)
-- already exist on the live DB and are intentionally NOT recreated here.
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS news (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    content JSONB NOT NULL DEFAULT '{}'::jsonb,
    image TEXT NOT NULL DEFAULT '',
    is_published BOOLEAN NOT NULL DEFAULT false,
    order_index INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    meta_title TEXT,
    meta_description TEXT,
    category_id UUID REFERENCES categories(id) ON DELETE SET NULL
);
