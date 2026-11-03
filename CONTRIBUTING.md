# Contributing

## Requirements

- Go 1.22+
- Docker Compose
- PostgreSQL 16, or the included Compose stack

## Pull request workflow

1. Create a focused feature branch.
2. Include tests for behavior changes.
3. Run `gofmt`, `go vet ./...`, and `go test ./...`.
4. Add a forward-only migration for schema changes and document operational impact.
5. Update architecture/runbook documentation where appropriate.
6. Describe the problem, implementation, compatibility impact, and validation in the PR.

## Engineering rules

- Preserve at-least-once delivery semantics; do not claim exactly-once execution.
- Use parameterized SQL and bounded contexts/timeouts.
- Treat payloads as untrusted data; never execute payload content.
- Fence worker state transitions with the current lease token.
- Keep secrets and production data out of commits, fixtures, logs, and tests.
- Prefer backward-compatible API and schema changes.
