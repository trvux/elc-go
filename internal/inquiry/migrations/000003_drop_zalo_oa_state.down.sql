-- Reverses 000003: re-adds the zalo_oa_* tables exactly as
-- 000002_zalo_oa_state.up.sql originally defined them.
CREATE TABLE IF NOT EXISTS zalo_oa_tokens (
    id            INT PRIMARY KEY DEFAULT 1,
    access_token  TEXT NOT NULL,
    refresh_token TEXT NOT NULL,
    expires_at    TIMESTAMPTZ NOT NULL,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_zalo_oa_tokens_singleton CHECK (id = 1)
);

CREATE TABLE IF NOT EXISTS zalo_oa_followers (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    zalo_user_id TEXT NOT NULL UNIQUE,
    display_name TEXT,
    is_active    BOOLEAN NOT NULL DEFAULT true,
    followed_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
