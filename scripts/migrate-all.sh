#!/bin/sh
# Applies pending migrations for every internal/*/migrations directory
# against the running `elc-postgres` container. Run this BEFORE swapping in
# a new elc-go container on deploy — so the new code never starts against a
# schema it hasn't been migrated for yet (this was the actual root cause of
# "deploy Go, site errors" incidents: nothing in the deploy pipeline ever
# applied per-module migrations to production; migrate-postgres.sh only ever
# ran the one-time Supabase cutover, gated by its own sentinel).
#
# Uses the official migrate/migrate image via `docker run` rather than
# assuming a `migrate` CLI is installed on the VPS host — same reasoning as
# migrate-postgres.sh's use of `docker run` for pg_dump/pg_restore. Joined
# to the compose network so it can reach Postgres by service name.
#
# Idempotent: each module tracks its own applied versions in
# schema_migrations_<module> (see Makefile's `migrations_table`), so
# re-running against an already-up-to-date DB is a safe no-op.
#
# sslmode=disable: golang-migrate's Postgres driver defaults to
# sslmode=require and hard-fails ("SSL is not enabled on the server")
# against our self-hosted postgres:17-alpine container (no TLS cert
# configured) — unlike the app's own pgx connection, which defaults to
# "prefer" and silently falls back to plaintext. Confirmed by running this
# script against the local dev DB before wiring it into deploy.yml.
#
# Usage: DATABASE_URL=postgresql://elc:<password>@postgres:5432/elc ./migrate-all.sh
set -eu

BASE_DIR="${BASE_DIR:-/var/www/elc-go}"
NETWORK="elc-go_default"
MIGRATE_IMAGE="migrate/migrate:v4.18.1"
MIGRATE_SSL="sslmode=disable"

if [ -z "${DATABASE_URL:-}" ]; then
  echo "migrate-all: DATABASE_URL not set" >&2
  exit 1
fi

for dir in "$BASE_DIR"/internal/*/migrations; do
  mod=$(basename "$(dirname "$dir")")
  table="schema_migrations_$(echo "$mod" | tr '-' '_')"
  echo "migrate-all: $mod"
  docker run --rm --network "$NETWORK" \
    -v "$dir":/migrations \
    "$MIGRATE_IMAGE" \
    -path=/migrations -database "${DATABASE_URL}?x-migrations-table=${table}&${MIGRATE_SSL}" up
done

echo "migrate-all: done"
