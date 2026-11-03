# Security Policy

## Reporting

Do not open public issues for suspected vulnerabilities. Use GitHub's private vulnerability reporting for this repository when enabled, or contact repository maintainers privately.

## Supported security posture

This project is an MVP and is not independently audited. Do not expose the development Compose stack to the public internet.

## Deployment requirements

- Use TLS and private networking.
- Set APP_ENV=production, strong API credentials (32+ characters), and separate API_DATABASE_URL and WORKER_DATABASE_URL credentials. Client IDs must be unique and are enforced as tenant boundaries by API query scoping and PostgreSQL row-level security.
- Provision djr_api and djr_worker using ops/postgres/roles.sql; never run application processes as the database owner.
- Store secrets in a secret manager and rotate them.
- Restrict health and metrics endpoints to trusted monitoring networks.
- Apply resource limits and request gateway controls. Per-client token-bucket limits are shared through PostgreSQL; database outages fail closed.
- Back up PostgreSQL and test restoration.
- Review migrations and rollback procedures before deployment.

## Execution model

Handlers receive a context and JSON payload. They must validate payload schemas, honor cancellation, bound resource use, and make external side effects idempotent. Cancellation is cooperative; it does not roll back side effects.
