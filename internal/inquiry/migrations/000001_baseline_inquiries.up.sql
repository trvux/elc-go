-- New table (does not pre-exist in Supabase, confirmed via `\dt` before
-- writing this). product_id/project_id/service_id reference products(id),
-- projects(id), services(id) — all UUID PKs, confirmed via `\d products` /
-- `\d projects` / `\d services` against the live DB.
CREATE TABLE IF NOT EXISTS inquiries (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          VARCHAR(255) NOT NULL,
    phone         VARCHAR(30) NOT NULL,
    email         VARCHAR(255),
    message       TEXT,
    product_id    UUID REFERENCES products(id) ON DELETE SET NULL,
    project_id    UUID REFERENCES projects(id) ON DELETE SET NULL,
    service_id    UUID REFERENCES services(id) ON DELETE SET NULL,
    status        VARCHAR(20) NOT NULL DEFAULT 'new',
    internal_note TEXT,
    source_ip     VARCHAR(64),
    user_agent    TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_inquiry_single_entity CHECK (
        (product_id IS NOT NULL)::int + (project_id IS NOT NULL)::int + (service_id IS NOT NULL)::int <= 1
    ),
    CONSTRAINT chk_inquiry_status CHECK (status IN ('new', 'contacted', 'converted', 'closed'))
);

CREATE INDEX IF NOT EXISTS idx_inquiries_status ON inquiries (status);
CREATE INDEX IF NOT EXISTS idx_inquiries_created_at ON inquiries (created_at DESC);
