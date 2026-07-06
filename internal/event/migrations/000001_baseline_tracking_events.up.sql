-- Baseline for a table that already exists in Supabase — IF NOT EXISTS so
-- this is a no-op against the real DB. Columns/nullability confirmed via
-- `\d tracking_events` before writing this migration.
--
-- This table predates this Go module (created directly in Supabase for an
-- earlier, now-removed client-side analytics implementation — see commit
-- 04a53c7 in elc-tem). Its two RLS policies ("Allow public insert" WITH
-- CHECK true, "Allow authenticated select" using auth.role()='authenticated')
-- are Supabase-client-era artifacts that don't apply to this module's pgx
-- connection (same as every other table in this DB) — left as-is
-- intentionally, not something this migration introduces or needs to fix.
--
-- event_name is a fixed, typed taxonomy at the Go layer (see
-- internal/event/domain/types.go) modeled on Google Analytics 4's
-- recommended-events naming (verb_noun, e.g. `view_item`) rather than
-- arbitrary strings — the previous implementation's ad-hoc event names
-- (`scroll_50`, etc.) were a real source of drift.
CREATE TABLE IF NOT EXISTS tracking_events (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT timezone('utc'::text, now()),
    event_name     TEXT NOT NULL,
    event_category TEXT,
    event_label    TEXT,
    page_path      TEXT,
    session_id     TEXT,
    metadata       JSONB
);

CREATE INDEX IF NOT EXISTS idx_tracking_events_lookup
    ON tracking_events (event_name, event_category, event_label);
CREATE INDEX IF NOT EXISTS idx_tracking_events_created_at ON tracking_events (created_at);
