-- Baseline for a table that already exists in Supabase — IF NOT EXISTS so
-- this is a no-op against the real DB. Only useful for spinning up a fresh
-- scratch Postgres for local testing.
--
-- The trigger `trg_brand_slug_registry` (syncs into the shared
-- `slug_registry` table) already exists on the live DB and is NOT
-- recreated here on purpose — same shared platform infrastructure as
-- `trigger_sync_service_group_slug` (see service-group's baseline).
--
-- Note the slug uniqueness here is a PARTIAL index (deleted_at IS NULL),
-- not a plain UNIQUE column like service_groups.slug — a soft-deleted brand
-- does not block a new brand from reusing its slug. `name`, on the other
-- hand, IS a full unique constraint (applies even to soft-deleted rows) —
-- see docs/brand.md for the gotcha this causes.
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS brands (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    slug TEXT NOT NULL,
    logo_url TEXT NOT NULL DEFAULT '',
    meta_title TEXT,
    meta_description TEXT,
    is_featured BOOLEAN DEFAULT false,
    order_index INTEGER DEFAULT 0,
    content JSONB,
    faq JSONB DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS brands_slug_unique_active ON brands (slug) WHERE deleted_at IS NULL;
