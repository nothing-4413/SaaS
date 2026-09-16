# Changelog

## Unreleased

### Added

- Multi-tenant organizations, users, roles and permissions
- HMAC Bearer tokens and permission middleware
- Products, SKUs, warehouses and tenant-scoped inventory
- Concurrent stock reservation, release and deduction
- Idempotent receipt, issue and sales order workflows
- Order lifecycle Outbox events and retrying notification dispatcher
- Signed Webhook delivery
- Low-stock alert producer
- Reports, CSV import/export and HTTP endpoints
- PostgreSQL migration schema and SQL transaction helpers
- Docker Compose, Dockerfile, health checks, graceful shutdown and metrics
- OpenAPI, architecture, security and contribution documentation

### Verification

- Unit tests cover tenant isolation, idempotency, concurrency, rollback, token validation, CSV validation and HTTP routing.
- GitHub Actions runs `gofmt`, `go test ./...` and `go build ./...` on every push and pull request.
