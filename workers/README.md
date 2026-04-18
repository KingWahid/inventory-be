# workers

Background processes: **outbox relay**, **alert consumer** (Redis Streams), scheduled jobs.

## Modes

| `WORKER_MODE` / `--mode` | Outbox relay (Postgres → `inventory.events`) | Alert consumer (`XREADGROUP` + HMAC verify) |
|--------------------------|---------------------------------------------|---------------------------------------------|
| `all` (default) | yes | yes |
| `outbox-relay` | yes | no |
| `alerts` | no | yes |

## Outbox relay (`--mode=outbox-relay` or `--mode=all`)

Polls `outbox_events` where `published = false`, publishes to Redis Stream **`inventory.events`** via `pkg/eventbus` (`BaseEvent` + HMAC), then sets `published = true` and `published_at`. Uses `SELECT … FOR UPDATE SKIP LOCKED` (one DB transaction per row) so multiple relay processes can run safely.

### Environment

| Variable | Required | Description |
|----------|----------|-------------|
| `DB_DSN` | yes | PostgreSQL DSN (`pgx`) |
| `REDIS_ADDR` | yes | Redis address for Streams |
| `EVENTBUS_HMAC_SECRET` | yes | Shared secret for `eventbus.SignEvent` (same as consumers expected to verify) |
| `OUTBOX_RELAY_POLL_MS` | no | Idle poll interval when queue empty (default `500`) |
| `OUTBOX_RELAY_BATCH` | no | Max rows per drain loop (default `100`) |

## Alert worker (`--mode=alerts` or `--mode=all`)

Subscribes to consumer group **`alerts`** on stream **`inventory.events`** (ARCHITECTURE §10). Verifies `event_signature` with `EVENTBUS_HMAC_SECRET`, then runs a **stub** handler for `StockBelowThreshold` (replace with notification integration).

| Variable | Required | Description |
|----------|----------|-------------|
| `REDIS_ADDR` | yes | Same as relay |
| `EVENTBUS_HMAC_SECRET` | yes | Must match relay signing secret |
| `ALERT_CONSUMER_NAME` | no | Default `worker-alerts-1` |

### Manual acceptance (§6.3)

1. Run Postgres + Redis; apply migrations; start worker with relay mode.
2. Confirm a draft movement via inventory API so outbox rows are inserted.
3. Read the stream, e.g. `redis-cli XREVRANGE inventory.events + - COUNT 5` — expect entries with `event_type` / `event_payload` matching the domain outbox payload (ARCHITECTURE §10).

Duplicates on at-least-once delivery are possible if the process crashes between `XADD` and DB update; consumers should dedupe using stable `event_id` (`outbox:<id>`).
