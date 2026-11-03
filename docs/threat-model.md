# Threat Model

## Assets and boundaries

Assets include job payloads/state, client credentials, database credentials, lease ownership, and service availability. HTTP clients and stored payloads are untrusted. Payloads are data only and are never executed.

## Controls

| Risk | Current control | Residual risk |
|---|---|---|
| Unauthorized API use | Bearer token or per-client API keys; constant-time comparison and client-scoped job access | Credential rotation and external identity provider not included |
| Request abuse | 1 MiB body limit, strict JSON, per-client token bucket | PostgreSQL-backed per-client token bucket shared across replicas; database outage fails closed |
| SQL injection | Parameterized pgx queries | Review new query paths |
| Excessive worker load | Configurable bounded concurrency | Handler-specific resource limits remain application responsibility |
| Stale worker writes | Lease-token fencing and heartbeat renewal | External side effects require idempotency |
| Cancellation race | Cancellation invalidates lease; handler context cancelled after heartbeat | Cooperative only; cannot forcibly stop a handler |
| Privilege escalation | Separate API/worker roles, column-scoped updates, and forced row-level security for client-scoped API operations | Migration owner credentials remain highly privileged |
| Secret leakage | Environment-based configuration; ignored local env file | Use a secret manager and redaction review |
| Trace spoofing | Trace ID is correlation metadata, not authorization | Do not trust trace context as identity |
| Malicious payload | Typed handler registry; no arbitrary code execution | Each handler must validate its own payload schema |

## Deployment assumptions

Compose is local development, not a hardened production deployment. Production requires TLS, private networking, secret management, strong unique API/database credentials, least-privilege roles, backups, monitoring, patched images, and tested recovery procedures.

Report vulnerabilities privately using SECURITY.md.
