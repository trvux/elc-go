-- Recreates the empty structure only. The dropped rows are not recoverable from here;
-- use a backup (db-dumps/prod-services-BEFORE-service-line-2026-11-06.sql.gz).
CREATE TABLE IF NOT EXISTS service_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    image_url TEXT,
    meta_title TEXT,
    meta_description TEXT,
    is_featured BOOLEAN DEFAULT false,
    order_index INTEGER DEFAULT 0,
    category_ids UUID[] DEFAULT '{}'::uuid[],
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    deleted_at TIMESTAMPTZ
);
ALTER TABLE services ADD COLUMN IF NOT EXISTS group_id UUID REFERENCES service_groups(id) ON DELETE SET NULL;
