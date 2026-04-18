---
paths:
  - "pkg/services/**"
  - "pkg/common/**"
  - "pkg/eventbus/**"
  - "workers/jobs/consumers/**"
---

# Business Logic Verification

## Service Unit Tests

When service code changes (`pkg/services/`, `pkg/common/`, `pkg/eventbus/`):

```bash
# Test specific affected package
go test -tags '!integration' ./pkg/services/{affected_module}/...

# Test common utilities
go test -tags '!integration' ./pkg/common/...

# Test eventbus
go test -tags '!integration' ./pkg/eventbus/...
```

All unit tests must pass. These tests use mocked repositories — no DB required.

## Consumer Handler Tests

When consumer handler logic changes (`workers/jobs/consumers/`):

```bash
go test -tags '!integration' ./workers/jobs/consumers/...
```

## Kong Routing Changes

When `infra/kong/kong.template.yml` changes:

- Kong config is generated from the template at container startup — no runtime apply needed
- If the dev server is up, verify routing by checking Kong's admin API or running E2E tests:
  ```bash
  make test-e2e-kong
  ```
  (Requires `industrix-kong` and `industrix-redis` running, plus `TEST_JWT_SECRET` set)
