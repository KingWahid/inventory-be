# Verification Pipeline

This pipeline runs **after implementation is complete** and **before code review**. The lead orchestrates verification. For failures in code written by a delegated agent, re-dispatch that agent to fix it. For failures in code the lead wrote directly, fix it directly.

## Pipeline Steps

Run steps in order. Stop at the first failure, fix the root cause, then re-run the entire pipeline.

### Step 1: Build + Tests

**If `industrix-postgres` is running:**
```bash
make test-all
```

**If `industrix-postgres` is NOT running:**
```bash
make test
```

This covers:
- Go compilation (implicit — tests won't run if code doesn't build)
- Unit tests across all packages
- Integration tests (if dev DB is up)

**Air hot-reload:** If dev containers are running, Air picks up code changes automatically. Check container logs for rebuild errors:
```bash
docker compose logs --tail=50 <compose-service-name>
```

**Responsible agent (if delegated):**
- `pkg/database/` failures → `db-dev`
- `pkg/services/`, `pkg/common/`, `pkg/eventbus/` failures → `biz-dev`
- `services/`, `workers/`, `sync-services/` failures → `app-dev`
- Build infrastructure failures (Dockerfile, Air config) → `infra-dev`

**Code generation:** If repository interfaces or OpenAPI specs changed, regenerate before re-running:
```bash
make mocks-generate      # if repository interfaces changed
make openapi-gen         # if OpenAPI specs changed
```

### Step 2: Database Verification (conditional)

**Trigger:** New files in `infra/database/migrations/` or `infra/database/cmd/seed/mock-data/`.

Test apply + rollback. See `.claude/rules/database/verification.md` for exact commands.

**Responsible agent:** `db-dev`

**Skip if:** No migration or mock data files changed.

### Step 3: Docker Verification (conditional)

**Trigger:** Changes to Dockerfiles, `docker-compose.yaml`, `docker-compose.prod.yaml`, `.air.toml`, or entrypoint scripts.

Build the production target for affected services:
```bash
docker compose build <compose-service-name>
```

Service name mapping (file path → compose service name):

| Changed file under | Compose service name |
|--------------------|---------------------|
| `services/authentication/` | `service-authentication` |
| `services/billing/` | `service-billing` |
| `services/common/` | `service-common` |
| `services/ftm/` | `service-ftm` |
| `services/notification/` | `service-notification` |
| `services/operation/` | `service-operation` |
| `sync-services/ftm/` | `sync-service-ftm` |
| `workers/` | `workers` |
| `infra/kong/` | `industrix-kong` |
| `infra/redis/` | `industrix-redis` |
| `infra/mosquitto/` | `industrix-mosquitto` |
| `infra/postgres/` | `industrix-postgres` |

If dev containers are running, also verify startup:
```bash
docker compose up -d <compose-service-name>
docker compose logs --tail=50 <compose-service-name>
```

**Responsible agent:** `infra-dev`

**Skip if:** No Dockerfile, docker-compose, `.air.toml`, or entrypoint changes.

## Failure Handling

1. Identify which step failed and which files/packages are involved
2. Route the fix:
   - Code you wrote directly → fix directly
   - Code a delegated agent wrote → re-dispatch that agent with the failure details
3. **Re-run the entire pipeline from Step 1** — a fix may introduce new issues
4. Repeat until the full pipeline passes

Do NOT proceed to code review until all applicable steps pass.
