-- Restores the exact broken trigger/function this migration removed, for
-- rollback parity only — re-applying this down migration reintroduces the
-- bug described in 000002_drop_broken_legacy_slug_trigger.up.sql (every
-- insert into project_type will fail again).
BEGIN;

CREATE OR REPLACE FUNCTION sync_service_type_slug_registry()
RETURNS trigger
LANGUAGE plpgsql
SECURITY DEFINER
AS $function$
BEGIN
  IF TG_OP = 'INSERT' THEN
    IF NEW.slug IS NOT NULL AND NEW.slug <> '' AND NEW.deleted_at IS NULL THEN
      INSERT INTO slug_registry (slug, entity_type, entity_id, created_at)
      VALUES (NEW.slug, 'service_type', NEW.id, NOW())
      ON CONFLICT (slug) DO UPDATE SET deleted_at = NULL, entity_type = EXCLUDED.entity_type, entity_id = EXCLUDED.entity_id, updated_at = NOW();
    END IF;
    RETURN NEW;

  ELSIF TG_OP = 'UPDATE' THEN
    IF NEW.deleted_at IS NOT NULL AND OLD.deleted_at IS NULL THEN
      UPDATE slug_registry SET deleted_at = NEW.deleted_at, updated_at = NOW() WHERE entity_id = OLD.id AND entity_type = 'service_type';

    ELSIF NEW.deleted_at IS NULL AND OLD.deleted_at IS NOT NULL THEN
      IF NEW.slug IS NOT NULL AND NEW.slug <> '' THEN
        INSERT INTO slug_registry (slug, entity_type, entity_id, created_at)
        VALUES (NEW.slug, 'service_type', NEW.id, NOW())
        ON CONFLICT (slug) DO UPDATE SET deleted_at = NULL, entity_type = EXCLUDED.entity_type, entity_id = EXCLUDED.entity_id, updated_at = NOW();
      END IF;

    ELSIF OLD.slug IS DISTINCT FROM NEW.slug AND NEW.deleted_at IS NULL THEN
      UPDATE slug_registry SET deleted_at = NOW(), updated_at = NOW() WHERE entity_id = OLD.id AND entity_type = 'service_type';
      IF NEW.slug IS NOT NULL AND NEW.slug <> '' THEN
        INSERT INTO slug_registry (slug, entity_type, entity_id, created_at)
        VALUES (NEW.slug, 'service_type', NEW.id, NOW())
        ON CONFLICT (slug) DO UPDATE SET deleted_at = NULL, entity_type = EXCLUDED.entity_type, entity_id = EXCLUDED.entity_id, updated_at = NOW();
      END IF;

    ELSIF NEW.deleted_at IS NULL THEN
      UPDATE slug_registry SET updated_at = NOW() WHERE entity_id = OLD.id AND entity_type = 'service_type';
    END IF;
    RETURN NEW;

  ELSIF TG_OP = 'DELETE' THEN
    DELETE FROM slug_registry WHERE entity_id = OLD.id AND entity_type = 'service_type';
    RETURN OLD;
  END IF;
  RETURN NULL;
END;
$function$;

CREATE TRIGGER trg_sync_service_type_slug
AFTER INSERT OR DELETE OR UPDATE ON project_type
FOR EACH ROW EXECUTE FUNCTION sync_service_type_slug_registry();

COMMIT;
