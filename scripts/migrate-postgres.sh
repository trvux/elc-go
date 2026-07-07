#!/bin/sh
# One-time migration: dump the `public` schema from the old Supabase Postgres
# and restore it into the self-hosted `elc-postgres` container. Idempotent —
# guarded by a sentinel file so re-running deploy never re-imports.
#
# Fixes baked in (discovered manually 2026-07-07, see elc-go conventions):
#   1. pg_dump/pg_restore run via a postgres:17-alpine container to match the
#      source server's major version exactly.
#   2. Supabase installs pgcrypto/uuid-ossp into a schema named `extensions`,
#      but pg_trgm/unaccent into `public` — recreated identically here so
#      schema-qualified references in the dump resolve.
#   3. `immutable_unaccent(text)`'s body calls bare `unaccent('unaccent', $1)`,
#      which fails under pg_restore's empty search_path — patched to the
#      fully-qualified `public.unaccent('public.unaccent'::regdictionary, $1)`
#      before loading.
#   4. RLS `CREATE POLICY ... TO authenticated` statements always error on a
#      vanilla Postgres (no such role) — harmless noise, left to fail; no
#      table has FORCE ROW LEVEL SECURITY, and the connecting role owns every
#      restored object, so RLS never actually applies to it anyway.
#
# Usage: SOURCE_DATABASE_URL=<supabase-url> ./migrate-postgres.sh
# BASE_DIR defaults to the real deploy path on the VPS; overridable so this
# script can be exercised locally against a throwaway directory first.
set -eu

BASE_DIR="${BASE_DIR:-/var/www/elc-go}"
# Lives inside backups/ on purpose — that's the one directory deploy.yml's
# rsync --delete excludes (it never exists in git), so the sentinel survives
# every future deploy instead of being wiped and re-triggering the import.
SENTINEL="$BASE_DIR/backups/.postgres-migrated"
NETWORK="elc-go_default"
CONTAINER="elc-postgres"
DB_USER="elc"
DB_NAME="elc"

if [ -f "$SENTINEL" ]; then
  echo "migrate-postgres: sentinel exists, already migrated — skipping"
  exit 0
fi

if [ -z "${SOURCE_DATABASE_URL:-}" ]; then
  echo "migrate-postgres: SOURCE_DATABASE_URL not set — skipping (nothing to migrate from)"
  exit 0
fi

mkdir -p "$BASE_DIR/backups"

echo "migrate-postgres: dumping public schema from source..."
docker run --rm -v "$BASE_DIR/backups":/backups postgres:17-alpine \
  pg_dump "$SOURCE_DATABASE_URL" -n public --no-owner --no-privileges -Fc -f /backups/cutover.dump

echo "migrate-postgres: converting to plain SQL for patching..."
docker run --rm -v "$BASE_DIR/backups":/backups postgres:17-alpine \
  pg_restore --no-owner --no-privileges -f /backups/cutover.sql /backups/cutover.dump

# See fix #3 above.
sed -i "s/SELECT unaccent('unaccent', \$1)/SELECT public.unaccent('public.unaccent'::regdictionary, \$1)/" \
  "$BASE_DIR/backups"/cutover.sql

echo "migrate-postgres: recreating schema + extensions..."
docker exec "$CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -v ON_ERROR_STOP=1 -c "
DROP SCHEMA IF EXISTS public CASCADE;
CREATE SCHEMA public;
DROP SCHEMA IF EXISTS extensions CASCADE;
CREATE SCHEMA extensions;
CREATE EXTENSION IF NOT EXISTS pgcrypto SCHEMA extensions;
CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\" SCHEMA extensions;
CREATE EXTENSION IF NOT EXISTS pg_trgm SCHEMA public;
CREATE EXTENSION IF NOT EXISTS unaccent SCHEMA public;
ALTER DATABASE $DB_NAME SET search_path TO public, extensions;
"

echo "migrate-postgres: restoring data (errors from Supabase-only RLS policies are expected and harmless)..."
docker run --rm --network "$NETWORK" -e PGPASSWORD="$POSTGRES_PASSWORD" \
  -v "$BASE_DIR/backups":/backups postgres:17-alpine \
  psql -h "$CONTAINER" -U "$DB_USER" -d "$DB_NAME" -v ON_ERROR_STOP=0 -f /backups/cutover.sql \
  > "$BASE_DIR/backups"/cutover-restore.log 2>&1 || true

echo "migrate-postgres: done. Row count sanity check:"
docker exec "$CONTAINER" psql -U "$DB_USER" -d "$DB_NAME" -c "
SELECT 'products' t, count(*) FROM products
UNION ALL SELECT 'contacts', count(*) FROM contacts
UNION ALL SELECT 'projects', count(*) FROM projects;
"

touch "$SENTINEL"
echo "migrate-postgres: sentinel written, will not re-run on future deploys"
