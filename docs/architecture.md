# Architecture

## Components

- API validates requests, authenticates clients, rate limits per client, persists jobs, and exposes list/get/cancel/replay operations.
- Workers run a bounded number of handlers concurrently, claim jobs atomically, renew leases, and record success/failure.
- PostgreSQL 16 is the durable source of truth.
- API and worker use separate restricted database roles.
- API and worker expose process-local Prometheus metrics.

## Claim and fencing protocol

A PostgreSQL statement marks expired running jobs at their attempt limit as dead, then claims an eligible row with FOR UPDATE SKIP LOCKED. Claim assigns a random lease token and expiry. Renew, complete, and fail require the current token and a live lease. Cancellation clears the lease; the worker heartbeat detects ownership loss and cancels the handler context.

## Handler registry and concurrency

The worker registry maps a bounded job type name to a Go handler accepting context.Context and JSON payload. The runner starts a configured number of polling loops. The registry is process-local and handlers must be compiled into the worker binary. Payload data is never evaluated as code.

## API security and tracing

Bearer credentials can be configured as one API token or a list of client IDs and tokens. Tokens are compared in constant time. Each authenticated client has a token-bucket limiter in each API process. Job reads, listing, cancellation, replay, and idempotency are scoped to the authenticated client ID. The limiter is local; multi-replica deployments need a gateway or shared limiter.

The HTTP middleware accepts W3C traceparent context, creates a trace ID when absent, returns X-Request-ID, and logs the trace ID. There is no span exporter in this MVP.

## Pagination, cancellation, and replay

List uses stable descending (created_at,id) keyset pagination. Cursors are base64url-encoded opaque values and should be passed unchanged. Cancellation transitions pending/running jobs to cancelled and invalidates the lease. Replay creates a new job row from a terminal source job owned by the same client, with attempts reset and no inherited idempotency key.

## Delivery semantics

Delivery is at-least-once. Handler side effects must be idempotent; exactly-once external effects are not promised. A cancellation request is cooperative and cannot roll back already-issued side effects.

## Database privileges

The migration owner provisions djr_api and djr_worker. The API role can select/insert and update only cancellation/idempotency columns. The worker role can select and update only claim/lease/execution columns. Neither application role can perform DDL or delete rows.

## Remaining production integration

Use an external OpenTelemetry exporter, gateway/shared rate limiter, secret manager, TLS, backup/restore drills, and deployment-specific resource policies before public production use.
