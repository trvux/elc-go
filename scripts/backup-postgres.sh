#!/bin/sh
# Nightly backup for the self-hosted Postgres — the Docker volume protects
# against container/rebuild churn but NOT against disk failure, a bad
# `docker volume rm`, or the VPS itself disappearing. Pushes an off-box copy
# to the same Cloudflare R2 account already used for image uploads (no new
# service to pay for/manage), via R2's S3-compatible API.
#
# Retention: BACKUP_RETENTION_DAYS days on both local and R2 (default 30).
# R2 is pruned by parsing the date out of the filename, not object metadata
# — this account's R2 token is object read/write only, not bucket-admin, so
# a native R2 lifecycle rule isn't available (confirmed 2026-09-01:
# GetBucketLifecycleConfiguration returns AccessDenied). If the token ever
# gets bucket-admin scope, prefer a real lifecycle rule over this — it stays
# correct even on a night this script fails to run.
set -eu

# BASE_DIR defaults to the real deploy path on the VPS; overridable so this
# script can be exercised locally against a throwaway directory first.
BASE_DIR="${BASE_DIR:-/var/www/elc-go}"
BACKUP_RETENTION_DAYS="${BACKUP_RETENTION_DAYS:-30}"

# Pull just the R2_* values out of .env by hand instead of `. .env` —
# SMTP_PASSWORD and a couple other values in that file contain unquoted
# spaces, which breaks a plain shell `source`.
ENV_FILE="$BASE_DIR/.env"
R2_ACCESS_KEY_ID=$(grep '^R2_ACCESS_KEY_ID=' "$ENV_FILE" | cut -d= -f2-)
R2_SECRET_ACCESS_KEY=$(grep '^R2_SECRET_ACCESS_KEY=' "$ENV_FILE" | cut -d= -f2-)
R2_BUCKET_NAME=$(grep '^R2_BUCKET_NAME=' "$ENV_FILE" | cut -d= -f2-)
R2_ACCOUNT_ID=$(grep '^R2_ACCOUNT_ID=' "$ENV_FILE" | cut -d= -f2-)

R2_ENDPOINT="https://${R2_ACCOUNT_ID}.r2.cloudflarestorage.com"

BACKUP_DIR="$BASE_DIR/backups"
STAMP=$(date +%Y%m%d-%H%M%S)
FILE="postgres-${STAMP}.dump"

mkdir -p "$BACKUP_DIR"

# Full database, no -n public schema restriction: confirmed 2026-09-01 by
# an actual restore-into-scratch-db test that `-n public` silently drops
# the `extensions` schema (pgcrypto, uuid-ossp live there) and, more
# critically, extension definitions themselves aren't captured under -n at
# all — restoring that dump into a fresh database failed outright
# (products.search_vector's generated column calls immutable_unaccent(),
# which needs the `unaccent` text search dictionary, which needs `unaccent`
# to have been CREATE EXTENSION'd first). `elc` is superuser so a full dump
# restores its own CREATE EXTENSION statements with no extra permission
# step needed — the schema restriction was buying nothing.
docker exec elc-postgres pg_dump -U elc -d elc --no-owner --no-privileges -Fc \
  -f "/tmp/${FILE}"
docker cp "elc-postgres:/tmp/${FILE}" "${BACKUP_DIR}/${FILE}"

# Integrity check before this dump is trusted anywhere: a file pg_restore
# can't even list the table of contents of is worse than no backup at all
# — silent corruption you only discover the night you actually need to
# restore. Fail loudly here instead, same night, while it's still cheap to
# just re-run the backup.
if ! docker exec elc-postgres pg_restore --list "/tmp/${FILE}" > /dev/null 2>&1; then
  docker exec elc-postgres rm -f "/tmp/${FILE}"
  echo "backup-postgres: FAILED integrity check on ${FILE} — not uploading, not pruning anything" >&2
  exit 1
fi
docker exec elc-postgres rm "/tmp/${FILE}"

docker run --rm \
  -e AWS_ACCESS_KEY_ID="$R2_ACCESS_KEY_ID" \
  -e AWS_SECRET_ACCESS_KEY="$R2_SECRET_ACCESS_KEY" \
  -e AWS_DEFAULT_REGION=auto \
  -v "${BACKUP_DIR}:/backups" \
  amazon/aws-cli s3 cp "/backups/${FILE}" "s3://${R2_BUCKET_NAME}/postgres-backups/${FILE}" \
  --endpoint-url "$R2_ENDPOINT"

# Local retention.
find "$BACKUP_DIR" -maxdepth 1 -name 'postgres-*.dump' -mtime "+${BACKUP_RETENTION_DAYS}" -delete

# R2 retention — same window. GNU date (VPS/Linux) vs BSD date (exercising
# this locally on a Mac) take the "N days ago" flag differently.
CUTOFF=$(date -d "-${BACKUP_RETENTION_DAYS} days" +%Y%m%d 2>/dev/null || date -v-"${BACKUP_RETENTION_DAYS}"d +%Y%m%d)
docker run --rm \
  -e AWS_ACCESS_KEY_ID="$R2_ACCESS_KEY_ID" \
  -e AWS_SECRET_ACCESS_KEY="$R2_SECRET_ACCESS_KEY" \
  -e AWS_DEFAULT_REGION=auto \
  amazon/aws-cli s3 ls "s3://${R2_BUCKET_NAME}/postgres-backups/" \
  --endpoint-url "$R2_ENDPOINT" \
  | awk '{print $4}' \
  | grep -oE '^postgres-[0-9]{8}-[0-9]{6}\.dump$' \
  | while read -r name; do
      fdate=$(echo "$name" | sed -E 's/postgres-([0-9]{8})-.*/\1/')
      if [ "$fdate" -lt "$CUTOFF" ]; then
        docker run --rm \
          -e AWS_ACCESS_KEY_ID="$R2_ACCESS_KEY_ID" \
          -e AWS_SECRET_ACCESS_KEY="$R2_SECRET_ACCESS_KEY" \
          -e AWS_DEFAULT_REGION=auto \
          amazon/aws-cli s3 rm "s3://${R2_BUCKET_NAME}/postgres-backups/${name}" \
          --endpoint-url "$R2_ENDPOINT"
      fi
    done

echo "backup-postgres: uploaded ${FILE} to r2://${R2_BUCKET_NAME}/postgres-backups/ (retention: ${BACKUP_RETENTION_DAYS}d, integrity-checked)"
