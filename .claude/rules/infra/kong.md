---
paths:
  - "infra/kong/**"
---

# Kong API Gateway — Infrastructure

Kong runs in **declarative mode** (no database). Configuration is generated at container startup from a Go template.

## Template Processing

- **Template file:** `infra/kong/kong.template.yml` — this is the ONLY file to edit
- **Engine:** [gomplate](https://docs.gomplate.ca/) (installed in the Dockerfile)
- **Startup flow:** `docker-entrypoint.sh` runs `gomplate -f template -o kong.yml`
- **Never edit the generated `kong.yml`** — it's overwritten on every container start

### Template Syntax

```yaml
password: {{ getenv "REDIS_PASSWORD" }}        # inject env var
{{ if (getenv "KONG_ENABLE_SSL") }}            # conditional block
  - name: mosquitto-tls-service
    ...
{{ end }}
```

## Service Definition Pattern

Every HTTP microservice follows the same structure:

```yaml
- name: {service-name}-service          # descriptive Kong service name
  host: service-{service-name}          # Docker Compose service name (DNS)
  port: 8080                            # all services expose 8080 internally
  protocol: http
  routes:
    - name: {service-name}-service-ping # health check (no auth)
      paths:
        - "~/{prefix}/ping$"
      plugins:
        - name: request-transformer
          config:
            replace:
              uri: /ping
    # ... security-critical routes with specific rate limits ...
    - name: {service-name}-service-strip-path  # catchall (last)
      paths:
        - /{prefix}
      strip_path: true
      plugins:
        - name: rate_limit_key_plugin
        - name: custom_auth_plugin
        - name: rate-limiting
```

### Key Rules

- `host` must match the Docker Compose service name exactly (Docker DNS resolution)
- All services use port `8080` internally
- `request-transformer` rewrites external paths to internal service paths
- Regex routes use `~` prefix: `"~/authentication/users/signin$"`
- Named captures for path params: `(?<deviceId>[0-9a-f-]+)`

## Route Ordering

Within a service, routes are ordered:

1. **Ping** — health check, no auth
2. **Security-critical specific routes** — exact path regex with strict rate limits
3. **Catchall strip-path** — generic route that strips the service prefix

The catchall MUST be last — Kong matches routes top-to-bottom for equal specificity.

## Custom Plugins

Two custom Lua plugins in `infra/kong/plugins/`:

| Plugin | Priority | Phase | Purpose |
|--------|----------|-------|---------|
| `rate_limit_key_plugin` | 100000 | rewrite | Extract user ID from JWT for per-user rate limiting |
| `custom_auth_plugin` | 10 | access | Authenticate requests via user JWT or device token |

### Plugin File Structure

```
infra/kong/plugins/{plugin_name}/
  handler.lua    # plugin logic (Kong PDK)
  schema.lua     # configuration schema
```

Plugins are installed in the Dockerfile:
```dockerfile
COPY plugins/{plugin_name} /usr/local/share/lua/5.1/kong/plugins/{plugin_name}
```

And registered via env var: `KONG_PLUGINS=bundled,custom_auth_plugin,rate_limit_key_plugin`

## Docker Build

- **Base image:** `kong/kong-gateway:3.9.0.0`
- **Dockerfile:** `infra/kong/docker/Dockerfile`
- **Entrypoint:** `infra/kong/docker/docker-entrypoint.sh`
- Template processing happens at runtime (not build time) so env vars can differ per environment

## Environment Variables Used in Template

| Variable | Purpose |
|----------|---------|
| `JWT_SECRET` | Used by `rate_limit_key_plugin` to decode JWT |
| `REDIS_PASSWORD` | Redis connection for rate limiting state |
| `KONG_ENABLE_SSL` | Enables TLS MQTT route (conditional block) |

## SSL Certificate Handling

Kong accepts SSL certificates as **inline PEM content** via environment variables — no file mounts or Docker secrets needed.

### How It Works

1. Set `KONG_SSL_CERT_CONTENT` and `KONG_SSL_CERT_KEY_CONTENT` with PEM content (literal `\n` for line breaks)
2. The entrypoint (`docker-entrypoint.sh`) writes them to `/kong/ssl/cert.pem` and `/kong/ssl/key.pem`
3. The entrypoint exports `KONG_SSL_CERT` and `KONG_SSL_CERT_KEY` pointing to those files
4. Kong reads the files as usual

### Environment Variables

| Variable (container) | Variable (host/.env) | Purpose |
|---------------------|---------------------|---------|
| `KONG_SSL_CERT_CONTENT` | `SSL_CERT_CONTENT` | Inline PEM certificate content |
| `KONG_SSL_CERT_KEY_CONTENT` | `SSL_KEY_CONTENT` | Inline PEM private key content |
| `KONG_ENABLE_SSL` | — | Set to `true` to enable SSL listeners |

When `KONG_SSL_CERT_CONTENT` is empty, the cert-writing step is skipped (local dev mode).
