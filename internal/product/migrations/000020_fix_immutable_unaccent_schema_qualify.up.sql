-- immutable_unaccent's body called bare unaccent('unaccent', $1), relying
-- on the caller's search_path to find it. Production's `elc` database has
-- `ALTER DATABASE elc SET search_path = public, extensions` configured,
-- so this worked fine in normal use — but that setting is database-level
-- config, not schema/data, so pg_dump/pg_restore never captures it.
-- Restoring a backup into a fresh database (default search_path
-- "$user", public) or letting pg_restore run with its own minimal
-- search_path both fail immediately on CREATE TABLE products, because the
-- unqualified unaccent(regdictionary, text) call can't resolve. Found
-- 2026-09-01 by actually restore-testing a backup into a scratch DB — see
-- docs/backup-restore.md.
--
-- Schema-qualifying the inner call makes this function correct regardless
-- of caller search_path, which is the general best practice for anything
-- referenced inside a function body anyway.
CREATE OR REPLACE FUNCTION immutable_unaccent(text) RETURNS text AS $$
    SELECT public.unaccent('unaccent', $1)
$$ LANGUAGE sql IMMUTABLE PARALLEL SAFE STRICT;
