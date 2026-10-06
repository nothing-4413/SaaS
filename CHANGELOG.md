# Changelog

## 1.0.0 - 2026-10-06

### Added

- PostgreSQL integration test suite (`//go:build integration`) with a dedicated CI job that starts a real PostgreSQL 16 service, applies `migrations/*.up.sql` with `psql` and runs `go test -tags=integration ./...`
- `scripts/local-postgres.ps1` for starting a portable PostgreSQL, applying migrations and stopping it again without Docker
- Admin console screenshots in `docs/` plus a `Status` section in the README
- `.gitignore` and `.gitattributes` (LF everywhere)

- Embedded StockPilot management console for the inventory, catalog, fulfillment and reporting workflow
- SMTP password-reset delivery, local Mailpit development inbox and an end-to-end Compose smoke flow covering inventory, orders, reports, CSV and email
- Production environment template and interview demonstration runbook

- Multi-tenant organizations, users, roles and permissions
- HMAC Bearer tokens and permission middleware
- Products, SKUs, warehouses and tenant-scoped inventory
- Concurrent stock reservation, release and deduction
- Idempotent receipt, issue and sales order workflows
- Order lifecycle Outbox events and retrying notification dispatcher
- Signed Webhook delivery
- Persistent webhook subscriptions and standalone Outbox Worker
- Low-stock alert producer
- Configurable persistent low-stock rules and scheduled scans
- Reports, CSV import/export and HTTP endpoints
- PostgreSQL migration schema and SQL transaction helpers
- Docker Compose, Dockerfile, health checks, graceful shutdown and metrics
- OpenAPI, architecture, security and contribution documentation

### Fixed

- The embedded console could not load `/app.js` or `/styles.css` in a browser: the security middleware sent `Content-Security-Policy: default-src 'none'` on every response, so the console's own scripts were blocked and the sign-in form degraded to a native GET submit that put the password in the URL. The policy now allows same-origin assets only, with a regression test.
- The PostgreSQL order store returned freshly created errors for insufficient stock, so `errors.Is(err, inventory.ErrInsufficient)` never matched and the API answered 400 where the in-memory store answered 409.
- Added the missing `/favicon.svg` asset and route; unknown assets still return 404.

### Changed

- GitHub Actions now use `actions/checkout@v5` and `actions/setup-go@v6`.

### Verification

- Unit tests cover tenant isolation, idempotency, concurrency, rollback, token validation, CSV validation and HTTP routing.
- `go test ./...`, `go vet ./...`, `go build ./...`, JavaScript syntax, shell syntax, Compose configuration and Git diff checks pass with Go 1.22.12.
- PostgreSQL workflows lock inventory rows in stable order and commit order/document state, stock changes and Outbox events atomically.
- GitHub Actions runs `gofmt`, `go test ./...`, `go vet`, `go test -race ./...` and `go build ./...` on every push and pull request, plus the PostgreSQL integration job and the Compose smoke flow.
- `go vet -tags=integration ./...` compiles the integration suite, and `TEST_DATABASE_URL=... go test -tags=integration ./internal/inventory/...` passes against PostgreSQL 16.6 both locally and in CI.
- The console was exercised in a real browser (sign-in, overview metrics, order queue) while capturing the README screenshots.
