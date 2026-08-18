#!/bin/sh
# Refreshes the local dev Postgres (`elc-postgres` container) with a full
# snapshot of production DATA. One-directional: production -> dev only,
# never touches production (pg_dump is read-only there). NOT automatic —
# dev and prod drift apart again between runs, since there's no ongoing
# replication; just re-run this whenever dev needs current data.
#
# Data-only by design: this never touches schema (no DROP SCHEMA, no
# restoring table/function definitions). Earlier versions did a full
# schema+data restore via DROP SCHEMA public CASCADE, which also wiped out
# any in-progress local migrations every time you just wanted fresh data —
# forcing a manual re-apply/merge before the next `git push`. Now schema is
# owned entirely by `make migrate-up` / migrate-all.sh, always — this
# script only ever replaces row data in tables that already exist.
#
# Consequence: dev's schema must already be current (migrations applied)
# before running this, or a table/column that exists in prod's dump but
# not yet in dev will fail to restore. Run `make migrate-up module=...` (or
# migrate-all.sh) first if you're not sure.
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

echo "sync-dev-db-from-prod: dumping production data only (read-only, --disable-triggers keeps FK/trigger order from mattering)..."
# schema_migrations* excluded from the dump itself (not just skipped when
# truncating) — each machine tracks its own applied-migrations state
# locally; overwriting dev's with prod's rows here would desync `migrate`
# from what's actually applied to dev's schema.
ssh "$PROD_HOST" "docker exec $PROD_CONTAINER pg_dump -U $DB_USER -d $DB_NAME --data-only --disable-triggers --exclude-table='schema_migrations*'" > "$DUMP_FILE"

echo "sync-dev-db-from-prod: truncating dev's existing tables (schema untouched — see header comment)..."
TABLES=$(docker exec "$DEV_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -tA -c "
  SELECT string_agg(quote_ident(table_name), ', ')
  FROM information_schema.tables
  WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
    AND table_name NOT LIKE 'schema_migrations%';
")
if [ -z "$TABLES" ]; then
  echo "sync-dev-db-from-prod: no tables found in dev's public schema — run migrations first (migrate-all.sh)." >&2
  exit 1
fi
docker exec "$DEV_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -v ON_ERROR_STOP=1 \
  -c "TRUNCATE TABLE $TABLES RESTART IDENTITY CASCADE;"

echo "sync-dev-db-from-prod: restoring data (a column/table missing here means dev's schema is behind prod's — run migrations first)..."
docker exec -i "$DEV_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -v ON_ERROR_STOP=1 < "$DUMP_FILE" \
  > "$RESTORE_LOG" 2>&1

echo "sync-dev-db-from-prod: done. Row count sanity check:"
docker exec "$DEV_CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -c "
SELECT 'products' t, count(*) FROM products
UNION ALL SELECT 'brands', count(*) FROM brands
UNION ALL SELECT 'categories', count(*) FROM categories;
"

echo "sync-dev-db-from-prod: full restore log at $RESTORE_LOG"
