---
paths:
  - "docker-compose*.yaml"
  - "**/docker/**"
  - "**/.air.toml"
---

# Dev Server

## Container Inventory

### Infrastructure

| Container | Purpose | Health Check |
|-----------|---------|--------------|
| `industrix-postgres` | PostgreSQL database | `pg_isready` |
| `industrix-redis` | Redis cache + streams | `redis-cli ping` |
| `industrix-kong` | Kong API gateway | HTTP `/status` |
| `industrix-minio` | MinIO object storage | HTTP `/minio/health/live` |

### Go Microservices

| Container | Service | Health Check |
|-----------|---------|--------------|
| `service-authentication` | Auth service | HTTP `/ping` |
| `service-notification` | Notification service | HTTP `/ping` |
| `service-common` | Common service | HTTP `/ping` |
| `service-ftm` | FTM service | HTTP `/ping` |
| `service-operation` | Operation service | HTTP `/ping` |
| `service-billing` | Billing service | HTTP `/ping` |
| `sync-service-ftm` | FTM MQTT sync | HTTP `/ping` |
| `workers` | Background workers | HTTP `/health` |

## Hot Reload (Air)

All Go microservices run with **Air** in development. Air watches `.go` files and automatically rebuilds + restarts the binary inside the container.

**Key behaviors:**
- Code changes in `services/`, `workers/`, `sync-services/` are picked up automatically via volume mounts
- Changes in `pkg/` are watched by Air via `include_dir` and volume mounts
- **No container restart needed** — Air detects file changes and rebuilds

**When Air does NOT pick up changes:**
- New dependencies added to `go.mod` → requires `docker compose up -d --build <service>` (re-runs `go mod download`)
- Changes to `docker-compose.yaml` or `Dockerfile` → requires `docker compose up -d --build <service>`
- Changes to `.env` → requires `docker compose up -d` to reload env vars

## Checking Container Status

```bash
# List all running containers
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"

# Check specific container
docker ps --filter name=industrix-postgres

# Check container logs (last 50 lines)
docker logs --tail 50 <container-name>

# Check if a container is running (exit code 0 = running)
docker inspect --format '{{.State.Running}}' <container-name>
```

## Workers Container

The `workers` container is special:
- It runs background jobs (outbox processing, cron jobs, event consumers)
- **Paused during integration tests** to prevent outbox worker from consuming test events
- `make test-all` and `make test-integration` handle pausing/resuming automatically
- Manual: `docker pause workers` / `docker unpause workers`
