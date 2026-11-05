-- When the slug of ANY public content row changes, record old URL -> new URL in
-- `redirects` automatically. Done in the database (not in each Go module) so it
-- covers every content type and every code path that edits a slug (admin form,
-- script, manual SQL) and cannot be forgotten when a new module is added.
--
-- Keeps the table flat and loop-free:
--   * the new URL is live, so it must not itself redirect anywhere (delete such row);
--   * anything that already pointed at the old URL now points at the new one;
--   * renaming back (A -> B, then B -> A) removes the A -> B row instead of looping.
CREATE OR REPLACE FUNCTION elc_record_slug_redirect() RETURNS trigger AS $$
DECLARE
    prefix   TEXT := TG_ARGV[0];
    old_path TEXT;
    new_path TEXT;
BEGIN
    IF NEW.slug IS NULL OR NEW.slug = '' OR OLD.slug IS NULL OR OLD.slug = '' THEN
        RETURN NEW;
    END IF;
    -- a soft-deleted row is not a live URL, nothing to preserve
    IF to_jsonb(NEW)->>'deleted_at' IS NOT NULL THEN
        RETURN NEW;
    END IF;

    old_path := prefix || OLD.slug;
    new_path := prefix || NEW.slug;

    DELETE FROM redirects WHERE from_path = new_path;
    UPDATE redirects SET to_path = new_path, updated_at = now() WHERE to_path = old_path;
    INSERT INTO redirects (from_path, to_path, source, note)
    VALUES (old_path, new_path, 'auto', 'Slug đổi trên bảng ' || TG_TABLE_NAME)
    ON CONFLICT (from_path) DO UPDATE
        SET to_path = EXCLUDED.to_path, is_active = true, source = 'auto',
            note = EXCLUDED.note, updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- URL prefix of each table's public pages — same mapping as scripts/nginx-live/live.py.
DO $$
DECLARE
    t RECORD;
BEGIN
    FOR t IN SELECT * FROM (VALUES
        ('branches',         '/thong-tin/'),
        ('brands',           '/san-pham/'),
        ('categories',       '/san-pham/'),
        ('group_categories', '/san-pham/'),
        ('hp_pages',         '/san-pham/'),
        ('news',             '/tin-tuc/'),
        ('pages',            '/'),
        ('products',         '/san-pham/'),
        ('project_type',     '/du-an/'),
        ('projects',         '/du-an/'),
        ('services',         '/dich-vu/')
    ) AS v(tbl, prefix)
    LOOP
        IF to_regclass('public.' || t.tbl) IS NOT NULL THEN
            EXECUTE format('DROP TRIGGER IF EXISTS trg_slug_redirect ON %I', t.tbl);
            EXECUTE format(
                'CREATE TRIGGER trg_slug_redirect AFTER UPDATE OF slug ON %I '
                'FOR EACH ROW WHEN (OLD.slug IS DISTINCT FROM NEW.slug) '
                'EXECUTE FUNCTION elc_record_slug_redirect(%L)', t.tbl, t.prefix);
        END IF;
    END LOOP;
END $$;
