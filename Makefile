include .env

migrate-up:
	@migrate -path internal/$(module)/migrations -database "$(DATABASE_URL)" up

migrate-down:
	@migrate -path internal/$(module)/migrations -database "$(DATABASE_URL)" down 1

migrate-create:
	@migrate create -ext sql -dir internal/$(module)/migrations -seq $(name)

migrate-force:
	@migrate -path internal/$(module)/migrations -database "$(DATABASE_URL)" force $(version)

run:
	@go run ./cmd/server
