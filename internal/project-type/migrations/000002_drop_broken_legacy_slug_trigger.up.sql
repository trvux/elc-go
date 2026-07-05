-- Bug fix, not a project-type-specific schema change: `trg_sync_service_type_slug`
-- (a legacy trigger left over from when this table was named service_type)
-- inserts into slug_registry with entity_type='service_type', but
-- slug_registry_entity_type_check no longer allows that value (only
-- 'project_type' is in the allowed list — confirmed via `\d slug_registry`).
-- This currently makes EVERY insert into project_type fail outright,
-- including from the existing Next.js/Supabase admin — reproduced with a
-- raw `INSERT INTO project_type (name, slug) VALUES (...)` via psql,
-- independent of any Go code. See docs/project-type.md.
--
-- `trigger_sync_project_type_slug` (the newer, correctly-named trigger) has
-- already fully superseded this one — it maintains the exact same
-- slug_registry row shape but with the correct entity_type='project_type' —
-- so the fix is to drop the broken trigger and its now-unused function
-- rather than patch it, avoiding two triggers doing the same job. Audited
-- slug_registry for any pre-existing entity_type='service_type' rows first
-- (zero found) and every currently-active project_type row already has a
-- correct entity_type='project_type' row (16 registry rows for 13 active
-- project types, including historical/soft-deleted ones) — no data repair
-- needed, this is a pure DDL fix.
BEGIN;

DROP TRIGGER IF EXISTS trg_sync_service_type_slug ON project_type;
DROP FUNCTION IF EXISTS sync_service_type_slug_registry();

COMMIT;
