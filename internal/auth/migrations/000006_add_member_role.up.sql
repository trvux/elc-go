-- 'member' is the public-facing role: anyone who signs in via Google or magic
-- link gets this automatically, with zero admin-panel permissions (absent
-- from rolePermissions in internal/auth/domain/permission.go). Ranked below
-- 'user' in roleRank (internal/auth/domain/role.go) — admin-panel access is
-- always a deliberate promotion by an existing admin, never automatic.
ALTER TYPE auth_role ADD VALUE IF NOT EXISTS 'member';
