-- Logs every message a shopper types into the AI chat finder (see
-- product/application/chat_search.go's ChatSearchProducts and the
-- frontend's ProductChatFinder.tsx) — the raw text itself is the data
-- point (real pain points/purchase intent in the shopper's own words),
-- logged regardless of which internal path actually answered it (a fresh
-- search, a comparison follow-up, a criterion/ranking question, or an
-- off-topic/purchase-intent message routed straight to Zalo — see `kind`).
-- No FK to any product/session table — visitor_id is the same anonymous,
-- server-issued cookie recently_viewed_items already uses (see
-- internal/platform/httpserver's EnsureVisitorID), not a real account.
CREATE TABLE IF NOT EXISTS chat_log_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    visitor_id TEXT NOT NULL,
    message TEXT NOT NULL,
    kind TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_chat_log_entries_created_at ON chat_log_entries (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_chat_log_entries_visitor_id ON chat_log_entries (visitor_id, created_at DESC);
