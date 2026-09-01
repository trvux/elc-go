-- 000020 schema-qualified the function name (public.unaccent(...)) but left
-- the dictionary argument as a bare 'unaccent' literal relying on
-- search_path to resolve it to the text search dictionary object. Turns
-- out pg_restore's session doesn't pick up a target database's configured
-- default search_path the way a normal psql/app connection does (verified
-- 2026-09-01: `ALTER DATABASE ... SET search_path = public, extensions`
-- is visible via `SHOW search_path` on a fresh psql connection, but
-- pg_restore still failed with `text search dictionary "unaccent" does
-- not exist` on the same target). Casting the dictionary name itself with
-- an explicit schema prefix removes the last search_path dependency —
-- verified working even with search_path forced down to just pg_catalog.
CREATE OR REPLACE FUNCTION immutable_unaccent(text) RETURNS text AS $$
    SELECT public.unaccent('public.unaccent'::regdictionary, $1)
$$ LANGUAGE sql IMMUTABLE PARALLEL SAFE STRICT;
