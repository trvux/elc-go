ALTER TABLE verification_tokens DROP COLUMN attempts;
ALTER TABLE verification_tokens DROP COLUMN code;
ALTER TABLE users ALTER COLUMN username SET NOT NULL;
ALTER TABLE users DROP COLUMN google_sub;
