-- google_sub links an account to a verified Google identity (nullable — a
-- magic-link-only account never has one). Unique so two Google accounts can
-- never collide onto the same row.
ALTER TABLE users ADD COLUMN google_sub TEXT UNIQUE;

-- Google/magic-link accounts never choose a username, and the unique
-- constraint must still allow any number of them side by side — Postgres
-- treats every NULL as distinct under UNIQUE, so dropping NOT NULL (and
-- storing NULL, never '') is enough; no need to touch the constraint itself.
ALTER TABLE users ALTER COLUMN username DROP NOT NULL;

-- verification_tokens is repurposed from invite/password-reset to magic-link
-- only (see internal/auth/domain/verification_token.go) — code is the 6-digit
-- number emailed alongside the link, attempts counts wrong guesses so a link
-- can be locked out before its TTL expires instead of allowing the full
-- window to brute-force the code. role/invited_by/user_id are left in place,
-- unused, rather than risk a destructive column drop.
ALTER TABLE verification_tokens ADD COLUMN code TEXT;
ALTER TABLE verification_tokens ADD COLUMN attempts INT NOT NULL DEFAULT 0;
