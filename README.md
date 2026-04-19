# Backend (Go monorepo)

Entry points: `services/*`, `infra/`, `pkg/`. Local orchestration: [`docker-compose.yml`](docker-compose.yml).

## Kong API Gateway (dev)

Compose runs **Kong** on **`http://localhost:8000`** (proxy) and `:8001` (admin API). Declarative config: [`infra/kong/kong.yml`](infra/kong/kong.yml).

- **JWT:** Kong does **not** validate JWTs (Option B): the gateway forwards `Authorization`; **authentication-service** and **inventory-api** verify HS256 with the same `JWT_SECRET` as in compose. Do not enable Kong’s `jwt` plugin without disabling duplicate verification in services.
- **CORS:** Allowed origins include `http://localhost:3000` and a placeholder production origin; edit `kong.yml` for your real frontend URL.
- **Request ID:** Global `correlation-id` plugin uses **`X-Request-Id`** (honors client value if present). Responses echo it when `echo_downstream` is enabled.
- **Rate limits:** `POST /api/v1/auth/login` and `POST /api/v1/auth/register` use dedicated routes with stricter `rate-limiting` (local policy); they are declared **before** the `/api/v1/auth` catch-all so Kong matches the longer paths first. Expect **429 Too Many Requests** when limits are exceeded (not 403).

### Health checks via Kong (no direct service port)

```bash
curl -sS http://localhost:8000/api/v1/inventory/health
curl -sS http://localhost:8000/api/v1/auth/health
curl -sS http://localhost:8000/api/v1/notifications/health
```

### Register → login → call inventory (all through Kong)

Replace bodies with real values; `jq` optional for extracting tokens from §9 envelopes.

```bash
BASE=http://localhost:8000

# Register (201)
curl -sS -X POST "$BASE/api/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d '{"tenant_name":"Acme","admin_name":"Owner","admin_email":"you@example.com","password":"strongpass123"}'

# Login (200) — response is §9 JSON; access token is under data.access_token
curl -sS -X POST "$BASE/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"you@example.com","password":"strongpass123"}'

TOKEN="<paste access_token from login data envelope>"

curl -sS "$BASE/api/v1/inventory/categories?page=1&per_page=20" \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Request-Id: demo-$(date +%s)"
```

### Demo DB seed (inventory + movements)

From [`infra/database/cmd/seed`](infra/database/cmd/seed): apply **after Goose migrations**. Requires **`DB_DSN`** (same as Goose / services).

```bash
# From repo backend/ (Makefile sets DB_DSN if you export it):
export DB_DSN='host=localhost ...'   # postgres URL
make seed-mock

# Or directly:
go run ./infra/database/cmd/seed --mode seed
```

**Rollback** demo tenant only (CASCADE wipes users, movements, stock, audit for that tenant):

```bash
make rollback-mock
# or: go run ./infra/database/cmd/seed --mode rollback
```

What it loads (single transaction):

| Area | Contents |
|------|-----------|
| Tenant | **Demo Tenant** (`slug` `demo-tenant-seed`) |
| Users | `admin@demo.local`, `staff@demo.local`, `viewer@demo.local` — dev password **`admin123`** |
| Masters | Categories, products (7 SKUs), warehouses **WH-JKT-01**, **WH-BDG-01**, **WH-SBY-01** |
| Movements | Confirmed inbound/transfer/outbound/adjustment + one **draft** + one **cancelled** (`reference_number` prefix **`SEED-`**) |
| Stock | `stock_balances` replayed for confirmed flows (each run **clears** stock for the tenant, deletes `SEED-*` movements, then re-inserts) |
| Audit | Sample `audit_logs` rows for list endpoints |

Seeding is **SQL-only**: it does **not** write `outbox_events` (workers) the way `POST …/confirm` does in the inventory API.

### Refresh, `/me`, logout (Kong)

After login, `data.refresh_token` is stored server-side (session row); use it **without** a Bearer header on refresh.

```bash
BASE=http://localhost:8000
# Set ACCESS, REFRESH from login §9 envelope (e.g. with jq: export ACCESS=$(jq -r '.data.access_token' login.json))
ACCESS="<paste access_token>"
REFRESH="<paste refresh_token>"

# Rotate tokens (200, §9 envelope; new pair in data)
curl -sS -X POST "$BASE/api/v1/auth/refresh" \
  -H "Content-Type: application/json" \
  -d "{\"refresh_token\":\"$REFRESH\"}"

# Current user (200, §9 envelope)
curl -sS "$BASE/api/v1/auth/me" -H "Authorization: Bearer $ACCESS"

# Logout (204; revokes all refresh sessions for this user)
curl -sS -o /dev/null -w "%{http_code}\n" -X POST "$BASE/api/v1/auth/logout" \
  -H "Authorization: Bearer $ACCESS"
```

After logout, the old **refresh** JWT should return **401** on `POST /auth/refresh`.

### Rate limit burst test (login)

Limits are configured in `kong.yml` (`minute` / `second`). Sending many rapid requests should eventually return **429**:

```bash
BASE=http://localhost:8000
BODY='{"email":"you@example.com","password":"wrong-or-right"}'
for i in $(seq 1 25); do
  code=$(curl -sS -o /dev/null -w "%{http_code}" -X POST "$BASE/api/v1/auth/login" \
    -H "Content-Type: application/json" -d "$BODY")
  echo "$i -> $code"
done
```

When limited, Kong adds rate-limit headers (unless `hide_client_headers`); response status is **429**.

### Direct upstream ports (bypass Kong)

Compose maps services directly for debugging only: inventory **8080**, notification **8081**, authentication **8082**. Prefer **8000** for integration tests that should match production routing.
