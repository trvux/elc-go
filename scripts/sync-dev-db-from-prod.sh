#!/bin/sh
# Refreshes the local dev Postgres (`elc-postgres` container) with a full
# snapshot of production data. One-directional: production -> dev only,
# never touches production (pg_dump is read-only there). NOT automatic —
# dev and prod drift apart again between runs, since there's no ongoing
# replication; just re-run this whenever dev needs current data.
#
# Same 3 gotchas as scripts/migrate-postgres.sh (the original Supabase
# cutover script) — this dev DB and prod DB both descend from that same
# schema, so the same fixes apply:
#   1. pg_dump/psql must match Postgres major version (17) — done by using
#      each container's own client tools via `docker exec`, never the host
#      machine's own (Homebrew) pg_dump/psql, which may be an older major
#      version and silently produce/read a subtly-incompatible dump.
#   2. pgcrypto/uuid-ossp live in an `extensions` schema, pg_trgm/unaccent
#      live in `public` — both recreated to match before restoring, or
#      restore fails outright (schema conflicts / missing extensions).
#   3. immutable_unaccent(text)'s body calls bare `unaccent('unaccent',
#      $1)`, which fails to resolve under psql's restore-time forced-empty
#      search_path (`SELECT pg_catalog.set_config('search_path', '',
#      false);`, which pg_dump always emits) — patched to the fully-
#      qualified `public.unaccent('public.unaccent'::regdictionary, $1)`
#      before loading. Without this fix, CREATE TABLE products fails (its
#      generated search_vector column calls immutable_unaccent), which
#      then cascades into "relation products does not exist" for every
#      table/policy that references it — looks like dozens of unrelated
#      failures, but it's this one thing.
#
# Requires: passwordless SSH to the production VPS already set up (`ssh
# root@103.179.189.179` must just work — see memory: reference_db_access
# for the DB credentials/environment split this script bridges).
#
# Usage: ./scripts/sync-dev-db-from-prod.sh
set -eu

PROD_HOST="root@103.179.189.179"
PROD_CONTAINER="elc-postgres"
DEV_CONTAINER="elc-postgres"
DB_USER="elc"
DB_NAME="elc"
DUMP_FILE="/tmp/elc_prod_sync_$$.sql"
RESTORE_LOG="/tmp/elc_dev_sync_restore.log"

cleanup() {
  rm -f "$DUMP_FILE"
}
trap cleanup EXIT

echo "sync-dev-db-from-prod: dumping production (read-only, plain SQL)..."
ssh "$PROD_HOST" "docker exec $PROD_CONTAINER pg_dump -U $DB_USER -d $DB_NAME" > "$DUMP_FILE"

echo "sync-dev-db-from-prod: patching immutable_unaccent for empty-search_path safety (fix #3 above)..."
sed -i.bak "s/SELECT unaccent('unaccent', \$1)/SELECT public.unaccent('public.unaccent'::regdictionary, \$1)/" "$DUMP_FILE"
rm -f "$DUMP_FILE.bak"

echo "sync-dev-db-from-prod: recreating schema + extensions in dev (fix #2 above)..."
docker exec "$DEV_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -v ON_ERROR_STOP=1 -c "
DROP SCHEMA IF EXISTS public CASCADE;
CREATE SCHEMA public;
DROP SCHEMA IF EXISTS extensions CASCADE;
CREATE SCHEMA extensions;
GRANT ALL ON SCHEMA public TO $DB_USER;
GRANT ALL ON SCHEMA public TO public;
CREATE EXTENSION IF NOT EXISTS pgcrypto SCHEMA extensions;
CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\" SCHEMA extensions;
CREATE EXTENSION IF NOT EXISTS pg_trgm SCHEMA public;
CREATE EXTENSION IF NOT EXISTS unaccent SCHEMA public;
ALTER DATABASE $DB_NAME SET search_path TO public, extensions;
"

echo "sync-dev-db-from-prod: restoring (RLS 'TO authenticated'/role-related errors are expected and harmless — dev's elc user owns every restored object, so RLS never actually applies to it anyway)..."
docker exec -i "$DEV_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -v ON_ERROR_STOP=0 < "$DUMP_FILE" \
  > "$RESTORE_LOG" 2>&1 || true

echo "sync-dev-db-from-prod: done. Row count sanity check:"
docker exec "$DEV_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -c "
SELECT 'products' t, count(*) FROM products
UNION ALL SELECT 'brands', count(*) FROM brands
UNION ALL SELECT 'categories', count(*) FROM categories;
"

echo "sync-dev-db-from-prod: full restore log at $RESTORE_LOG"
