#!/bin/sh
# One-off data migration for the structured attribute system (see
# internal/attribute, docs on catalog specs redesign): seeds
# attribute_definitions and maps the legacy free-text products.specs onto
# product_attribute_values. Run AFTER migrate-all.sh (needs the
# attribute_definitions/product_attribute_values tables to already exist)
# and BEFORE swapping in the new elc-go container.
#
# Idempotent end-to-end, safe to re-run on every future deploy:
#   - backup-products-specs.sql: CREATE TABLE IF NOT EXISTS, only ever
#     captures the true pre-migration snapshot once.
#   - seed-attribute-definitions.sql: ON CONFLICT DO NOTHING per definition.
#   - cmd/migrate-specs-to-attributes: ON CONFLICT DO NOTHING per
#     (product_id, attribute_definition_id).
# None of this touches products.specs itself — the legacy column is never
# cleared, only read from, so it stays available as a fallback/audit trail
# regardless of how many times this runs.
#
# Usage: DATABASE_URL=postgresql://elc:<password>@postgres:5432/elc ./deploy-attribute-migration.sh
set -eu

BASE_DIR="${BASE_DIR:-/var/www/elc-go}"
NETWORK="elc-go_default"
GO_IMAGE="golang:1.26"

if [ -z "${DATABASE_URL:-}" ]; then
  echo "deploy-attribute-migration: DATABASE_URL not set" >&2
  exit 1
fi

echo "deploy-attribute-migration: snapshotting products.specs (pre-migration backup table)"
docker exec -i elc-postgres psql -U elc -d elc < "$BASE_DIR/scripts/backup-products-specs.sql"

echo "deploy-attribute-migration: seeding attribute_definitions"
docker exec -i elc-postgres psql -U elc -d elc < "$BASE_DIR/scripts/seed-attribute-definitions.sql"

echo "deploy-attribute-migration: mapping legacy specs -> product_attribute_values"
docker run --rm --network "$NETWORK" \
  -v "$BASE_DIR":/app -w /app \
  -e DATABASE_URL="${DATABASE_URL}?sslmode=disable" \
  "$GO_IMAGE" go run ./cmd/migrate-specs-to-attributes

echo "deploy-attribute-migration: done"
