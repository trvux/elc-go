-- 308 redirects managed from the admin UI. nginx reads this table (scripts/
-- nginx-live/live.py regenerates elc-redirects.conf every 2 minutes), so a
-- row saved here is live without a deploy.
--   source: 'manual' (typed in admin), 'auto' (created by the slug-change
--           trigger in migration 000002), 'import' (legacy redirects moved
--           from the old hand-built nginx file).
CREATE TABLE IF NOT EXISTS redirects (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    from_path  TEXT NOT NULL,
    to_path    TEXT NOT NULL,
    is_active  BOOLEAN NOT NULL DEFAULT true,
    source     TEXT NOT NULL DEFAULT 'manual',
    note       TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT redirects_from_path_key UNIQUE (from_path),
    CONSTRAINT chk_redirects_source CHECK (source IN ('manual', 'auto', 'import')),
    CONSTRAINT chk_redirects_from_path CHECK (from_path LIKE '/%' AND from_path <> '/'),
    CONSTRAINT chk_redirects_no_self CHECK (from_path <> to_path)
);

-- Repointing ("everything that redirected to X now goes to Y") filters on to_path.
CREATE INDEX IF NOT EXISTS idx_redirects_to_path ON redirects (to_path);
