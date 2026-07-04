-- Baseline for group_categories table that already exists in Supabase.
-- Using IF NOT EXISTS so this is a no-op against the real database.
CREATE TABLE IF NOT EXISTS group_categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    slug TEXT NOT NULL,
    image_url TEXT,
    meta_title TEXT,
    meta_description TEXT,
    is_featured BOOLEAN NOT NULL DEFAULT false,
    order_index INTEGER NOT NULL DEFAULT 0,
    content JSONB,
    faq JSONB DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS group_categories_slug_unique_active ON group_categories (slug) WHERE deleted_at IS NULL;
