---
paths:
  - "infra/kong/kong.template.yml"
  - "services/*/api/openapi.yaml"
---

# Kong Routing — Business Logic Decisions

Every path defined in a service's `openapi.yaml` is accessed by clients through Kong with a **service prefix**. The prefix is defined in `infra/kong/kong.template.yml` and must match the OpenAPI `servers` URL.

## Path Prefix Mapping

| Service | Kong Prefix | OpenAPI `servers.url` | Docker Host |
|---------|-------------|----------------------|-------------|
| notification | `/notification` | `{{backendUrl}}/notification` | `service-notification` |
| authentication | `/authentication` | `{{backendUrl}}/authentication` | `service-authentication` |
| common | `/common` | `{{backendUrl}}/common` | `service-common` |
| ftm | `/ftm` | `{{backendUrl}}/ftm` | `service-ftm` |
| operation | `/operation` | `{{backendUrl}}/operation` | `service-operation` |
| billing | `/billing` | `{{backendUrl}}/billing` | `service-billing` |

**When adding a new service:** add a corresponding service block in `kong.template.yml` with the same prefix.

**When adding a new endpoint:** most endpoints are covered by the catchall strip-path route. Only add a specific route when the endpoint needs different auth or rate limiting than the default.

## Authentication Modes

The `custom_auth_plugin` supports three modes. Choose based on what the endpoint serves:

| Mode | When to Use | Header |
|------|-------------|--------|
| `userAuth` | Standard user-facing endpoints | `Authorization: Bearer <JWT>` |
| `deviceAuth` | Device-only endpoints (IoT, sensors) | `X-Device-Authorization: <token>` |
| `userAuthOrDeviceAuth` | Endpoints used by both users and devices | Either header |

### Public Endpoints (No Auth)

Some routes intentionally skip authentication:

- **Ping/health checks** — no plugins needed beyond `request-transformer`
- **Sign-in** — user doesn't have a token yet
- **Password forgot/reset** — user may not be able to authenticate
- **Permission check endpoints** — called by Kong's own auth plugin internally

When adding a public endpoint, create a **specific regex route** above the catchall (which has auth enabled).

## Rate Limiting Strategy

### Per-IP Rate Limiting (Public/Sensitive Endpoints)

Use `limit_by: ip` for endpoints that don't require authentication or are security-sensitive:

```yaml
- name: rate-limiting
  config:
    minute: 10
    hour: 100
    policy: redis
    redis:
      host: industrix-redis
      port: 6379
      password: {{ getenv "REDIS_PASSWORD" }}
    limit_by: ip
```

**Typical limits for security-critical endpoints:**

| Category | Minute | Hour | Rationale |
|----------|--------|------|-----------|
| Sign-in | 10 | 100 | Brute force protection |
| Password forgot/reset | 5 | 20 | Abuse prevention |
| Token exchange | 30 | 200 | Token generation rate |
| OTP generation | 10/min, 3/sec | 20 (+ 50/day) | Per device:user pair — absorb UX bursts, hard-cap daily abuse |
| Email sending | 20 | 200 | Per user, abuse prevention |

### Per-User Rate Limiting (Authenticated Endpoints)

The `rate_limit_key_plugin` extracts user ID from the JWT and sets `X-Rate-Limit-User-Id`. If the token is invalid/missing, it falls back to `ip:{client_ip}`.

```yaml
- name: rate_limit_key_plugin
  config:
    jwt_secret: {{ getenv "JWT_SECRET" }}
- name: rate-limiting
  config:
    second: 60
    minute: 2000
    hour: 40000
    limit_by: header
    header_name: X-Rate-Limit-User-Id
```

### Global Safety Net

A global rate-limiting plugin applies to ALL routes (200/s, 6000/min, 100000/hr per IP). This is the last line of defense — individual routes should have tighter limits.

## When to Add a Specific Route

Add a dedicated route (above the catchall) when:

1. **No authentication needed** — public endpoints must bypass the catchall's auth plugins
2. **Stricter rate limiting** — security-sensitive operations (login, password reset, OTP, email)
3. **Different auth mode** — endpoint needs `deviceAuth` while the service default is `userAuth`
4. **Custom rate limit key** — rate limit by something other than user ID (e.g., device:user pair)

Otherwise, the catchall strip-path route handles the endpoint automatically.

## Adding a New Service Checklist

When creating a new microservice:

1. Add service block in `kong.template.yml` with:
   - `name: {name}-service`
   - `host: service-{name}` (must match Docker Compose service name)
   - `port: 8080`, `protocol: http`
2. Add ping route (no auth)
3. Add any security-critical specific routes
4. Add catchall strip-path route with standard auth + rate limiting
5. Ensure OpenAPI `servers.url` matches: `{{backendUrl}}/{prefix}`
