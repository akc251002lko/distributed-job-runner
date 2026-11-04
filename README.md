# Distributed Job Runner

A durable PostgreSQL-backed distributed job processing service written in Go.

## Capabilities

- Durable HTTP job submission and inspection.
- PostgreSQL 16 coordination with atomic SKIP LOCKED claims.
- Idempotency keys with conflict detection.
- Renewable leases, fencing tokens, recovery, exponential retries, and dead-lettering.
- Bounded concurrent worker execution and an extensible in-process handler registry.
- Cursor pagination, cooperative cancellation, and replay of terminal jobs.
- Per-client bearer credentials and token-bucket API rate limits.
- W3C Trace Context propagation and request correlation IDs.
- Separate least-privilege API and worker database roles.
- Structured logs, health/readiness endpoints, Prometheus metrics, Docker Compose, non-root containers, CI, and Dependabot.

## What This Repository Is For

This repository is a reference implementation of a durable, PostgreSQL-backed job-processing service. It is intended for applications that need to enqueue background work over HTTP and process it asynchronously with multiple worker processes, retry and recovery behavior, tenant isolation, and operational visibility. Execution is at-least-once, so application handlers must be designed for idempotency.

## How To Use

For local development, review `.env.example`, ensure Go 1.26+ and Docker Compose are available, then start the stack with:

```bash
docker compose up --build
```

Check API health and readiness at `/healthz` and `/readyz`. Submit work to `POST /v1/jobs` using the bearer-authentication configuration described below. Apply database migrations and provision the separate API and worker roles before running outside the local development setup. See the HTTP API, authentication, database roles, and operational-limit sections for endpoint details and production requirements.

## Components

- cmd/api — authenticated HTTP API.
- cmd/worker — concurrent worker process.
- internal/api, internal/store, internal/worker, internal/config, internal/observability — application packages.
- migrations/ — ordered PostgreSQL migrations.
- ops/postgres/roles.sql — provisions separate application database roles.

PostgreSQL is the durable source of truth. API and worker processes can be scaled independently.

## HTTP API

All /v1 endpoints require bearer authentication when API_TOKEN or API_KEYS is configured. Production requires authentication. /healthz, /readyz, and /metrics are operational endpoints.

| Method | Endpoint | Purpose |
|---|---|---|
| GET | /healthz | Process health |
| GET | /readyz | PostgreSQL readiness |
| GET | /metrics | Prometheus metrics |
| POST | /v1/jobs | Enqueue a job |
| GET | /v1/jobs/{id} | Get a job |
| GET | /v1/jobs?limit=50&status=pending | Cursor-paginated job list |
| POST | /v1/jobs/{id}/cancel | Cancel pending/running job |
| POST | /v1/jobs/{id}/replay | Replay a terminal job |

### Enqueue

POST /v1/jobs accepts type, payload, optional idempotency_key, max_attempts, and optional RFC3339 run_at.

    curl -sS -X POST http://localhost:8080/v1/jobs \
      -H 'Authorization: Bearer YOUR_TOKEN' \
      -H 'Content-Type: application/json' \
      -H 'Idempotency-Key: invoice-123' \
      -d '{"type":"noop","payload":{"invoice_id":"123"}}'

Reusing a key with different job data returns HTTP 409.

### List and pagination

GET /v1/jobs?limit=25&status=dead returns items and next_cursor. Pass the cursor unchanged as the next request's cursor query parameter. Page size is 1–100. Supported statuses: pending, running, succeeded, dead, cancelled.

### Cancel and replay

Cancellation changes pending or running jobs to cancelled and invalidates an active lease. A running handler receives cancellation when its heartbeat detects lease loss; it must honor context.Context. Cancellation cannot undo external side effects.

Replay is allowed only for succeeded, dead, or cancelled jobs. It creates a new job ID, resets attempts, schedules it immediately, and does not reuse the source idempotency key.

## Authentication and rate limiting

- Set API_TOKEN for one shared bearer credential.
- Or set API_KEYS as comma-separated client_id=token entries. Each token must contain at least 32 characters. Client IDs are tenant boundaries: jobs can only be listed, read, cancelled, or replayed by their owning client. Idempotency keys are scoped per client, and each client receives an independent PostgreSQL-backed rate-limit bucket. PostgreSQL row-level security enforces ownership using transaction-local client identity.
- API_RATE_LIMIT_RPS and API_RATE_LIMIT_BURST configure the PostgreSQL-backed token bucket (defaults 10 and 20).
- Rate limits are stored in PostgreSQL and shared across API replicas. If PostgreSQL is unavailable, protected API requests fail closed with HTTP 503.
- Health/readiness/metrics are not protected by bearer authentication; restrict their network exposure.

## Worker handlers and concurrency

internal/worker.Registry supports registering typed Go handlers that receive a context and JSON payload. Register application handlers in the worker binary before constructing the runner. Do not execute arbitrary payload code.

WORKER_CONCURRENCY controls maximum simultaneous handlers per worker process (default 4; allowed 1–256). Scale worker replicas independently. Handlers must be idempotent, honor cancellation, and bound their own resource use.

## Database roles and migrations

Apply migrations in order as a database owner:

    psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f migrations/000001_init.sql
    psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f migrations/000002_job_cancellation.sql
    psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f migrations/000003_client_scoping.sql
    psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f migrations/000004_api_rate_limits.sql

Provision separate API and worker roles using strong unique passwords:

    psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -v api_password="$API_DB_PASSWORD" -v worker_password="$WORKER_DB_PASSWORD" -f ops/postgres/roles.sql

Configure API_DATABASE_URL and WORKER_DATABASE_URL separately with distinct database role usernames. Production mode rejects the shared DATABASE_URL fallback and requires encrypted PostgreSQL connections (`sslmode=require`, `verify-ca`, or `verify-full`). Prefer `verify-full` where certificate and hostname verification are available. API role can select/insert and cancel jobs under row-level security; worker role can select and update execution state across tenants. Keep migration credentials separate and never use the database owner in application processes.

## Local development

Requirements: Go 1.26+ and Docker Compose.

1. Review local-only credentials in .env.example.
2. Start with docker compose up --build.
3. Check http://localhost:8080/healthz, http://localhost:8080/readyz, and worker metrics at http://localhost:9090/metrics.

Commands: make test, make test-race, make vet, make build, make migrate-up, make provision-roles.

## Delivery guarantees and operational limits

Execution is at-least-once, not exactly-once. A worker may perform a side effect and crash before committing success. Use downstream idempotency keys or transactional outbox patterns for external effects.

Trace Context is propagated and request IDs are logged, but this MVP does not export spans to an OpenTelemetry collector. Rate limiting is PostgreSQL-backed and shared across replicas. Handler registration is compiled into the worker binary. Cancellation is cooperative and cannot terminate a handler that ignores context cancellation.

Compose credentials are for local development only. Production requires TLS, secret management, private networking, backups, monitoring, resource limits, migration/rollback procedures, and a security review. See SECURITY.md and docs/runbook.md.

## Validation

GitHub Actions provisions PostgreSQL 16 and verifies migrations, application-role grants, formatting, go vet, race-enabled tests, Go builds, and both container images.

## Community & Support

For official project announcements, release and repository updates, technical discussions, questions, and collaboration, join the [Telegram community](https://t.me/githubakc251002lko).

## License

Apache License 2.0. See LICENSE.