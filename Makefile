include .env

# golang-migrate tracks applied versions in a single `schema_migrations`
# table by default — shared across the whole database. Since every module
# restarts its own migration numbering at 000001, that collides silently
# (module B's 000001 looks "already applied" because module A's 000001 ran).
# Each module gets its own tracking table to keep them independent.
migrations_table = schema_migrations_$(subst -,_,$(module))

# golang-migrate's Postgres driver defaults to sslmode=require and hard-fails
# ("SSL is not enabled on the server") against our self-hosted postgres:17-
# alpine container, which has no TLS cert configured — unlike the app's own
# pgx connection, which defaults to "prefer" and silently falls back to
# plaintext. Explicit sslmode=disable here only affects the `migrate` CLI's
# connection, not the running app. Revisit if DATABASE_URL ever points at a
# TLS-terminated Postgres again (e.g. a managed provider).
migrate_ssl = sslmode=disable

migrate-up:
	@migrate -path internal/$(module)/migrations -database "$(DATABASE_URL)?x-migrations-table=$(migrations_table)&$(migrate_ssl)" up

migrate-down:
	@migrate -path internal/$(module)/migrations -database "$(DATABASE_URL)?x-migrations-table=$(migrations_table)&$(migrate_ssl)" down 1

migrate-create:
	@migrate create -ext sql -dir internal/$(module)/migrations -seq $(name)

migrate-force:
	@migrate -path internal/$(module)/migrations -database "$(DATABASE_URL)?x-migrations-table=$(migrations_table)&$(migrate_ssl)" force $(version)

# Applies pending migrations for every module that has a migrations/
# directory, in one shot. Idempotent — golang-migrate only applies versions
# newer than each module's own schema_migrations_<module> table (see
# `migrations_table` above), so running this against an already-up-to-date
# DB is a safe no-op. Use this locally after `git pull` brings in migration
# files you haven't applied yet; the deploy pipeline runs the equivalent via
# scripts/migrate-all.sh.
migrate-up-all:
	@set -e; \
	for dir in internal/*/migrations; do \
		mod=$$(basename $$(dirname $$dir)); \
		table=schema_migrations_$$(echo $$mod | tr '-' '_'); \
		echo "==> $$mod"; \
		migrate -path $$dir -database "$(DATABASE_URL)?x-migrations-table=$$table&$(migrate_ssl)" up; \
	done

run:
	@air

# Creates or resets the password of one admin account. Values come from the
# environment at call time only — never hardcode real ADMIN_* values here or
# in any committed file. Example:
#   ADMIN_USERNAME=... ADMIN_EMAIL=... ADMIN_PASSWORD=... ADMIN_NAME=... ADMIN_PHONE=... make seed-admin
seed-admin:
	@go run ./cmd/seed-admin

# One-time Zalo OA authorization (see cmd/zalo-authorize's doc comment) —
# only needs to be run once after the Zalo App is created; the background
# token refresher keeps it fresh afterward. Example:
#   ZALO_OA_APP_ID=... ZALO_OA_APP_SECRET=... ZALO_OA_REDIRECT_URI=... make zalo-authorize
zalo-authorize:
	@go run ./cmd/zalo-authorize
