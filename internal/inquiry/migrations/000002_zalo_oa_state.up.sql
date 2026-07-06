-- New tables (no pre-existing Supabase equivalent). zalo_oa_tokens is a
-- singleton row (id fixed at 1, enforced by CHECK) holding the current
-- OAuth token pair for the company's Zalo OA app — Zalo rotates the refresh
-- token on every use, so both columns are overwritten together on every
-- refresh (see internal/inquiry/infrastructure/zalo_token_refresher.go).
CREATE TABLE IF NOT EXISTS zalo_oa_tokens (
    id            INT PRIMARY KEY DEFAULT 1,
    access_token  TEXT NOT NULL,
    refresh_token TEXT NOT NULL,
    expires_at    TIMESTAMPTZ NOT NULL,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_zalo_oa_tokens_singleton CHECK (id = 1)
);

-- zalo_oa_followers records staff Zalo accounts that have followed the OA
-- and messaged it at least once (captured via the OA webhook's follow /
-- message-from-user events) — the only recipients Zalo allows the OA to
-- push a message back to.
CREATE TABLE IF NOT EXISTS zalo_oa_followers (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    zalo_user_id TEXT NOT NULL UNIQUE,
    display_name TEXT,
    is_active    BOOLEAN NOT NULL DEFAULT true,
    followed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
