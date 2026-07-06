include .env

# golang-migrate tracks applied versions in a single `schema_migrations`
# table by default — shared across the whole database. Since every module
# restarts its own migration numbering at 000001, that collides silently
# (module B's 000001 looks "already applied" because module A's 000001 ran).
# Each module gets its own tracking table to keep them independent.
migrations_table = schema_migrations_$(subst -,_,$(module))

migrate-up:
	@migrate -path internal/$(module)/migrations -database "$(DATABASE_URL)?x-migrations-table=$(migrations_table)" up

migrate-down:
	@migrate -path internal/$(module)/migrations -database "$(DATABASE_URL)?x-migrations-table=$(migrations_table)" down 1

migrate-create:
	@migrate create -ext sql -dir internal/$(module)/migrations -seq $(name)

migrate-force:
	@migrate -path internal/$(module)/migrations -database "$(DATABASE_URL)?x-migrations-table=$(migrations_table)" force $(version)

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
