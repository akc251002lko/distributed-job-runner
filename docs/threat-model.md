# Threat Model

## Assets

Job payloads and execution state; database credentials and API token; lease ownership and retry integrity; availability of API, workers, and PostgreSQL.

## Trust boundaries

HTTP clients are untrusted. The API rejects unknown JSON fields, limits request bodies to 1 MiB, validates type and idempotency-key lengths, and can require bearer authentication. Workers treat stored payloads as untrusted data and never execute payload content.

## Risks and controls

| Risk | Current control | Remaining work |
|---|---|---|
| Unauthorized API access | Bearer token; required by production config | Rotation, client identity and authorization |
| Oversized/malformed requests | 1 MiB cap, strict JSON decoding, field validation | Rate limiting |
| SQL injection | Parameterized pgx queries | Review future queries |
| Duplicate execution | At-least-once contract; idempotency keys | Downstream idempotency helpers |
| Stale worker state writes | Lease-token fencing, periodic lease renewal, and cancellation after lease loss | Downstream side effects still require idempotency |
| Secret leakage | Environment config; local env ignored by Git | Secret manager and redaction review |
| Database compromise | Isolated local Compose network | Least-privilege roles, TLS, encryption, backups |
| Malicious payload | No arbitrary code execution | Per-handler payload schemas |

## Deployment assumptions

The Compose stack is a local development environment, not a hardened production deployment. Production requires private networking, TLS, strong credentials, secret management, restricted database roles, patched images, backups, monitoring, and authenticated API access.

Report vulnerabilities privately using [SECURITY.md](../SECURITY.md).
