-- New tables — unlike other modules, nothing here already exists in
-- Supabase. Supabase's built-in auth.users stays untouched and unused by
-- this module; this is a fresh, Go-owned schema from the start.

CREATE TYPE auth_role AS ENUM ('admin', 'super_admin');
CREATE TYPE auth_user_status AS ENUM ('active', 'disabled');
CREATE TYPE auth_token_purpose AS ENUM ('invite', 'password_reset');

-- There is no public registration: the only way a row is ever inserted here
-- is through accept-invite (application.AcceptInvite), which requires a
-- valid, unexpired, unconsumed invite token from verification_tokens.
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role auth_role NOT NULL DEFAULT 'admin',
    status auth_user_status NOT NULL DEFAULT 'active',
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Backs both the admin-invite flow and forgot-password. Only token_hash
-- (never the raw token) is stored, so a leaked row can't be replayed.
CREATE TABLE IF NOT EXISTS verification_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    purpose auth_token_purpose NOT NULL,
    token_hash TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL,
    role auth_role,
    invited_by UUID REFERENCES users(id),
    user_id UUID REFERENCES users(id),
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_verification_tokens_hash ON verification_tokens(token_hash);

-- Refresh-token sessions. Access tokens (JWT) are stateless and never
-- stored — only refresh tokens need a row, so logout and password-reset can
-- actually revoke them instead of waiting out an access-token TTL.
CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    token_hash TEXT NOT NULL UNIQUE,
    user_agent TEXT,
    ip_address TEXT,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_sessions_hash ON sessions(token_hash);
CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);
