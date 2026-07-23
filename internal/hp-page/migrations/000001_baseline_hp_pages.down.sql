-- WARNING: never run `migrate-down` for this migration against a shared
-- production DATABASE_URL — it drops the real hp_pages table and all its
-- rows. Deliberately does NOT touch check_slug_conflict()'s function body
-- (see 000001_baseline_hp_pages.up.sql's top-of-file note) — that function
-- is shared, multi-entity infrastructure; this migration only reverts the
-- pieces it itself introduced (table, trigger, its own function, and the
-- slug_registry entity_type list it extended).
BEGIN;

DROP TRIGGER IF EXISTS trg_hp_page_slug_registry ON hp_pages;
DROP FUNCTION IF EXISTS sync_hp_page_slug_registry();

ALTER TABLE slug_registry DROP CONSTRAINT IF EXISTS slug_registry_entity_type_check;
ALTER TABLE slug_registry ADD CONSTRAINT slug_registry_entity_type_check
    CHECK (entity_type = ANY (ARRAY[
        'page'::text, 'product'::text, 'category'::text, 'categories'::text,
        'group'::text, 'brand'::text, 'project'::text, 'project_type'::text,
        'service_group'::text, 'service'::text, 'news_category'::text, 'news'::text
    ]));

DROP TABLE IF EXISTS hp_pages;

COMMIT;
