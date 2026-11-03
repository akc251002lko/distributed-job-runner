# Operations Runbook

## Local startup

1. Copy `.env.example` to `.env` and review the local-only settings.
2. Start the stack with `docker compose up --build`.
3. Wait for PostgreSQL health and the one-shot migration service to complete.
4. Check `GET http://localhost:8080/healthz` and `GET http://localhost:8080/readyz`.

## Submit and inspect a job

```sh
curl -sS -X POST http://localhost:8080/v1/jobs \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: sample-001' \
  -d '{"type":"noop","payload":{"source":"runbook"}}'
```

Use the returned ID with `GET /v1/jobs/{id}`. If `API_TOKEN` is configured, send `Authorization: Bearer <token>`.

## Troubleshooting

- **API not ready:** inspect API logs and confirm PostgreSQL health/connectivity.
- **Jobs remain pending:** confirm the worker is running and inspect worker logs.
- **Jobs become dead:** inspect `last_error`, `attempts`, and `max_attempts`; confirm the handler type is supported.
- **Jobs are reclaimed:** investigate worker restarts and lease duration. Reclaim may cause duplicate execution; handler side effects must be idempotent.
- **Database unavailable:** restore PostgreSQL connectivity before restarting processes; persisted pending jobs remain in the database.

## Shutdown and recovery

Send SIGTERM. The API drains active requests; the worker stops polling. A process killed during execution may leave a running lease that becomes eligible for reclaim after expiry. External side effects are not rolled back.

## Production caution

Compose credentials and unauthenticated local defaults are for development only. Before production use, configure TLS, strong unique credentials, API authentication, private networking, database least-privilege roles, backups, monitoring, resource limits, and a reviewed migration/rollback procedure. Do not expose this stack directly to the public internet.
