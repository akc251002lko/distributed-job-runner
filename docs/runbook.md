# Operations Runbook

## Local startup

Run docker compose up --build. The migration service applies all SQL migrations and provisions separate local API/worker database roles. Check /healthz, /readyz, API /metrics, and worker :9090/metrics.

## Submit and inspect

Submit using POST /v1/jobs; provide Authorization: Bearer ... when credentials are configured. Use the returned ID with GET /v1/jobs/{id}. Browse with GET /v1/jobs?limit=50&status=pending; follow next_cursor unchanged. Every job operation is scoped to the authenticated client identity.

## Cancel and replay

POST to /v1/jobs/{id}/cancel to cancel a pending/running job. Running handlers are asked to stop through context cancellation after the heartbeat observes the invalidated lease. A handler that ignores cancellation may continue external work; side effects are never rolled back.

POST to /v1/jobs/{id}/replay to create a fresh job from a succeeded, dead, or cancelled job. It receives a new ID and starts with zero attempts.

## Scaling

Set WORKER_CONCURRENCY between 1 and 256 for per-process handler concurrency. Scale worker replicas horizontally. Choose lease duration based on database latency and scheduling delays; heartbeat interval is one-third of the lease. Each worker process exposes its own metrics endpoint, so scrape every replica.

API token-bucket limits are shared through PostgreSQL across API replicas. A database outage fails protected API requests closed with HTTP 503.

## Troubleshooting

- API not ready: inspect API logs and PostgreSQL connectivity.
- Jobs pending: check worker replicas, concurrency, supported handler type, and DB permissions.
- Jobs dead: inspect last_error, attempts, and max_attempts.
- Jobs repeatedly reclaimed: inspect worker restarts, handler cancellation behavior, database latency, CPU starvation, and lease duration.
- 401: verify bearer token or API_KEYS client mapping.
- 429: tune API_RATE_LIMIT_RPS and API_RATE_LIMIT_BURST, and inspect the per-client PostgreSQL token bucket.
- Database permission denied: verify API/worker connection strings use matching roles and re-run grants as migration owner.

## Shutdown and recovery

Send SIGTERM. API drains requests; worker stops claiming and cancels active handler contexts. Jobs with active leases are recovered after expiry if the worker exits before recording a terminal state. External side effects are not rolled back.

## Production caution

Local Compose credentials are not production secrets. Use separate strong database passwords, TLS, private networking, a secret manager, backups, monitoring, resource limits, and a reviewed migration/rollback plan. Do not expose metrics or health endpoints publicly.
