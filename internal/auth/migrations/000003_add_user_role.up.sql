-- 'user' is the lowest admin-panel role (content work only, no account
-- management) — see internal/auth/domain/permission.go for what it grants.
ALTER TYPE auth_role ADD VALUE IF NOT EXISTS 'user';
