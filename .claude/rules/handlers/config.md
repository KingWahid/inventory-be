---
paths:
  - "services/*/config/**"
  - "workers/config/**"
  - "sync-services/*/config/**"
---

# Application Config Rules

Every HTTP API service, worker, and sync service has a `config/config.go` that follows a strict structure. Do not deviate.

## Reference

Use an existing service config as your reference. Pick the one closest to your service's needs:

| If the service needs... | Reference |
|---|---|
| Standard HTTP API (DB, Redis, JWT) | `services/operation/config/config.go` |
| HTTP API + MQTT | `services/ftm/config/config.go` |
| HTTP API + storage (S3/Supabase) | `services/billing/config/config.go` |
| HTTP API + email providers | `services/notification/config/config.go` |
| HTTP API + multiple integrations | `services/common/config/config.go` |
| Worker with consumers | `workers/config/config.go` |
| Sync service (MQTT, no HTTP) | `sync-services/ftm/config/config.go` |

## Config Struct

### Field Ordering

Group fields by category, in this order. Use `mapstructure` tags for all fields.

1. **Core service** — `Service`, `Env`, `Port`, `LogLevel`
2. **Database** — `DBHost`, `DBPort`, `DBUser`, `DBPassword`, `DBName`, `DBSslMode`, `DBSslRootCert`
3. **Redis** — `RedisHost`, `RedisPort`, `RedisPassword`
4. **Authentication** — `JwtSecret`
5. **CORS** — `CorsAllowedOrigins`
6. **Caching** — `CachingEnabled` (or service-specific toggles like `UserCachingEnabled`)
7. **Observability** — `SentryDsn`
8. **Service-specific fields** — MQTT, storage, email, consumer config, etc.

### Field Type Rules

- Strings for hosts, passwords, keys, URLs, DSNs
- `int` for ports, counts, sizes
- `bool` for feature toggles
- `[]string` for lists (CORS origins, provider priorities)
- `time.Duration` for timeouts and intervals
- Never use `float32` — use `float64` if needed

## LoadConfig Function

The function follows a fixed four-phase pattern:

### Phase 1: Environment Bindings

Bind all env vars using the bindings slice pattern:

```go
bindings := []struct {
    key string
    env string
}{
    {"service", "SERVICE"},
    {"port", "PORT"},
    // ... all bindings
}

for _, b := range bindings {
    if err := viper.BindEnv(b.key, b.env); err != nil {
        return nil, common.NewCustomErrorf("failed to bind %s", b.key).
            WithErrorCode(errorcodes.InvalidRequest).
            WithError(err)
    }
}
```

Binding rules:
- `key` is lowercase with underscores (matches `mapstructure` tag)
- `env` is SCREAMING_SNAKE_CASE
- Error wrapping uses `common.NewCustomErrorf` with `errorcodes.InvalidRequest`

### Phase 2: Defaults

Set defaults grouped by category, matching the struct ordering:

```go
// Core
viper.SetDefault("service", "service-name")
viper.SetDefault("port", 8080)
viper.SetDefault("env", "development")

// Database
viper.SetDefault("db_host", "localhost")
viper.SetDefault("db_port", 5432)
viper.SetDefault("db_ssl_mode", "disable")

// Redis
viper.SetDefault("redis_host", "localhost")
viper.SetDefault("redis_port", 6379)
viper.SetDefault("redis_password", "")

// CORS
viper.SetDefault("cors_allowed_origins", []string{"https://dashboard.industrix.id"})

// Caching
viper.SetDefault("caching_enabled", true)
```

Default rules:
- `service` must match the directory name (e.g., `"authentication"`, `"ftm"`)
- `port` is `8080` for HTTP services, `8095` for worker health checks
- DB defaults always: host=localhost, port=5432, ssl_mode=disable
- Redis defaults always: host=localhost, port=6379, password=""
- Sync services set `cors_allowed_origins` to empty `[]string{}`

### Phase 3: Pflags

Register pflags for all config values, grouped the same way. Then parse and bind:

```go
pflag.Int("port", viper.GetInt("port"), "Server port")
// ... all pflags

pflag.Parse()

if err := viper.BindPFlags(pflag.CommandLine); err != nil {
    return nil, common.NewCustomError("failed to bind pflags").
        WithErrorCode(errorcodes.InvalidRequest).
        WithError(err)
}
```

### Phase 4: Unmarshal and Post-Processing

```go
var config Config
if err := viper.Unmarshal(&config); err != nil {
    return nil, common.NewCustomError("failed to load configuration").
        WithErrorCode(errorcodes.InitializationError).
        WithError(err)
}

// LogLevel is always derived from Env — never set directly
if config.Env == constants.EnvDevelopment {
    config.LogLevel = "debug"
} else {
    config.LogLevel = "info"
}

return &config, nil
```

