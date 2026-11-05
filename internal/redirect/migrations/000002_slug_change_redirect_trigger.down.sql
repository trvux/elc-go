DO $$
DECLARE
    t RECORD;
BEGIN
    FOR t IN SELECT c.relname FROM pg_trigger g JOIN pg_class c ON c.oid = g.tgrelid
             WHERE g.tgname = 'trg_slug_redirect'
    LOOP
        EXECUTE format('DROP TRIGGER IF EXISTS trg_slug_redirect ON %I', t.relname);
    END LOOP;
END $$;
DROP FUNCTION IF EXISTS elc_record_slug_redirect();
