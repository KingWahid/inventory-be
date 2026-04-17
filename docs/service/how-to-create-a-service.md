# How to Create a New Service

A step-by-step guide to creating a new service in `backend/services/`.

## Table of Contents

- [Service Structure](#service-structure)
- [Step-by-Step Creation](#step-by-step-creation)
  - [Service Code Setup](#service-code-setup)
  - [Hot Reloading with Air](#hot-reloading-with-air)
- [Infrastructure Registration](#infrastructure-registration)
  - [Kong Gateway Configuration](#kong-gateway-configuration)
  - [Docker Compose Configuration](#docker-compose-configuration)
  - [Database Permission Registration](#database-permission-registration)

---

## Service Structure

Every service follows this structure:

```
backend/services/your-service/
├── main.go              # Entrypoint - starts the HTTP server
├── go.mod               # Go module dependencies
├── generate.go          # Code generation directives
├── config/
│   └── config.go        # Configuration management (Viper)
├── openapi/
│   └── openapi.yaml     # API specification
├── stub/
│   └── openapi.gen.go   # Generated code from OpenAPI (DO NOT EDIT)
├── api/
│   └── handlers.go      # HTTP request handlers
├── service/
│   └── service.go       # Business logic layer
└── locales/             # Translation files
    ├── en/
    └── id/
```

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Step-by-Step Creation

### Service Code Setup

1. **Create directory structure:**
   ```bash
   mkdir -p backend/services/your-service/{config,openapi,api,service,stub,locales/{en,id}}
   ```

2. **Create go.mod:**
   ```bash
   cd backend/services/your-service
   go mod init github.com/industrix-id/backend/services/your-service
   ```

3. **Add replace directives** (in go.mod):
   ```go
   replace github.com/industrix-id/backend/pkg/common => ../../pkg/common
   replace github.com/industrix-id/backend/pkg/database => ../../pkg/database
   ```

4. **Create config/config.go:**
   - Define Config struct
   - Implement LoadConfig() with Viper
   - See [how-to-use-configuration.md](./how-to-use-configuration.md)

5. **Create openapi/openapi.yaml:**
   - Define your API endpoints
   - Define request/response schemas
   - See [how-to-structure-openapi.md](./how-to-structure-openapi.md)

6. **Create generate.go:**
   - Add code generation directive

7. **Generate stub code:**
   ```bash
   go generate ./...
   ```

8. **Create service/service.go:**
   - Define Service interface
   - Implement service struct
   - Implement NewService constructor
   - Implement business logic methods
   - See [how-to-write-service-layer.md](./how-to-write-service-layer.md)

9. **Create api/handlers.go:**
   - Create ServerHandler struct
   - Implement stub.ServerInterface methods
   - See [how-to-write-handlers.md](./how-to-write-handlers.md)

10. **(If needed) Add JWT Auth Middleware:**
    If your handlers need access to the JWT token (to pass to service methods), use the centralized middleware:
    ```go
    // In main.go
    app := initialization.NewEcho(&cfg.DashboardUIHostname)
    app.Use(initialization.JWTAuthMiddleware([]string{"/ping"}))
    ```

    Then in handlers, retrieve the token:
    ```go
    // In api/handlers.go
    import "github.com/industrix-id/backend/pkg/common/jwt"

    func (h *ServerHandler) GetItems(ctx echo.Context, params stub.GetItemsParams) error {
        token, err := jwt.GetJWTFromEchoContext(ctx)
        if err != nil {
            return err
        }
        // Pass token to service layer...
    }
    ```

11. **Create main.go:**
    - Follow the pattern from common/authentication services
    - Initialize everything in order
    - Register handlers
    - Start server

    ```go
    func main() {
        // 1. Load configuration from env vars, flags, or config files
        cfg, err := config.LoadConfig()
        if err != nil {
            panic(fmt.Sprintf("failed to load config: %v", err))
        }

        // 2. Initialize Sentry (error tracking)
        flush, err := initialization.InitSentry(cfg.SentryDsn, cfg.Env)
        if err != nil {
            panic(fmt.Sprintf("failed to initialize Sentry: %v", err))
        }
        // defer flush ensures pending Sentry error reports are sent before service shuts down.
        // Without this, errors queued during shutdown might be lost.
        // The 2 second timeout gives Sentry time to send pending events.
        defer flush(2 * time.Second)

        // 3. Initialize logger (zap) with Sentry integration
        if err = initialization.InitLogger(cfg.LogLevel, cfg.Env, cfg.Service, sentry.CurrentHub().Client()); err != nil {
            panic(fmt.Sprintf("failed to initialize logger: %v", err))
        }

        // 4. Load translation files (i18n)
        if err = translations.LoadServiceLocales(localesFS, "your-service"); err != nil {
            panic(fmt.Sprintf("failed to load translations: %v", err))
        }

        // 5. Create Echo web framework instance
        app := initialization.NewEcho(&cfg.DashboardUIHostname)

        // 6. Create your service instance (business logic)
        yourService, err := service.NewService(/* all config params */)
        if err != nil {
            panic(fmt.Sprintf("failed to create service: %v", err))
        }

        // 7. Create HTTP handlers
        handler := api.NewServerHandler(yourService)

        // 8. Register routes from OpenAPI spec
        stub.RegisterHandlers(app, handler)

        // 9. Start HTTP server
        if err = app.Start(fmt.Sprintf(":%d", cfg.Port)); err != nil {
            panic(fmt.Sprintf("failed to start server: %v", err))
        }
    }
    ```

12. **Add translation files:**
    - Create locales/en/your-service.yaml
    - Create locales/id/your-service.yaml

13. **Test it locally:**
    ```bash
    go run main.go
    ```

### Hot Reloading with Air

**What is Air?**

Normally when you change Go code, you have to:
1. Stop the running server (Ctrl+C)
2. Run `go build` to compile
3. Run the binary again

This gets tedious when you're making frequent changes. **Air** solves this by watching your files and automatically rebuilding + restarting whenever you save a file. This is called "hot reloading" - your changes go live instantly without manual restarts.

**Create `.air.toml` in your service directory:**

```toml
# Config file for Air - enables automatic rebuild on file changes

[build]
bin = "/usr/local/bin/main"               # Where to put the compiled binary
cmd = "cd /backend/services/your-service && go mod download && go build -o /usr/local/bin/main ."
include_ext = ["go"]             # Only watch .go files
exclude_dir = ["tmp", "vendor"]  # Ignore these folders
exclude_file = []                # Ignore specific files (none by default)
delay = 2000                     # Wait 2 seconds after change before rebuilding
stop_on_error = true             # Stop the app if build fails

[log]
level = "debug"                  # Show detailed logs
```

**What each setting does:**

| Setting | What it does | Why it matters |
|---------|--------------|----------------|
| `bin` | Where the compiled program goes | Air needs to know where to find the binary to run it |
| `cmd` | The command to build your code | This runs every time you save a file |
| `include_ext` | Which file types to watch | We only care about `.go` files changing |
| `exclude_dir` | Folders to ignore | Don't rebuild when `tmp` or `vendor` changes |
| `delay` | Wait time before rebuilding (ms) | Prevents rebuilding 10 times if you save rapidly |
| `stop_on_error` | Stop if build fails | Shows you the error instead of running old code |

**How it works:**

```
You're coding...
     │
     ▼
┌─────────────────────────────────────┐
│  You save a .go file (Ctrl+S)       │
└─────────────────────────────────────┘
     │
     ▼
┌─────────────────────────────────────┐
│  Air detects the file changed       │
│  (it's watching all .go files)      │
└─────────────────────────────────────┘
     │
     ▼
┌─────────────────────────────────────┐
│  Air waits 2 seconds (delay=2000)   │
│  in case you're still typing        │
└─────────────────────────────────────┘
     │
     ▼
┌─────────────────────────────────────┐
│  Air runs: go build -o main .       │
│  (compiles your code)               │
└─────────────────────────────────────┘
     │
     ▼
┌─────────────────────────────────────┐
│  Air stops the old server           │
│  and starts the new one             │
└─────────────────────────────────────┘
     │
     ▼
Your changes are live! Test them now.
```

**Important: The paths are for Docker containers**

Notice the paths start with `/backend/services/...` - these are paths **inside the Docker container**, not your local machine. When you run services via Docker Compose, your local files are mounted into the container at `/backend`.

**Running Air (two ways):**

1. **Via Docker (recommended)** - Air runs automatically when you start services with Docker Compose. No setup needed!

2. **Locally (without Docker):**
   ```bash
   # First, install Air (one-time setup)
   go install github.com/air-verse/air@latest

   # Then run it in your service folder
   cd backend/services/your-service
   air

   # Air will now watch for changes and auto-rebuild
   ```

**Troubleshooting:**

| Problem | Solution |
|---------|----------|
| "air: command not found" | Run `go install github.com/air-verse/air@latest` and make sure `$GOPATH/bin` is in your PATH |
| Changes not detected | Check `include_ext` includes "go" and your file isn't in `exclude_dir` |
| Rebuilds too often | Increase `delay` value (e.g., 3000 for 3 seconds) |
| Build errors not showing | Set `stop_on_error = true` and check terminal output |

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Infrastructure Registration

After your service code is working locally, register it in the infrastructure:

### Kong Gateway Configuration

**File:** `backend/infra/kong/kong.template.yml`

Register your service and routes in Kong for API gateway routing.

**When do you need a specific route in Kong?**

Most endpoints **don't need** their own Kong route entry. They use the catch-all `strip-path` route which:
- Strips the service prefix (e.g., `/common/users` → `/users`)
- Applies the `custom_auth_plugin` for authentication

**You only need a specific route when the endpoint needs different behavior:**

| Reason | Example | Why |
|--------|---------|-----|
| **No authentication** | `/ping`, `/endpoint-permissions` | Health checks and internal auth endpoints must be accessible without a token |
| **Custom rate limiting** | `/ftm/.../otp` | OTP endpoints need stricter limits (1/sec, 5/min) to prevent brute force |
| **Special request transformation** | Capturing path params for rate limit keys | Need to extract IDs from URL for composite rate limiting |

**Example: Common service routes explained:**

```yaml
routes:
  # 1. Health check - NO AUTH (no custom_auth_plugin)
  - name: common-service-ping
    paths:
      - "~/common/ping$"
    plugins:
      - name: request-transformer
        config:
          replace:
            uri: /ping

  # 2. Internal auth endpoint - NO AUTH (called BY Kong for auth checks)
  - name: common-service-endpoint-permissions
    paths:
      - "~/common/endpoint-permissions$"
    plugins:
      - name: request-transformer
        config:
          replace:
            uri: /endpoint-permissions

  # 3. Catch-all route - WITH AUTH (all other endpoints)
  - name: common-service-strip-path
    paths:
      - /common
    strip_path: true
    plugins:
      - name: custom_auth_plugin  # <-- Auth plugin applied here
        config:
          mode: "userAuth"
          userAuthEndpoint: "http://service-common:8080/endpoint-permissions"
```

**Example: FTM OTP route with custom rate limiting:**

```yaml
# Specific route BEFORE the catch-all (Kong matches in order)
- name: ftm-service-device-user-otp
  paths:
    - "~/ftm/fuel-tank-monitoring-devices/(?<deviceId>[0-9a-f-]+)/users/(?<userId>[0-9a-f-]+)/otp$"
  plugins:
    # Extract path params for rate limit key
    - name: pre-function
      config:
        access:
          - |
            local captures = ngx.ctx.router_matches.uri_captures or {}
            kong.service.request.set_header("X-Rate-Limit-Key",
              captures.deviceId .. ":" .. captures.userId)
    # Stricter rate limiting for OTP
    - name: rate-limiting
      config:
        second: 1
        minute: 5
        hour: 10
        day: 30
    # Still needs auth
    - name: custom_auth_plugin
      config:
        mode: "userAuth"
        userAuthEndpoint: "http://service-common:8080/endpoint-permissions"
```

**Basic service registration (for most services):**

```yaml
services:
  - name: your-service
    host: service-your-service  # Docker service name
    port: 8080
    protocol: http
    routes:
      # Health check (no auth)
      - name: your-service-ping
        paths:
          - "~/your-service/ping$"
        plugins:
          - name: request-transformer
            config:
              replace:
                uri: /ping
      # All other routes (with auth)
      - name: your-service-strip-path
        paths:
          - /your-service
        strip_path: true
        plugins:
          - name: custom_auth_plugin
            config:
              mode: "userAuth"
              userAuthEndpoint: "http://service-common:8080/endpoint-permissions"
              deviceAuthEndpoint: "http://service-common:8080/device-tokens/authenticate"
```

### Docker Compose Configuration

**File:** `backend/docker-compose.yaml`

Add your service container:

```yaml
service-your-service:
  image: ${YOUR_SERVICE_IMAGE:-ghcr.io/industrix-id/backend/service/your-service:latest}
  build:
    context: .
    dockerfile: ./services/your-service/docker/Dockerfile
    platforms:
      - linux/amd64
      - linux/arm64
    args:
      - ENVIRONMENT=${ENVIRONMENT}
      - GO_VERSION=${GO_VERSION}
      - PRODUCTION_IMAGE=${PRODUCTION_IMAGE}
  container_name: service-your-service
  ports:
    - "8094:8080"  # Choose an available port
  environment:
    - SERVICE=your-service
    - PORT=8080
    - ENV=${ENVIRONMENT}
    # ... other env vars
  networks:
    - industrix
  depends_on:
    - industrix-postgres
    - industrix-redis
  restart: unless-stopped
```

### Database Permission Registration

**File:** `backend/infra/database/migrations/XXXXXX_initial_data_your_service_permissions.up.sql`

Create a new migration to register your endpoints in the permissions system.

**Important:** Do NOT use hardcoded UUIDs for permission IDs. Let the database auto-generate them with `gen_random_uuid()`, then use subqueries to resolve IDs for translations.

```sql
-- ============================================================================
-- INSERT PERMISSIONS (IDs auto-generated by database)
-- ============================================================================
INSERT INTO common.permissions (endpoint_path, endpoint_action, action, resource, hide) VALUES
    ('/your-service/items', 'POST', 'create', 'items', false),
    ('/your-service/items', 'GET', 'list', 'items', false),
    ('/your-service/items/{id}', 'GET', 'show', 'items', false),
    ('/your-service/items/{id}', 'PUT', 'update', 'items', false),
    ('/your-service/items/{id}', 'DELETE', 'delete', 'items', false)
ON CONFLICT (action, resource) DO NOTHING;

-- ============================================================================
-- INSERT PERMISSION TRANSLATIONS
-- Note: Permission IDs are resolved using subqueries based on action + resource
-- ============================================================================
INSERT INTO common.permission_translations (permission_id, locale, name, description, resource) VALUES
    -- Create Item
    ((SELECT id FROM common.permissions WHERE action = 'create' AND resource = 'items' LIMIT 1), 'en', 'Create Item', 'Permission to create items', 'Items'),
    ((SELECT id FROM common.permissions WHERE action = 'create' AND resource = 'items' LIMIT 1), 'id', 'Buat Item', 'Hak akses untuk membuat item', 'Item'),
    -- List Items
    ((SELECT id FROM common.permissions WHERE action = 'list' AND resource = 'items' LIMIT 1), 'en', 'List Items', 'Permission to list items', 'Items'),
    ((SELECT id FROM common.permissions WHERE action = 'list' AND resource = 'items' LIMIT 1), 'id', 'Daftar Item', 'Hak akses untuk melihat daftar item', 'Item'),
    -- Show Item
    ((SELECT id FROM common.permissions WHERE action = 'show' AND resource = 'items' LIMIT 1), 'en', 'View Item Details', 'Permission to view item details', 'Items'),
    ((SELECT id FROM common.permissions WHERE action = 'show' AND resource = 'items' LIMIT 1), 'id', 'Lihat Detail Item', 'Hak akses untuk melihat detail item', 'Item'),
    -- Update Item
    ((SELECT id FROM common.permissions WHERE action = 'update' AND resource = 'items' LIMIT 1), 'en', 'Update Item', 'Permission to update items', 'Items'),
    ((SELECT id FROM common.permissions WHERE action = 'update' AND resource = 'items' LIMIT 1), 'id', 'Perbarui Item', 'Hak akses untuk memperbarui item', 'Item'),
    -- Delete Item
    ((SELECT id FROM common.permissions WHERE action = 'delete' AND resource = 'items' LIMIT 1), 'en', 'Delete Item', 'Permission to delete items', 'Items'),
    ((SELECT id FROM common.permissions WHERE action = 'delete' AND resource = 'items' LIMIT 1), 'id', 'Hapus Item', 'Hak akses untuk menghapus item', 'Item')
ON CONFLICT (permission_id, locale) DO NOTHING;

-- Optional: Insert permission dependencies (also using subqueries)
INSERT INTO common.permission_dependencies (permission_id, dependency_permission_id) VALUES
    -- Example: 'update' requires 'show' permission
    (
        (SELECT id FROM common.permissions WHERE action = 'update' AND resource = 'items' LIMIT 1),
        (SELECT id FROM common.permissions WHERE action = 'show' AND resource = 'items' LIMIT 1)
    )
ON CONFLICT (permission_id, dependency_permission_id) DO NOTHING;
```

**Important Notes:**
- Do NOT hardcode UUIDs - let the database generate them with `gen_random_uuid()`
- Use subqueries with `action` + `resource` to resolve permission IDs for translations and dependencies
- The `endpoint_path` must match your OpenAPI spec paths
- The `endpoint_action` is the HTTP method (GET, POST, PUT, DELETE, PATCH)
- The `action` + `resource` combination must be unique
- Always create both `.up.sql` and `.down.sql` migration files

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Related Guides

- [How to Write Service Layer](./how-to-write-service-layer.md)
- [How to Write Handlers](./how-to-write-handlers.md)
- [How to Use Configuration](./how-to-use-configuration.md)
- [How to Structure OpenAPI](./how-to-structure-openapi.md)
- [How to Understand Architecture](./how-to-understand-architecture.md)
