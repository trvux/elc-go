-- Genuinely new table (unlike brand/category/group, which pre-existed in
-- Supabase before Go migrations existed) — this migration is the first
-- real DDL for hp_pages, its trigger, AND the shared check_slug_conflict()
-- function ever committed to a migration file (previously only lived in
-- the live DB / pg_dump backups, see backups/elc_public.sql). Handle with
-- care: check_slug_conflict is shared, multi-entity infrastructure.

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS hp_pages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    slug TEXT NOT NULL,
    image_url TEXT NOT NULL DEFAULT '',
    meta_title TEXT,
    meta_description TEXT,
    order_index INTEGER DEFAULT 0,
    content JSONB,
    -- Generic, not hardcoded to phan_khuc_hp, so this same mechanism is
    -- reusable for a different attribute_definitions.code later with zero
    -- migration — see internal/hp-page/domain/types.go.
    attribute_code TEXT NOT NULL DEFAULT 'phan_khuc_hp',
    attribute_values TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT now(),
    updated_at TIMESTAMPTZ DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS hp_pages_slug_unique_active ON hp_pages (slug) WHERE deleted_at IS NULL;

-- CRITICAL: slug_registry_entity_type_check currently allow-lists
-- ('page','product','category','categories','group','brand','project',
-- 'project_type','service_group','service','news_category','news') and does
-- NOT include 'hp_page' (confirmed via backups/elc_public.sql:1099). Without
-- this ALTER, every INSERT into hp_pages fails identically to the bug
-- fixed in internal/project-type/migrations/000002_drop_broken_legacy_slug_trigger.up.sql.
ALTER TABLE slug_registry DROP CONSTRAINT IF EXISTS slug_registry_entity_type_check;
ALTER TABLE slug_registry ADD CONSTRAINT slug_registry_entity_type_check
    CHECK (entity_type = ANY (ARRAY[
        'page'::text, 'product'::text, 'category'::text, 'categories'::text,
        'group'::text, 'brand'::text, 'project'::text, 'project_type'::text,
        'service_group'::text, 'service'::text, 'news_category'::text,
        'news'::text, 'hp_page'::text
    ]));

CREATE OR REPLACE FUNCTION public.sync_hp_page_slug_registry() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF TG_OP = 'INSERT' THEN
    IF NEW.slug IS NOT NULL AND NEW.slug <> '' AND NEW.deleted_at IS NULL THEN
      PERFORM check_slug_conflict(NEW.slug, NEW.id, 'hp_page');
      INSERT INTO slug_registry (slug, entity_type, entity_id, created_at)
      VALUES (NEW.slug, 'hp_page', NEW.id, NOW())
      ON CONFLICT (slug) DO UPDATE SET deleted_at = NULL, entity_type = EXCLUDED.entity_type, entity_id = EXCLUDED.entity_id, updated_at = NOW();
    END IF;
    RETURN NEW;
  ELSIF TG_OP = 'UPDATE' THEN
    IF NEW.deleted_at IS NOT NULL AND OLD.deleted_at IS NULL THEN
      UPDATE slug_registry SET deleted_at = NEW.deleted_at, updated_at = NOW() WHERE entity_id = OLD.id AND entity_type = 'hp_page';
    ELSIF NEW.deleted_at IS NULL AND OLD.deleted_at IS NOT NULL THEN
      IF NEW.slug IS NOT NULL AND NEW.slug <> '' THEN
        PERFORM check_slug_conflict(NEW.slug, NEW.id, 'hp_page');
        INSERT INTO slug_registry (slug, entity_type, entity_id, created_at) VALUES (NEW.slug, 'hp_page', NEW.id, NOW())
        ON CONFLICT (slug) DO UPDATE SET deleted_at = NULL, entity_type = EXCLUDED.entity_type, entity_id = EXCLUDED.entity_id, updated_at = NOW();
      END IF;
    ELSIF OLD.slug IS DISTINCT FROM NEW.slug AND NEW.deleted_at IS NULL THEN
      IF NEW.slug IS NOT NULL AND NEW.slug <> '' THEN PERFORM check_slug_conflict(NEW.slug, NEW.id, 'hp_page'); END IF;
      UPDATE slug_registry SET deleted_at = NOW(), updated_at = NOW() WHERE entity_id = OLD.id AND entity_type = 'hp_page';
      IF NEW.slug IS NOT NULL AND NEW.slug <> '' THEN
        INSERT INTO slug_registry (slug, entity_type, entity_id, created_at) VALUES (NEW.slug, 'hp_page', NEW.id, NOW())
        ON CONFLICT (slug) DO UPDATE SET deleted_at = NULL, entity_type = EXCLUDED.entity_type, entity_id = EXCLUDED.entity_id, updated_at = NOW();
      END IF;
    ELSIF NEW.deleted_at IS NULL THEN
      UPDATE slug_registry SET updated_at = NOW() WHERE entity_id = OLD.id AND entity_type = 'hp_page';
    END IF;
    RETURN NEW;
  ELSIF TG_OP = 'DELETE' THEN
    DELETE FROM slug_registry WHERE entity_id = OLD.id AND entity_type = 'hp_page';
    RETURN OLD;
  END IF;
END; $$;

CREATE TRIGGER trg_hp_page_slug_registry AFTER INSERT OR DELETE OR UPDATE ON public.hp_pages FOR EACH ROW EXECUTE FUNCTION public.sync_hp_page_slug_registry();

-- Shared function, reproduced verbatim from the live DB (see
-- backups/elc_public.sql:81-118) with exactly ONE new ELSIF branch added
-- before the ELSE fallback. First time this function is captured in a
-- migration file in this repo — see note at top of file.
CREATE OR REPLACE FUNCTION public.check_slug_conflict(p_slug text, p_entity_id uuid, p_entity_type text) RETURNS void
    LANGUAGE plpgsql
    AS $$
DECLARE
  conflicting_record RECORD;
  type_vn TEXT;
  entity_name TEXT := '';
BEGIN
  SELECT entity_type, entity_id INTO conflicting_record FROM slug_registry
  WHERE slug = p_slug AND deleted_at IS NULL AND (entity_id <> p_entity_id OR entity_type <> p_entity_type);
  IF FOUND THEN
    IF conflicting_record.entity_type = 'group' THEN
      type_vn := 'nhóm danh mục'; SELECT name INTO entity_name FROM group_categories WHERE id = conflicting_record.entity_id;
    ELSIF conflicting_record.entity_type = 'category' THEN
      type_vn := 'danh mục sản phẩm'; SELECT name INTO entity_name FROM categories WHERE id = conflicting_record.entity_id;
    ELSIF conflicting_record.entity_type = 'brand' THEN
      type_vn := 'thương hiệu'; SELECT name INTO entity_name FROM brands WHERE id = conflicting_record.entity_id;
    ELSIF conflicting_record.entity_type = 'product' THEN
      type_vn := 'sản phẩm'; SELECT name INTO entity_name FROM products WHERE id = conflicting_record.entity_id;
    ELSIF conflicting_record.entity_type = 'hp_page' THEN
      type_vn := 'trang công suất máy lạnh'; SELECT name INTO entity_name FROM hp_pages WHERE id = conflicting_record.entity_id;
    ELSE
      type_vn := conflicting_record.entity_type;
    END IF;
    RAISE EXCEPTION 'Đường dẫn (slug) "%" đã trùng với % "%" đang hoạt động trong hệ thống. Vui lòng chọn đường dẫn khác.', p_slug, type_vn, COALESCE(entity_name, 'chưa xác định');
  END IF;
END;
$$;
