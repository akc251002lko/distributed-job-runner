# Security Policy

## Supported versions

Until a stable release policy is published, security fixes target the latest development branch.

## Reporting a vulnerability

Do not disclose exploitable vulnerabilities in public issues. Use GitHub's **Report a vulnerability** feature on this repository to submit a private advisory. Include affected commit/version, impact, reproduction steps, and suggested mitigation. Never include real credentials or personal data.

## Scope

Security-sensitive areas include HTTP authentication and validation, SQL queries, lease ownership, retry handling, dependency updates, container configuration, and secret handling.

Default Compose credentials and unauthenticated local API settings are strictly for local development. Production requires `APP_ENV=production` and a non-empty `API_TOKEN`, plus TLS, private networking, secret storage, database permissions, and monitoring.
