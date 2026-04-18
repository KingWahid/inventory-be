---
paths:
  - "**/docker/**"
  - "**/Dockerfile"
  - "docker-compose*.yaml"
  - ".dockerignore"
---

# Docker Rules

## Multi-Stage Build Pattern

All Go service Dockerfiles use three stages: `builder`, `development`, `production`.

```dockerfile
# Stage 1: BUILDER — compiles the Go binary
FROM golang:${GO_VERSION} AS builder

# Stage 2: DEVELOPMENT — hot reload with Air
FROM builder AS development

# Stage 3: PRODUCTION — minimal Alpine with static binary
FROM ${PRODUCTION_IMAGE} AS production
```

Never collapse stages. Never add development tools to the production stage.

## Layer Caching Order

Copy files from least-frequently to most-frequently changed. This maximizes Docker layer cache hits:

```dockerfile
# Layer 1: Module files (rarely change)
COPY services/auth/go.mod services/auth/go.sum ./services/auth/
COPY pkg/common/go.mod pkg/common/go.sum ./pkg/common/
COPY pkg/database/go.mod pkg/database/go.sum ./pkg/database/
COPY pkg/services/go.mod pkg/services/go.sum ./pkg/services/
COPY pkg/eventbus/go.mod pkg/eventbus/go.sum ./pkg/eventbus/
COPY pkg/test_utils/go.mod pkg/test_utils/go.sum ./pkg/test_utils/

# Layer 2: Download deps (cached in BuildKit between builds)
RUN --mount=type=cache,target=/go/pkg/mod \
    cd services/auth && go mod download

# Layer 3: Shared packages (occasionally change)
COPY pkg ./pkg

# Layer 4: Service source (frequently changes)
COPY services/auth/api ./services/auth/api
COPY services/auth/cmd ./services/auth/cmd
```

## Static Binary Build

All Go binaries are compiled as fully static with no CGO:

```dockerfile
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /usr/local/bin/main ./cmd
```

Use BuildKit cache mounts for module and build caches. Dependencies are downloaded via `go mod download` in Layer 2.

## Non-Root User

All containers run as `appuser:1000`. Never run as root in production.

```dockerfile
# Alpine (production)
RUN addgroup -g 1000 appuser && adduser -D -u 1000 -G appuser appuser
USER appuser

# Debian (development)
RUN groupadd -g 1000 appuser && useradd -m -u 1000 -g appuser -s /bin/bash appuser
```

## Health Checks

Every service and infrastructure container must have a health check:

```dockerfile
# Go services — HTTP ping
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:${PORT}/ping || exit 1

# Workers — HTTP health
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:${HEALTH_PORT}/health || exit 1

# Redis — CLI ping with auth
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD redis-cli -a "$REDIS_PASSWORD" --no-auth-warning ping | grep -q PONG || exit 1
```

## Build Arguments

Standard ARGs used across all Dockerfiles:

| ARG | Purpose |
|-----|---------|
| `GO_VERSION` | Go compiler version (e.g., `1.25`) |
| `ENVIRONMENT` | `development` or `production` |
| `PRODUCTION_IMAGE` | Minimal Alpine base for production stage |

## Development Stage

Development stage includes:
- **Air** for hot reload (`go install github.com/air-verse/air@latest`)
- **gosu** for privilege dropping in entrypoint
- **`ENV GOWORK=off`** to disable workspace mode (modules resolve via volume mounts)
- Volume mounts for `pkg/`, service source, Go module cache, and build cache

## Entrypoint Scripts

Location: `{service}/docker/entrypoint.sh`

Pattern: Fix cache ownership, then drop privileges with `gosu`:

```bash
#!/bin/bash
set -e
if [ "$(id -u)" = "0" ]; then
    chown -R appuser:appuser /home/appuser/.cache 2>/dev/null || true
    chown -R appuser:appuser /home/appuser/go 2>/dev/null || true
    exec gosu appuser "$@"
else
    exec "$@"
fi
```

Always use `exec` to replace the shell process. Always `set -e` for fail-fast.

## Infrastructure Services

Infrastructure Dockerfiles (postgres, redis, mosquitto, kong) use template-based config generation with **gomplate** where dynamic configuration is needed. Config templates live alongside the Dockerfile in the `docker/` directory.

## Docker Compose

- Network: All services on a single `industrix` bridge network
- Volumes: Named volumes for data persistence, bind mounts for development hot-reload
- Port convention: Internal port `8080`, mapped to unique host ports per service
- Environment variables are interpolated via `${VAR}` syntax — never hardcode values in compose files

### Environment Files

Docker Compose reads variables from the root `.env` file automatically (no `env_file:` directive). All `${VAR}` references in compose files resolve from this file.

| File | Purpose |
|------|---------|
| `.env` | Local development — loaded by `docker compose` automatically |
| `.env.example` | Template for `.env` — commit this, never commit `.env` |
| `.env.test` | Test environment variables (DB, Redis, JWT for integration tests) |
| `.env.test.example` | Template for `.env.test` |
| `.env.prod` | Production runtime config |
| `.env.prod.build` | Production Docker build arguments (`GO_VERSION`, `PRODUCTION_IMAGE`, image tags) |
| `.env.prod.example` | Template for `.env.prod` |
| `.env.deployment` | Deployment-specific config (registry, tags) |
| `.env.staging.deployment` | Staging deployment overrides |
| `.env.ci` | CI pipeline environment |

Never commit `.env`, `.env.prod`, or `.env.test`. All other env files are committed.

## .dockerignore

The root `.dockerignore` excludes: `.git`, `docs`, `*.md`, `.env`, `docker-compose*.yaml`, `infra/`, `*_test.go`, development tooling. Keep it strict to minimize build context size.
