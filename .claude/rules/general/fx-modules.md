# FX Module Rules

Every package that participates in dependency injection defines its wiring in a file named `MODULE.go` (uppercase). This file contains the `Params` struct, `Result` struct, `Provide` function, and `Module` variable. Nothing else.

## File Name

The file is always `MODULE.go` — uppercase. Disregard any existing files named `module.go` (lowercase); those are non-conforming.

## Structure

Every `MODULE.go` follows this exact layout:

```go
package <name>

import (
    "go.uber.org/fx"
    // ... only imports needed for DI wiring
)

// Params defines dependencies for <Thing>.
type Params struct {
    fx.In

    // Group fields by category with comments
    DB           *gorm.DB
    CacheManager *caches.CacheManager
    // Named dependencies use backtick tags
    CachingEnabled bool `name:"cachingEnabled"`
}

// Result defines what <Thing> provides.
type Result struct {
    fx.Out

    <Thing> <Interface>
}

// Provide creates a <Thing> instance for FX dependency injection.
func Provide(params Params) (Result, error) {
    logger.Debug("Creating <Thing>")

    instance, err := New<Thing>(params.Field1, params.Field2, ...)
    if err != nil {
        return Result{}, err
    }

    logger.Info("<Thing> created successfully")
    return Result{<Thing>: instance}, nil
}

// Module provides <Thing> via FX.
var Module = fx.Module("<module-name>",
    fx.Provide(Provide),
)
```

## Params Struct

- Embeds `fx.In` as the first field (no tag)
- Groups fields by category with comments: core infrastructure, repositories, named config
- Named dependencies use backtick struct tags: `` `name:"field_name"` ``
- Optional dependencies use: `` `name:"field_name" optional:"true"` ``
- Field names match the interface or type they inject (e.g., `UserRepo users.Repository`, not `Repo users.Repository`)

## Result Struct

- Embeds `fx.Out` as the first field (no tag)
- Exposes **interfaces**, not concrete types
- Field name matches the interface name (e.g., `Repository Repository`, `Service Service`)
- Multiple results are allowed when a package provides more than one component:

```go
type Result struct {
    fx.Out

    Service        BackgroundProcessService
    Repository     BackgroundProcessRepository
    EnqueueService EnqueueService
}
```

## Provide Function

### Naming

- Repositories and common packages: `Provide`
- Services: `ProvideService`
- App-layer handlers: `RegisterRoutes` (used with `fx.Invoke`, not `fx.Provide`)

### Signature

- Takes `Params` by value (not pointer) — add `//nolint:gocritic` if the linter warns about large structs
- Returns `(Result, error)` when construction can fail
- Returns `Result` (no error) when construction always succeeds

### Body

1. Log at DEBUG: `logger.Debug("Creating <Thing>")`
2. Call the actual constructor (`NewRepository`, `newServiceWithDependencies`, etc.)
3. Handle error if applicable
4. Log at INFO: `logger.Info("<Thing> created successfully")`
5. Return the Result

### Error Handling

Errors in Provide functions use `common.NewCustomError`:

```go
return Result{}, common.NewCustomError("failed to create JWT encoder").
    WithErrorCode(errorcodes.InitializationError).
    WithError(err)
```

## Module Variable

```go
var Module = fx.Module("<module-name>",
    fx.Provide(Provide),
)
```

### Module Name Convention

| Layer | Format | Example |
|-------|--------|---------|
| Repository | `"{entity}-repository"` | `"device-repository"`, `"user-repository"` |
| Service | `"{domain}"` | `"authentication"`, `"billing"`, `"access"` |
| Common utility | `"{utility}"` | `"cache"`, `"jwt"`, `"security"`, `"email"`, `"storage"` |
| App-layer handler | `"handler"` | `"handler"` |
| Worker component | `"{component}"` | `"outbox"`, `"asynq-server"` |
| Config | `"config"` | `"config"` |

### fx.Provide vs fx.Invoke

