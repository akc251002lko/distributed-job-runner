# Architecture

## Components

- **API:** validates HTTP requests, applies optional bearer-token authentication, and persists jobs.
- **Worker:** polls PostgreSQL, atomically claims eligible jobs, renews active leases, executes registered job types, and records success or failure.
- **PostgreSQL 16:** durable source of truth for job state, retry scheduling, idempotency, and leases.
- **Metrics:** each process exposes Prometheus text metrics for its own counters; API metrics are served at `/metrics`, worker metrics on its configured metrics listener.

The API and worker are independent processes. No in-memory queue is authoritative.

## Claim and fencing protocol

A single PostgreSQL statement first moves expired running jobs that exhausted their attempts to `dead`, then selects an eligible job with `FOR UPDATE SKIP LOCKED`. The claim update assigns a random lease token, worker ID, expiry, and increments the attempt count.

The worker renews the lease every one-third of the lease duration. Renewal, completion, and failure require the exact lease token and an unexpired lease. A stale worker cannot renew or mutate state after a newer worker reclaims the job. If renewal fails, the worker cancels the handler context and abandons the attempt.

## Delivery guarantees

This is **at-least-once** delivery. A worker may perform an external side effect and crash before recording success. Handlers must be idempotent or pass a downstream idempotency key. Exactly-once side effects are not promised.

## Initial handlers

- `noop`: completes without side effects.
- `echo`: example job type; currently validates dispatch and completes.
- Unknown types fail and follow the retry/dead-letter lifecycle.

Payloads are data only. The system never executes arbitrary payload content.

## Data model and indexes

The `jobs` table stores JSONB payload, status, attempt limits, scheduled time, lease metadata, worker ID, and last error. Partial indexes support pending claims, expired-lease recovery, and unique non-null idempotency keys.

## Known MVP gaps

A production handler registry, bounded concurrent execution, job listing/pagination, cancellation/replay APIs, per-client authorization/rate limiting, tracing, and least-privilege database roles remain planned work. Handlers must honor context cancellation and remain idempotent.
