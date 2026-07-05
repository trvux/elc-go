-- Baseline for tables that already exist in Supabase — IF NOT EXISTS so this
-- is a no-op against the real DB. Column types/defaults/constraints
-- confirmed against the live DB via `\d project_type` / `\d project_type_category`
-- before writing this migration.
--
-- project_type.slug is a PLAIN UNIQUE constraint (`service_type_slug_unique`
-- — legacy name, the table was renamed from service_type to project_type at
-- some point but constraint/trigger/index names were never renamed) —
-- unlike brand/group/category's partial "unique among non-deleted rows"
-- index, this one applies even to soft-deleted rows. That's why Create()
-- must resurrect a soft-deleted row on slug reuse rather than plain-INSERT.
-- See docs/project-type.md.
--
-- project_type_category's PK is composite (project_type_id, category_id).
--
-- Two separate triggers already sync project_type into the shared
-- slug_registry table on the live DB — `trg_sync_service_type_slug_registry`
-- (legacy name, entity_type='service_type') and
-- `trigger_sync_project_type_slug` (entity_type='project_type'). Both run on
-- every INSERT/UPDATE/DELETE; this predates the Go migration and is
-- intentionally left as-is (not something this migration introduces or
-- fixes) — see docs/project-type.md. `update_service_type_updated_at`
-- (also legacy-named) sets updated_at on every UPDATE and is likewise not
-- recreated here.
CREATE TABLE IF NOT EXISTS project_type (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT timezone('utc'::text, now()),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT timezone('utc'::text, now()),
    deleted_at TIMESTAMPTZ,
    slug TEXT NOT NULL UNIQUE,
    image TEXT,
    meta_title TEXT,
    meta_description TEXT,
    is_featured BOOLEAN DEFAULT false,
    order_index INTEGER DEFAULT 0
);

CREATE TABLE IF NOT EXISTS project_type_category (
    project_type_id UUID NOT NULL REFERENCES project_type(id) ON DELETE CASCADE,
    category_id UUID NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT timezone('utc'::text, now()),
    PRIMARY KEY (project_type_id, category_id)
);

CREATE INDEX IF NOT EXISTS idx_service_type_category_category_id ON project_type_category (category_id);
CREATE INDEX IF NOT EXISTS idx_service_type_category_service_type_id ON project_type_category (project_type_id);
