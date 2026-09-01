CREATE OR REPLACE FUNCTION immutable_unaccent(text) RETURNS text AS $$
    SELECT public.unaccent('unaccent', $1)
$$ LANGUAGE sql IMMUTABLE PARALLEL SAFE STRICT;
