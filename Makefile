.PHONY: test test-race fmt vet build migrate-up provision-roles compose-up compose-down

test:
	go test ./...

test-race:
	go test -race ./...

fmt:
	gofmt -w ./cmd ./internal

vet:
	go vet ./...

build:
	mkdir -p bin
	go build -trimpath -o bin/djr-api ./cmd/api
	go build -trimpath -o bin/djr-worker ./cmd/worker

migrate-up:
	psql "$(DATABASE_URL)" -v ON_ERROR_STOP=1 -f migrations/000001_init.sql
	psql "$(DATABASE_URL)" -v ON_ERROR_STOP=1 -f migrations/000002_job_cancellation.sql

provision-roles:
	psql "$(DATABASE_URL)" -v ON_ERROR_STOP=1 -v api_password="$(API_DB_PASSWORD)" -v worker_password="$(WORKER_DB_PASSWORD)" -f ops/postgres/roles.sql

compose-up:
	docker compose up --build

compose-down:
	docker compose down
