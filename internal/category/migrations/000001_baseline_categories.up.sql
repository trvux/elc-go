-- Baseline for categories table that already exists in Supabase.
-- Using IF NOT EXISTS so this is a no-op against the real database. Column
-- types/defaults/constraints below were confirmed against the live DB via
-- `\d categories` before writing this migration.
CREATE TABLE IF NOT EXISTS categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    group_id UUID REFERENCES group_categories(id) ON DELETE SET NULL,
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

CREATE UNIQUE INDEX IF NOT EXISTS category_slug_unique_active ON categories (slug) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_category_group_id ON categories (group_id) WHERE deleted_at IS NULL;

-- The triggers `trg_category_slug_registry` (syncs into the shared
-- slug_registry table — same shared platform infrastructure as
-- trg_brand_slug_registry/trg_product_slug_registry/trigger_sync_service_slug)
-- and `update_category_updated_at` (sets updated_at on every UPDATE) already
-- exist on the live DB and are intentionally NOT recreated here — see
-- docs/category.md.
