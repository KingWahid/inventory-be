---
paths:
  - "services/**"
  - "workers/**"
  - "sync-services/**"
---

# Application Layer Verification

## Build Verification

When any Go code changes, verify compilation:

**If dev server is running** — Air picks up changes automatically. Check container logs for build errors:
```bash
docker logs --tail 20 <affected-container>
```

**If dev server is NOT running** — compile all services to `/tmp`:
```bash
make build
```

This builds all services (`services/`, `sync-services/`, `workers/`) and outputs binaries to `/tmp`. All must compile cleanly.

## Handler Integration Tests

When handler code changes (`services/*/api/`), handler tests require `industrix-postgres` and `industrix-redis`:

```bash
# Test specific service handlers
make test-endpoint SERVICE=<service_name>

# Test all service handlers
make test-endpoint
```

Service names: `authentication`, `billing`, `common`, `ftm`, `notification`, `operation`.

## Full Test Suite

Run all tests (unit + integration) when `industrix-postgres` is available:

```bash
make test-all
```

This cleans test cache, pauses the `workers` container, runs with `-tags=integration_all`, then resumes workers.

If `industrix-postgres` is NOT available, run unit tests only:

```bash
make test
```

## OpenAPI Spec Changes

When `services/*/api/openapi.yaml` changes, generated code must be regenerated:
```bash
make openapi-gen
```

Verify no uncommitted changes to `services/*/stub/` files after regeneration.

## Worker Wiring Changes

When worker wiring changes (`workers/` but not `workers/jobs/consumers/`):

```bash
go test -tags '!integration' ./workers/...
```