## FX Module

Every config package exports two things:

```go
// ProvideConfig provides the Config for fx dependency injection.
func ProvideConfig() (*Config, error) {
    return LoadConfig()
}

// Module provides the config module for fx.
var Module = fx.Module("config",
    fx.Provide(ProvideConfig),
    fx.Provide(
        // Named dependencies grouped by category
    ),
)
```

### Named Dependencies

Provide config values as named FX dependencies using `fx.Annotate`. Group by category with comments:

```go
fx.Provide(
    // Database connection parameters
    fx.Annotate(func(c *Config) string { return c.DBHost }, fx.ResultTags(`name:"db_host"`)),
    fx.Annotate(func(c *Config) int { return c.DBPort }, fx.ResultTags(`name:"db_port"`)),
    // ... all DB fields

    // Redis connection parameters
    fx.Annotate(func(c *Config) string { return c.RedisHost }, fx.ResultTags(`name:"redis_host"`)),
    fx.Annotate(func(c *Config) int { return c.RedisPort }, fx.ResultTags(`name:"redis_port"`)),
    fx.Annotate(func(c *Config) string { return c.RedisPassword }, fx.ResultTags(`name:"redis_password"`)),
    fx.Annotate(func(_ *Config) int { return 0 }, fx.ResultTags(`name:"redis_db"`)),

    // Caching parameters
    fx.Annotate(func(c *Config) bool { return c.CachingEnabled }, fx.ResultTags(`name:"cachingEnabled"`)),

    // Server parameters
    fx.Annotate(func(c *Config) int { return c.Port }, fx.ResultTags(`name:"port"`)),
    fx.Annotate(func(c *Config) []string { return c.CorsAllowedOrigins }, fx.ResultTags(`name:"cors_allowed_origins"`)),

    // Observability parameters
    fx.Annotate(func(c *Config) string { return c.SentryDsn }, fx.ResultTags(`name:"sentryDsn"`)),
    fx.Annotate(func(c *Config) string { return c.Env }, fx.ResultTags(`name:"env"`)),
    fx.Annotate(func(c *Config) string { return c.LogLevel }, fx.ResultTags(`name:"logLevel"`)),
    fx.Annotate(func(c *Config) string { return c.Service }, fx.ResultTags(`name:"service"`)),

    // JWT middleware skip paths
    fx.Annotate(func(_ *Config) []string {
        return []string{
            "/ping",
            // Add service-specific public endpoints here
        }
    }, fx.ResultTags(`name:"jwt_skip_paths"`)),
),
```

### Named Dependency Rules

- `redis_db` is always hardcoded to `0` for HTTP services (cache DB)
- Workers use `redis_db: 0` for Asynq, separate `RedisStreamsDB` for event streams
- `cachingEnabled` is `true` for HTTP services, `false` for workers
- `jwt_skip_paths` always includes `/ping`, plus service-specific public endpoints
- Sync services do NOT provide `jwt_skip_paths` (no HTTP middleware)
- Use `func(_ *Config)` (underscore) when the value is hardcoded, not derived from config

## Workers: Sub-Module Config Pattern

Workers use a two-tier config approach. The main `workers/config/config.go` holds all env vars. Sub-modules define their own focused config structs with constructors:

```go
// workers/jobs/consumers/{name}/config.go
type Config struct {
    consumer.BaseConfig
    // Optional service-specific fields
}

func NewConfig(cfg *config.Config) *Config {
    return &Config{
        BaseConfig: consumer.BaseConfig{
            Enabled:        cfg.XxxConsumerEnabled,
            Group:          cfg.XxxConsumerGroup,
            MaxRetries:     cfg.OutboxMaxRetries,
            BatchSize:      10,
            IdempotencyTTL: cfg.IdempotencyTTL,
            HMACSecret:     cfg.EventHMACSecret,
            RedisHost:      cfg.RedisHost,
            RedisPort:      cfg.RedisPort,
            RedisPassword:  cfg.RedisPassword,
            RedisStreamsDB:  cfg.RedisStreamsDB,
        },
    }
}
```

Sub-module configs are instantiated via `NewConfig()` constructors, not FX providers.

## Forbidden

- Do not add `LogLevel` to env bindings or pflags — it is always derived from `Env`
- Do not use `fmt.Errorf` for config loading errors in HTTP services — use `common.NewCustomError`
- Do not provide config values as unnamed FX dependencies — all must use `fx.ResultTags` with a name
- Do not hardcode secrets in defaults (JWT secret in dev defaults is the sole exception)
- Do not add new config fields without corresponding env binding, default, and pflag
