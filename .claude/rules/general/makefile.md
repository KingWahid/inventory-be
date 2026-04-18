---
paths:
  - "Makefile"
---

# Makefile Rules

## Never Run `go test` Directly for Integration Tests

**Always use `make test-all` or `make test-integration`** to run integration tests. Never run `go test -tags=integration ...` directly.

The Makefile loads credentials from `.env` (via `-include .env`) and maps them to `TEST_DB_*` / `TEST_REDIS_*` environment variables that integration tests expect. Running `go test` directly skips this, resulting in empty credentials and database authentication failures.

### How the Environment Pipeline Works

```
.env                          Makefile                         go test
─────────────                 ─────────────                    ─────────────
POSTGRES_PASSWORD=xxx    →    TEST_DB_PASSWORD ?= $(POSTGRES_PASSWORD)    →    os.Getenv("TEST_DB_PASSWORD")
POSTGRES_USERNAME=xxx    →    TEST_DB_USER ?= $(POSTGRES_USERNAME)        →    os.Getenv("TEST_DB_USER")
POSTGRES_DB=xxx          →    TEST_DB_NAME ?= $(POSTGRES_DB)              →    os.Getenv("TEST_DB_NAME")
REDIS_PASSWORD=xxx       →    TEST_REDIS_PASSWORD ?= $(REDIS_PASSWORD)    →    os.Getenv("TEST_REDIS_PASSWORD")
JWT_SECRET=xxx           →    TEST_JWT_SECRET ?= $(JWT_SECRET)            →    os.Getenv("TEST_JWT_SECRET")
```

The `?=` operator means variables can be overridden: `TEST_DB_PASSWORD=custom make test-all`.

## Key Targets

### Testing

| Target | What it does | When to use |
|--------|-------------|-------------|
| `make test` | Unit tests only (`go test` without build tags) | Quick check, no DB/Redis needed |
| `make test-short` | Tests with `-short` flag | Pre-commit hook (fastest) |
| `make test-all` | Unit + integration tests (`-tags=integration_all`) | Full test suite, requires DB + Redis |
| `make test-integration` | Integration tests only (`-tags=integration`) | When you only want integration tests |
| `make test-ci` | Full tests with coverage + race detection | CI pipeline |
| `make test-endpoint` | Handler/endpoint integration tests | `SERVICE=billing make test-endpoint` for one service |
| `make test-e2e-kong` | Kong rate limiting E2E tests | Requires Kong + Redis running |

**Worker container handling:** `test-all`, `test-integration`, `test-ci`, and `test-endpoint` automatically pause the workers container during tests (to prevent the outbox worker from consuming test events) and unpause it afterward via `trap`.

### Code Quality

| Target | What it does |
|--------|-------------|
| `make lint` | Run golangci-lint on all workspace modules |
| `make lint-fix` | Lint with auto-fix |
| `make fmt` | Format with gofmt + goimports |
| `make pre-commit` | Format + lint-fix + short tests (runs on git commit via hook) |

### Code Generation

| Target | What it does | When to run |
|--------|-------------|-------------|
| `make mocks-generate` | Regenerate repository mocks (mockery) | After changing repository interfaces |
| `make openapi-gen` | Regenerate OpenAPI stubs for all services | After modifying `openapi.yaml` specs |

### Build

| Target | What it does |
|--------|-------------|
| `make build` | Compile all services + seeder to `/tmp` (no Docker) |
| `make docker-build-prod-all` | Build all production Docker images (multi-platform) |
| `make docker-build-prod-{service}` | Build one service's production image |

### Database

| Target | What it does |
|--------|-------------|
| `make db-setup` | Run migrations + seed mock data |
| `make db-migrate` | Run migrations only (`infra/database make up`) |
| `make db-seed-mock` | Seed mock data only |
| `make db-down-all` | Rollback all migrations |

## Verification Pipeline Target Selection

When running the verification pipeline:

- **DB container running** → `make test-all` (includes integration tests)
- **DB container NOT running** → `make test` (unit tests only, no integration)

Never skip the test suite entirely — always run at least `make test`.
