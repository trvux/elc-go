#!/bin/sh
# Nightly backup for the self-hosted Postgres — the Docker volume protects
# against container/rebuild churn but NOT against disk failure, a bad
# `docker volume rm`, or the VPS itself disappearing. Pushes an off-box copy
# to the same Cloudflare R2 account already used for image uploads (no new
# service to pay for/manage), via R2's S3-compatible API.
set -eu

# BASE_DIR defaults to the real deploy path on the VPS; overridable so this
# script can be exercised locally against a throwaway directory first.
BASE_DIR="${BASE_DIR:-/var/www/elc-go}"

# Pull just the R2_* values out of .env by hand instead of `. .env` —
# SMTP_PASSWORD and a couple other values in that file contain unquoted
# spaces, which breaks a plain shell `source`.
ENV_FILE="$BASE_DIR/.env"
R2_ACCESS_KEY_ID=$(grep '^R2_ACCESS_KEY_ID=' "$ENV_FILE" | cut -d= -f2-)
R2_SECRET_ACCESS_KEY=$(grep '^R2_SECRET_ACCESS_KEY=' "$ENV_FILE" | cut -d= -f2-)
R2_BUCKET_NAME=$(grep '^R2_BUCKET_NAME=' "$ENV_FILE" | cut -d= -f2-)
R2_ACCOUNT_ID=$(grep '^R2_ACCOUNT_ID=' "$ENV_FILE" | cut -d= -f2-)

BACKUP_DIR="$BASE_DIR/backups"
STAMP=$(date +%Y%m%d-%H%M%S)
FILE="postgres-${STAMP}.dump"

mkdir -p "$BACKUP_DIR"

docker exec elc-postgres pg_dump -U elc -d elc -n public --no-owner --no-privileges -Fc \
  -f "/tmp/${FILE}"
docker cp "elc-postgres:/tmp/${FILE}" "${BACKUP_DIR}/${FILE}"
docker exec elc-postgres rm "/tmp/${FILE}"

docker run --rm \
  -e AWS_ACCESS_KEY_ID="$R2_ACCESS_KEY_ID" \
  -e AWS_SECRET_ACCESS_KEY="$R2_SECRET_ACCESS_KEY" \
  -e AWS_DEFAULT_REGION=auto \
  -v "${BACKUP_DIR}:/backups" \
  amazon/aws-cli s3 cp "/backups/${FILE}" "s3://${R2_BUCKET_NAME}/postgres-backups/${FILE}" \
  --endpoint-url "https://${R2_ACCOUNT_ID}.r2.cloudflarestorage.com"

# Keep 14 days of local copies (the R2 copies are the real retention; this is
# just so a same-day restore doesn't need a network round trip).
find "$BACKUP_DIR" -maxdepth 1 -name 'postgres-*.dump' -mtime +14 -delete

echo "backup-postgres: uploaded ${FILE} to r2://${R2_BUCKET_NAME}/postgres-backups/"
