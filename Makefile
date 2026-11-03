.PHONY: test test-race fmt vet build migrate-up compose-up compose-down

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

compose-up:
	docker compose up --build

compose-down:
	docker compose down
