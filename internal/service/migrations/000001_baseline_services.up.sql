-- Baseline for a table that already exists in Supabase — IF NOT EXISTS so
-- this is a no-op against the real DB. Only useful for spinning up a fresh
-- scratch Postgres for local testing.
--
-- `trigger_sync_service_slug` (shared slug_registry sync) already exists on
-- the live DB and is intentionally not recreated here — see contact/service-group.
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS services (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    group_id UUID REFERENCES service_groups(id) ON DELETE SET NULL,
    category_id UUID REFERENCES categories(id) ON DELETE SET NULL,
    original_price BIGINT,
    sale_price BIGINT,
    discount_percent INTEGER,
    price_display_text TEXT,
    labels TEXT[],
    description TEXT,
    content JSONB,
    image TEXT,
    meta_title TEXT,
    meta_description TEXT,
    is_featured BOOLEAN DEFAULT false,
    is_published BOOLEAN DEFAULT true,
    order_index INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    deleted_at TIMESTAMPTZ
);
