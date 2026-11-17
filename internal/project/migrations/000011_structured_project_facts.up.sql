-- Typed, per-project facts so every project carries the same complete data
-- (SEO + GEO: AI engines and Google quote concrete numbers, place and
-- equipment condition, not free text). See docs/project-structured-facts.md.
--
-- Everything is additive and nullable/defaulted, so the previous binary keeps
-- working while this migration runs (deploy.yml migrates before the swap).
-- Constraints are dropped-then-added so the file is re-runnable (migration
-- 000008's bare ADD CONSTRAINT is not).
--
-- province/ward hold the CURRENT administrative names (Vietnam moved to a
-- two-level province/commune system on 2025-07-01). The pre-existing
-- `location` column is kept untouched as the free-text/historical name that
-- the original article used; it is no longer shown once province is filled.
--
-- Equipment condition (new/used) is deliberately NOT a column: it already
-- lives in project_category.condition (28/71 projects have 'used'), and a
-- second copy would drift.
ALTER TABLE projects
    ADD COLUMN IF NOT EXISTS province TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS ward TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS completed_precision TEXT NOT NULL DEFAULT 'unknown',
    ADD COLUMN IF NOT EXISTS client_kind TEXT,
    ADD COLUMN IF NOT EXISTS client_consent_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS client_consent_note TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS indoor_units INTEGER,
    ADD COLUMN IF NOT EXISTS capacity_hp NUMERIC(7,1),
    ADD COLUMN IF NOT EXISTS airflow_m3h INTEGER,
    ADD COLUMN IF NOT EXISTS area_m2 NUMERIC(8,1),
    ADD COLUMN IF NOT EXISTS floors INTEGER,
    ADD COLUMN IF NOT EXISTS duration_days INTEGER,
    ADD COLUMN IF NOT EXISTS warranty_months INTEGER,
    ADD COLUMN IF NOT EXISTS brands TEXT[] NOT NULL DEFAULT '{}'::text[];

ALTER TABLE projects DROP CONSTRAINT IF EXISTS projects_completed_precision_check;
ALTER TABLE projects ADD CONSTRAINT projects_completed_precision_check
    CHECK (completed_precision IN ('day', 'month', 'year', 'unknown'));

ALTER TABLE projects DROP CONSTRAINT IF EXISTS projects_client_kind_check;
ALTER TABLE projects ADD CONSTRAINT projects_client_kind_check
    CHECK (client_kind IS NULL OR client_kind IN ('person', 'business'));

ALTER TABLE projects DROP CONSTRAINT IF EXISTS projects_facts_positive_check;
ALTER TABLE projects ADD CONSTRAINT projects_facts_positive_check CHECK (
    (indoor_units IS NULL OR indoor_units > 0)
    AND (capacity_hp IS NULL OR capacity_hp > 0)
    AND (airflow_m3h IS NULL OR airflow_m3h > 0)
    AND (area_m2 IS NULL OR area_m2 > 0)
    AND (floors IS NULL OR floors > 0)
    AND (duration_days IS NULL OR duration_days > 0)
    AND (warranty_months IS NULL OR warranty_months > 0)
);

-- Raw SQL inserts (hand-run content fixes) must not publish by accident; the
-- Go layer always passes is_published explicitly, so only those are affected.
ALTER TABLE projects ALTER COLUMN is_published SET DEFAULT false;
