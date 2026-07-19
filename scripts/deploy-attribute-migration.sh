#!/bin/sh
# One-time data migration for the structured attribute system (see
# internal/attribute, docs on catalog specs redesign): seeds
# attribute_definitions and maps the legacy free-text products.specs onto
# product_attribute_values. Run AFTER migrate-all.sh (needs the
# attribute_definitions/product_attribute_values tables to already exist)
# and BEFORE swapping in the new elc-go container.
#
# Sentinel-gated like migrate-postgres.sh, NOT "safe to re-run every
# deploy" as originally written — found 2026-07-19: cmd/migrate-specs-to-
# attributes-v2 upserts (ON CONFLICT ... DO UPDATE) rather than skipping
# existing rows, so a value manually corrected after this ran once (e.g.
# the 130-product WebSearch-verified data audit, or any future admin edit
# to an attribute value that happens to share a product+definition with
# something the legacy specs parser also produces) would silently get
# reverted back to the raw-parsed version on the very next deploy. This
# migration is conceptually the same shape as migrate-postgres.sh's
# Supabase cutover — convert legacy representation once, then the new
# structured table (product_attribute_values) becomes the one source of
# truth going forward, edited directly via the admin UI, never re-derived
# from products.specs again.
#
# Usage: DATABASE_URL=postgresql://elc:<password>@postgres:5432/elc ./deploy-attribute-migration.sh
set -eu

BASE_DIR="${BASE_DIR:-/var/www/elc-go}"
# Lives inside backups/ on purpose — that's the one directory deploy.yml's
# rsync --delete excludes (it never exists in git), so the sentinel survives
# every future deploy instead of being wiped and re-triggering the migration.
SENTINEL="$BASE_DIR/backups/.attribute-values-migrated"
NETWORK="elc-go_default"
GO_IMAGE="golang:1.26"

if [ -f "$SENTINEL" ]; then
  echo "deploy-attribute-migration: sentinel exists, already migrated — skipping"
  exit 0
fi

if [ -z "${DATABASE_URL:-}" ]; then
  echo "deploy-attribute-migration: DATABASE_URL not set" >&2
  exit 1
fi

echo "deploy-attribute-migration: snapshotting products.specs (pre-migration backup table)"
docker exec -i elc-postgres psql -U elc -d elc < "$BASE_DIR/scripts/backup-products-specs.sql"

echo "deploy-attribute-migration: seeding attribute_definitions"
for seed in "$BASE_DIR"/scripts/seed-attribute-definitions*.sql; do
  echo "deploy-attribute-migration: seeding from $(basename "$seed")"
  docker exec -i elc-postgres psql -U elc -d elc < "$seed"
done

echo "deploy-attribute-migration: mapping legacy specs -> product_attribute_values"
# cmd/migrate-specs-to-attributes-v2 is cross-database by design (built for
# dev's workflow: read the old flat schema from a separate reference DB,
# write structured data into the redesigned target DB) — it wants
# SOURCE_DATABASE_URL/TARGET_DATABASE_URL, not v1's single DATABASE_URL, and
# queries `SELECT id, category_id, specs FROM products WHERE deleted_at IS
# NULL` against whatever SOURCE_DATABASE_URL points at. On production there
# is no separate old-schema database — migrate-all.sh already dropped
# products.specs from this same DB moments earlier — so point SOURCE at a
# same-database shim schema exposing the pre-migration backup table
# (products_specs_pre_migration_backup, captured above) under the shape v2
# expects, via search_path. Found + fixed via a cutover rehearsal against a
# real production backup (2026-07-19); verified end-to-end there before
# writing this permanently — v2 doesn't need this shim on dev, only on any
# environment where products.specs is already gone by the time this runs.
docker exec -i elc-postgres psql -U elc -d elc -v ON_ERROR_STOP=1 -c "
CREATE SCHEMA IF NOT EXISTS legacy_specs_shim;
CREATE OR REPLACE VIEW legacy_specs_shim.products AS
SELECT product_id AS id, category_id, specs, NULL::timestamptz AS deleted_at
FROM products_specs_pre_migration_backup;
"

docker run --rm --network "$NETWORK" \
  -v "$BASE_DIR":/app -w /app \
  -e SOURCE_DATABASE_URL="${DATABASE_URL}?sslmode=disable&options=-csearch_path%3Dlegacy_specs_shim" \
  -e TARGET_DATABASE_URL="${DATABASE_URL}?sslmode=disable" \
  "$GO_IMAGE" go run ./cmd/migrate-specs-to-attributes-v2

docker exec -i elc-postgres psql -U elc -d elc -c "DROP SCHEMA legacy_specs_shim CASCADE;"

mkdir -p "$BASE_DIR/backups"
touch "$SENTINEL"
echo "deploy-attribute-migration: done, sentinel written, will not re-run on future deploys"
