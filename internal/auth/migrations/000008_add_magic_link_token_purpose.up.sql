-- 000007 added the code/attempts columns for magic-link tokens but never
-- extended auth_token_purpose itself, so every verification_tokens insert
-- with purpose='magic_link' (application.RequestMagicLink) fails with
-- "invalid input value for enum auth_token_purpose" — surfaced to clients as
-- a bare 500 from POST /auth/magic-link.
ALTER TYPE auth_token_purpose ADD VALUE IF NOT EXISTS 'magic_link';
