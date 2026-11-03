# Distributed Job Runner

A durable, PostgreSQL-backed distributed job processing system written in Go.

> **Status:** Early development. Review the known limitations below before using this outside local development.

## Current capabilities

- Versioned HTTP API to enqueue and inspect jobs.
- PostgreSQL 16 as the durable source of truth.
- Concurrent-safe claims using `FOR UPDATE SKIP LOCKED`.
- Idempotency keys and bounded retry scheduling with exponential backoff and jitter.
- Renewable worker leases with fencing tokens; expired leases can be recovered safely.
- Dead-letter transition after retry exhaustion, including a worker that disappears during its final attempt.
- Structured JSON logs, health/readiness endpoints, and dependency-free Prometheus metrics.
- Docker Compose local stack and GitHub Actions CI.

## Architecture

- `cmd/api` — HTTP API for submitting and inspecting jobs.
- `cmd/worker` — worker process that claims and executes jobs.
- `internal/api`, `internal/store`, `internal/worker`, `internal/config`, `internal/observability` — application components.
- `migrations/` — versioned PostgreSQL schema.

The API and worker are independent processes. PostgreSQL coordinates workers; no in-memory queue is authoritative.

## API

- `GET /healthz` — process health.
- `GET /readyz` — PostgreSQL readiness.
- `GET /metrics` — Prometheus text metrics for the API process.
- `POST /v1/jobs` — enqueue a job.
- `GET /v1/jobs/{id}` — inspect a job.

Example:

```sh
curl -sS -X POST http://localhost:8080/v1/jobs \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: sample-001' \
  -d '{"type":"noop","payload":{"source":"readme"}}'
```

If `API_TOKEN` is configured, send `Authorization: Bearer <token>`. Reusing an idempotency key with a different job type, payload, or attempt limit returns HTTP `409 Conflict`. The worker metrics endpoint is available at `http://localhost:9090/metrics` in the local Compose stack.

## Delivery semantics

The system provides **at-least-once execution**, not exactly-once side effects. A worker can perform an external side effect and crash before recording success. Handlers must therefore be idempotent or use downstream idempotency. Lease tokens prevent stale workers from changing job state after ownership changes; lease renewal keeps long-running handlers alive while they retain ownership.

## Local development

Requirements: Go 1.22+ and Docker Compose.

1. Copy `.env.example` to `.env` if running binaries outside Compose.
2. Start the stack with `docker compose up --build`.
3. Check `http://localhost:8080/healthz` and `http://localhost:8080/readyz`.

Useful commands: `make test`, `make test-race`, `make vet`, `make build`, and `make compose-down`.

## Security

Do not commit credentials, production data, or real secrets. `.env` is ignored by Git. The Compose credentials and unauthenticated local API defaults are for development only. Production requires API authentication, TLS, private networking, secret management, restricted database roles, backups, and monitoring.

Report suspected vulnerabilities privately as described in [SECURITY.md](SECURITY.md).

## Known MVP limitations

The initial worker registers only short `noop` and `echo` handlers. A production handler registry, bounded concurrent execution, graceful cancellation hooks for arbitrary handlers, pagination, job cancellation/replay, per-client authorization/rate limits, tracing, and production-grade database roles remain future work. The current lease heartbeat is active during handler execution, but handler-specific idempotency remains mandatory.

## License

Apache License 2.0. See [LICENSE](LICENSE).
