-- New tables — the AI chat feature is entirely Go-owned from the start,
-- nothing here pre-exists in Supabase.

CREATE TABLE IF NOT EXISTS ai_providers (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name              TEXT NOT NULL UNIQUE, -- internal slug, e.g. "deepseek"
    display_name      TEXT NOT NULL,
    base_url          TEXT NOT NULL,
    -- Official pricing page, fetched by cmd/sync-ai-pricing to keep
    -- ai_models.pricing current instead of a one-time hand-typed snapshot.
    pricing_doc_url   TEXT,
    -- AES-256-GCM ciphertext (12-byte nonce prepended) — see
    -- internal/ai/infrastructure/secret_crypto.go. Never decrypted outside
    -- infrastructure; never returned to a client.
    api_key_encrypted BYTEA NOT NULL,
    is_active         BOOLEAN NOT NULL DEFAULT true,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS ai_models (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id        UUID NOT NULL REFERENCES ai_providers(id) ON DELETE CASCADE,
    model_name         TEXT NOT NULL, -- wire model id sent to the API, e.g. "deepseek-chat"
    display_name       TEXT NOT NULL,
    -- 'chat': answers customers, tried in fallback_priority order.
    -- 'classifier': the cheap pre-completion guardrail check.
    role               TEXT NOT NULL DEFAULT 'chat',
    -- Self-describing pricing config (peak price + off_peak_multiplier,
    -- not two independent numbers — see internal/ai/domain's Pricing doc
    -- comment). Admin-editable, kept current by cmd/sync-ai-pricing.
    pricing            JSONB NOT NULL DEFAULT '{}',
    -- Lower tried first, scoped to role — the fallback chain.
    fallback_priority  INT NOT NULL DEFAULT 0,
    is_default         BOOLEAN NOT NULL DEFAULT false,
    is_active          BOOLEAN NOT NULL DEFAULT true,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_ai_models_role CHECK (role IN ('chat', 'classifier')),
    UNIQUE (provider_id, model_name)
);

CREATE INDEX IF NOT EXISTS idx_ai_models_role_priority ON ai_models (role, fallback_priority);

-- visitor_id/user_id reference the same identity split as
-- wishlist/recently-viewed (visitor_id, cookie-issued by
-- httpserver.EnsureVisitorID) plus an optional link to a real account when
-- the visitor happens to be logged in — see the RFC for why login isn't
-- required (would shrink conversation volume, the data this exists to
-- collect). users(id) confirmed UUID PK in internal/auth's migration.
CREATE TABLE IF NOT EXISTS ai_conversations (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    visitor_id UUID NOT NULL,
    user_id    UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_ai_conversations_visitor_id ON ai_conversations (visitor_id);

CREATE TABLE IF NOT EXISTS ai_messages (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES ai_conversations(id) ON DELETE CASCADE,
    role            TEXT NOT NULL,
    content         TEXT NOT NULL,
    -- Set when the guardrail classifier rejected this turn — no
    -- provider_id/model_id/cost below in that case, since no chat
    -- completion was ever called (the whole point of running the
    -- classifier first: rejected turns don't pay for a full completion).
    blocked_reason  TEXT,
    provider_id     UUID REFERENCES ai_providers(id) ON DELETE SET NULL,
    model_id        UUID REFERENCES ai_models(id) ON DELETE SET NULL,
    input_tokens      INT,
    output_tokens     INT,
    cache_hit_tokens  INT,
    cost_usd          NUMERIC(12, 6),
    -- Product slugs the search_products tool surfaced this turn — feeds
    -- the ads/marketing-planning analytics goal, not just support review.
    products_shown    JSONB,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_ai_messages_role CHECK (role IN ('user', 'assistant', 'tool'))
);

CREATE INDEX IF NOT EXISTS idx_ai_messages_conversation_id ON ai_messages (conversation_id);
