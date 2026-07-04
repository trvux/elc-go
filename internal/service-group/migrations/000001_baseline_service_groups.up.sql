-- Baseline for a table that already exists in Supabase — IF NOT EXISTS so
-- this is a no-op against the real DB. Only useful for spinning up a fresh
-- scratch Postgres for local testing.
--
-- The trigger `trigger_sync_service_group_slug` (syncs into the shared
-- `slug_registry` table) already exists on the live DB and is NOT
-- recreated here on purpose — it's shared platform infrastructure used by
-- several tables (services, categories, brands, ...), not owned by this
-- module alone.
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS service_groups (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
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
