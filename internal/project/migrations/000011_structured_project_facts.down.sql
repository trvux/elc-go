-- DESTRUCTIVE: drops the typed facts. Back up the projects table first
-- (pg_dump -t projects) — rolling back loses everything entered since 000011.
ALTER TABLE projects ALTER COLUMN is_published SET DEFAULT true;

ALTER TABLE projects DROP CONSTRAINT IF EXISTS projects_facts_positive_check;
ALTER TABLE projects DROP CONSTRAINT IF EXISTS projects_client_kind_check;
ALTER TABLE projects DROP CONSTRAINT IF EXISTS projects_completed_precision_check;

ALTER TABLE projects
    DROP COLUMN IF EXISTS brands,
    DROP COLUMN IF EXISTS warranty_months,
    DROP COLUMN IF EXISTS duration_days,
    DROP COLUMN IF EXISTS floors,
    DROP COLUMN IF EXISTS area_m2,
    DROP COLUMN IF EXISTS airflow_m3h,
    DROP COLUMN IF EXISTS capacity_hp,
    DROP COLUMN IF EXISTS indoor_units,
    DROP COLUMN IF EXISTS client_consent_note,
    DROP COLUMN IF EXISTS client_consent_at,
    DROP COLUMN IF EXISTS client_kind,
    DROP COLUMN IF EXISTS completed_precision,
    DROP COLUMN IF EXISTS ward,
    DROP COLUMN IF EXISTS province;