- **`fx.Provide`** — the module produces a dependency that others consume. Used for repositories, services, common utilities.
- **`fx.Invoke`** — the module runs a side effect (route registration, lifecycle hooks). Used for app-layer handler wiring.
- A module can use both when it provides components AND registers lifecycle hooks:

```go
var Module = fx.Module("outbox",
    fx.Provide(NewOutboxComponents),
    fx.Invoke(RegisterLifecycle),
)
```

## Layer-Specific Patterns

### Repositories (`pkg/database/repositories/*/MODULE.go`)

All repository modules look nearly identical:

```go
type Params struct {
    fx.In
    DB             *gorm.DB
    CacheManager   *caches.CacheManager
    CachingEnabled bool `name:"cachingEnabled"`
}

type Result struct {
    fx.Out
    Repository Repository
}
```

Every repository receives `*gorm.DB`, `*caches.CacheManager`, and `cachingEnabled` — even if it doesn't use caching. This keeps the interface uniform.

### Services (`pkg/services/*/MODULE.go`)

Service Params group dependencies by category:

```go
type ServiceParams struct {
    fx.In

    // Core infrastructure
    TxManager    transaction.Manager
    DB           *gorm.DB
    CacheManager *caches.CacheManager

    // Repositories (injected from repository modules)
    UserRepo    users.Repository
    OrgRepo     organizations.Repository

    // Named config dependencies
    PasswordResetBaseURL string `name:"password_reset_base_url"`
}
```

Services call `newServiceWithDependencies(...)` — an unexported constructor that takes all dependencies as individual parameters.

### Common Utilities (`pkg/common/*/MODULE.go` or `pkg/common/fx/*.go`)

Two sub-patterns:

1. **Standalone packages** (email, storage, background_processes) — use `MODULE.go` in the package directory
2. **Shared fx package** (`pkg/common/fx/`) — groups small utilities into individual files named by function (`cache.go`, `jwt.go`, `security.go`). Module variables are exported as `CacheModule`, `JWTModule`, `SecurityModule`.

### App-Layer Handlers (`services/*/fx/handler.go`)

Handler modules use `fx.Invoke` to register routes as a side effect:

```go
type HandlerParams struct {
    fx.In
    Echo    *echo.Echo
    Service authservice.Service
}

func RegisterRoutes(params HandlerParams) {
    handler := api.NewServerHandler(params.Service)
    stub.RegisterHandlers(params.Echo, handler)
}

var HandlerModule = fx.Module("handler",
    fx.Invoke(RegisterRoutes),
)
```

## App Composition (`services/*/cmd/main.go`)

The `main.go` composes all modules in a fixed order with section comments:

```go
app := fx.New(
    // FX logging
    commonfx.FxLoggerOption,

    // Configuration
    config.Module,

    // Initialization (Sentry before Logger)
    commonfx.SentryModule,
    commonfx.LoggerModule,
    commonfx.LocalesModule(locales.FS, "<service>"),

    // Infrastructure
    commonfx.PostgresModule,
    commonfx.RedisModule,

    // Shared utilities
    commonfx.CacheModule,
    commonfx.JWTModule,
    transaction.Module,

    // Repositories
    users.Module,
    organizations.Module,
    // ...

    // Business services
    authservice.Module,

    // App layer
    commonfx.EchoModule,
    commonfx.JWTMiddlewareModule,
    localfx.HandlerModule,
)
app.Run()
```

Composition order: **Config -> Init -> Infrastructure -> Utilities -> Repositories -> Services -> App layer**. Each module gets a comment stating what it provides.

## Forbidden

- Do not put business logic in `MODULE.go` — only DI wiring
- Do not use `fx.Supply` to inject config values — config values flow through named dependencies from `config.Module`
- Do not use anonymous functions in `fx.Provide` inside module packages — always use a named `Provide` function with Params/Result structs
- Do not mix `fx.Provide` and `fx.Invoke` unless the module genuinely needs lifecycle hooks
- Do not create a `MODULE.go` for packages that are never injected (pure utility packages with no state)
