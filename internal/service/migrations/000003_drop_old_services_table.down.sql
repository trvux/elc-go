-- Structural rollback only — restores the table shape, not the dropped data.
-- Row data (if ever needed) lives in the pg_dump backup taken before the
-- corresponding up migration ran, kept outside git.
CREATE TABLE IF NOT EXISTS old_services (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    title             text NOT NULL,
    slug              text NOT NULL,
    content           jsonb NOT NULL DEFAULT '{}',
    image             text NOT NULL DEFAULT '',
    is_published      boolean NOT NULL DEFAULT false,
    order_index       integer NOT NULL DEFAULT 0,
    created_at        timestamptz DEFAULT now(),
    updated_at        timestamptz DEFAULT now(),
    deleted_at        timestamptz,
    meta_title        text,
    meta_description  text,
    CONSTRAINT old_services_slug_key UNIQUE (slug)
);
