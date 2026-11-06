-- Contract step 3 of the service restructure (docs/rfc-service-restructure.md in elc-temp).
-- Services are grouped by `line`; the service-group module, its admin page and every
-- reader of services.group_id are gone (code deployed first). Drops the column, the
-- table and the slug registry rows that pointed at it.
ALTER TABLE services DROP COLUMN IF EXISTS group_id;
DELETE FROM slug_registry WHERE entity_type = 'service_group';
DROP TABLE IF EXISTS service_groups;
