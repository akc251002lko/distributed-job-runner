# Distributed Job Runner

A durable, PostgreSQL-backed distributed job processing system written in Go.

> **Status:** MVP implementation complete; not production-hardened. Review the known limitations below before deployment.

## Current capabilities

- Versioned HTTP API to enqueue and inspect jobs.
- PostgreSQL 16 as the durable source of truth.
- Concurrent-safe claims using `FOR UPDATE SKIP LOCKED`.
- Idempotency keys; conflicting reuse returns HTTP 409.
- Bounded retry scheduling with exponential backoff and jitter.
- Renewable worker leases with fencing tokens and safe expired-lease recovery.
- Dead-letter transition after retry exhaustion, including abandoned final attempts.
- Structured JSON logs, health/readiness endpoints, and dependency-free Prometheus metrics.
- Docker Compose local stack, non-root container images, GitHub Actions CI, and Dependabot updates.

## Architecture

- `cmd/api` — HTTP API for submitting and inspecting jobs.
- `cmd/worker` — worker process that claims and executes jobs.
- `internal/api`, `internal/store`, `internal/worker`, `internal/config`, `internal/observability` — application components.
- `migrations/` — PostgreSQL schema.

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

If `API_TOKEN` is configured, send `Authorization: Bearer <token>`. Production mode requires a token of at least 32 characters. Reusing an idempotency key with a different job type, payload, or attempt limit returns HTTP `409 Conflict`. The worker metrics endpoint is available at `http://localhost:9090/metrics` in the local Compose stack.

## Delivery semantics

The system provides **at-least-once execution**, not exactly-once side effects. A worker can perform an external side effect and crash before recording success. Handlers must therefore be idempotent or use downstream idempotency. Lease tokens prevent stale workers from changing job state after ownership changes; lease renewal keeps long-running handlers alive while they retain ownership.

## Local development

Requirements: Go 1.22+ and Docker Compose.

1. Copy `.env.example` to `.env` if running binaries outside Compose.
2. Start the stack with `docker compose up --build`.
3. Check `http://localhost:8080/healthz` and `http://localhost:8080/readyz`.

Useful commands: `make test`, `make test-race`, `make vet`, `make build`, and `make compose-down`.

## Security

Do not commit credentials, production data, or real secrets. `.env` is ignored by Git. Compose credentials and unauthenticated local API defaults are for development only. Production requires API authentication, TLS, private networking, secret management, restricted database roles, backups, and monitoring.

Report suspected vulnerabilities privately as described in [SECURITY.md](SECURITY.md).

## Known MVP limitations

The initial worker registers only short `noop` and `echo` demonstration handlers. A production handler registry, bounded concurrent execution, job listing/pagination, job cancellation/replay, per-client authorization/rate limits, tracing, and production-grade database roles remain future work. Handlers must honor context cancellation and make external side effects idempotent.

## Validation

GitHub Actions validates the project against PostgreSQL 16 with module integrity checks, schema migration, formatting, `go vet`, race-enabled tests, Go builds, and API/worker container builds.

## License

Apache License 2.0. See [LICENSE](LICENSE).
