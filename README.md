# Distributed Job Runner

A durable, PostgreSQL-backed distributed job processing system written in Go.

> **Status:** Initial development. Interfaces and operational behavior may change before the first stable release.

## Goals

- Accept and inspect jobs through a versioned HTTP API.
- Persist job state in PostgreSQL so queued work survives process restarts.
- Claim work safely across concurrent workers using PostgreSQL row locking.
- Provide at-least-once execution with bounded retries, exponential backoff, and jitter.
- Prevent stale workers from completing jobs after their lease has been replaced.
- Support idempotency keys, worker heartbeats, lease recovery, and a dead-letter state.
- Shut down gracefully and expose health, readiness, structured logs, and metrics.

## Architecture

The initial implementation is organized around three executables:

- `cmd/api` — HTTP API for submitting and inspecting jobs.
- `cmd/worker` — worker process that claims and executes jobs.
- `cmd/djr` — administrative command-line interface.

Core packages live under `internal/`; database migrations are versioned under `migrations/`. PostgreSQL is the durable source of truth. Workers use short transactions and lease fencing to coordinate concurrent execution.

## Delivery semantics

The system targets **at-least-once execution**, not exactly-once execution. A worker can complete an external side effect and crash before recording success. Job handlers must therefore make side effects idempotent or use a downstream idempotency mechanism. Lease tokens fence off stale workers from mutating the job record after ownership changes.

## Development

The local development stack uses Docker Compose and PostgreSQL 16. See the project documentation and Makefile for setup, migrations, tests, and operational commands as those components land.

## Security

Do not commit credentials, production data, or real secrets. Copy `.env.example` to a local `.env` for development; `.env` is ignored by Git. The initial stack is intended for local development and must not be exposed to untrusted networks without authentication and deployment hardening.

Report suspected vulnerabilities privately as described in [SECURITY.md](SECURITY.md).

## Project status

This repository is being built incrementally. The first milestone establishes the Go module, database schema, configuration, API and worker foundations, local PostgreSQL stack, CI, tests, and contributor/security documentation.

## License

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE).
