-- Baseline for tables that already exist in Supabase — IF NOT EXISTS so this
-- is a no-op against the real DB. Only useful for spinning up a fresh
-- scratch Postgres for local testing. Column types/defaults/constraints
-- confirmed against the live DB via `\d projects` / `\d project_category` /
-- `\d project_service` before writing this migration.
--
-- projects.slug is a PLAIN UNIQUE constraint (`projects_slug_key`) — unlike
-- brand/group/category's partial "unique among non-deleted rows" index, this
-- one applies even to soft-deleted rows. That's why Create() must resurrect
-- a soft-deleted row on slug reuse rather than plain-INSERT — the constraint
-- itself would reject the insert otherwise. See docs/project.md.
--
-- project_category's PK is composite (project_id, category_id, condition):
-- the same project can be attached to the same category twice, once per
-- condition (new/used). product_condition enum already created by
-- catalog's baseline; DO block here is a harmless no-op if it already exists
-- (matches internal/catalog/migrations' own guard).
--
-- The triggers `trg_sync_project_slug` (syncs into the shared slug_registry
-- table) and `update_projects_modtime` (sets updated_at on every UPDATE)
-- already exist on the live DB and are intentionally NOT recreated here —
-- see docs/project.md.
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

DO $$ BEGIN
    CREATE TYPE product_condition AS ENUM ('new', 'used');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category_id UUID NOT NULL,
    title TEXT NOT NULL,
    description JSONB NOT NULL DEFAULT '{}'::jsonb,
    images TEXT[] NOT NULL DEFAULT '{}'::text[],
    is_published BOOLEAN DEFAULT true,
    order_index INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT now(),
    slug TEXT NOT NULL UNIQUE,
    updated_at TIMESTAMPTZ DEFAULT now(),
    is_featured BOOLEAN DEFAULT false,
    deleted_at TIMESTAMPTZ,
    meta_title TEXT,
    meta_description TEXT,
    project_type_id UUID REFERENCES project_type(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS project_category (
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    category_id UUID NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT timezone('utc'::text, now()),
    condition product_condition NOT NULL DEFAULT 'new'::product_condition,
    PRIMARY KEY (project_id, category_id, condition)
);

CREATE INDEX IF NOT EXISTS idx_project_category_category_id ON project_category (category_id);
CREATE INDEX IF NOT EXISTS idx_project_category_project_id ON project_category (project_id);

CREATE TABLE IF NOT EXISTS project_service (
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    service_id UUID NOT NULL REFERENCES services(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT timezone('utc'::text, now()),
    PRIMARY KEY (project_id, service_id)
);
