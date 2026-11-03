# Architecture

## Components

- **API:** validates HTTP requests, applies optional bearer-token authentication, and persists jobs.
- **Worker:** polls PostgreSQL, atomically claims one eligible job, executes a registered job type, and records success or failure.
- **PostgreSQL 16:** durable source of truth for job state, retry scheduling, idempotency, and leases.

The API and worker are independent processes. No in-memory queue is authoritative.

## Claim and fencing protocol

A single PostgreSQL statement selects an eligible row with `FOR UPDATE SKIP LOCKED`, then updates it with a random lease token, worker ID, expiry, and incremented attempt count. Completion and failure updates require the exact lease token and running status. A stale worker therefore cannot overwrite state after a newer worker has reclaimed the job.

Expired running jobs can be reclaimed. This is **at-least-once** delivery: a worker may perform an external side effect and crash before recording success. Handlers must be idempotent or pass a downstream idempotency key. Exactly-once side effects are not promised.

## Initial handlers

- `noop`: completes without side effects.
- `echo`: example job type; currently validates the dispatch path and completes.
- Unknown types fail and are retried until the attempt limit, then enter `dead`.

Payloads are data only. The system never executes arbitrary payload content.

## Data model and indexes

The `jobs` table stores JSONB payload, status, attempt limits, scheduled time, lease metadata, worker ID, and last error. Partial indexes support pending claims, expired-lease recovery, and unique non-null idempotency keys.

## Known MVP gaps

Long-running handler heartbeat renewal/cancellation, concurrent handler execution, paginated listing, cancellation/replay APIs, Prometheus metrics, distributed tracing, and least-privilege database roles remain planned work. The current worker should only run short handlers whose execution time is comfortably below the configured lease duration.
