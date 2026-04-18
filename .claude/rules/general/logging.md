# Logging Rules

## Logger Initialization

Every package declares a package-level named logger using dot-notation:

```go
var logger = zap.S().Named("repositories.organizations")
var logger = zap.S().Named("services.ftm")
var logger = zap.S().Named("workers.outbox")
var logger = zap.S().Named("fx.mqtt_manager")
```

Format: `{layer}.{domain}` — matches the module's position in the architecture.

## Forbidden

Never use `fmt.Print`, `fmt.Printf`, `fmt.Println`, `log.Print`, `log.Println`, or `log.Printf` in application code (`pkg/`, `services/`, `workers/`, `sync-services/`). Only `zap.S()` or the package logger.

## Log Levels

**ERROR** — Operations that failed and need attention. Always include `zap.Error(err)`:
```go
logger.With(zap.Error(err)).Error("Failed to create device host")
```

**WARN** — Recoverable failures, degraded operations, handled edge cases:
```go
logger.With(zap.Error(err)).Warn("Failed to trigger device refresh after OTP generation")
```

**INFO** — Significant lifecycle events (startup, shutdown, major state changes):
```go
logger.Info("FTM service created successfully")
```

**DEBUG** — Diagnostic detail, intermediate steps, performance metrics:
```go
logger.Debug("MQTT user refresh completed",
    zap.String("user_id", userID.String()),
    zap.Duration("duration", time.Since(startTime)))
```

## Structured Fields

Always use typed zap fields. Never concatenate strings into messages:

```go
// Correct
logger.With(zap.Error(err), zap.String("device_id", deviceID.String())).
    Error("Failed to create device host")

// Wrong
logger.Error("Failed to create device host: " + err.Error() + " for device " + deviceID.String())
```

Common field types:
- `zap.Error(err)` — always for errors
- `zap.String("key", val)` — identifiers, names
- `zap.Int("count", n)` — counts, sizes
- `zap.Duration("elapsed", d)` — timing
- `zap.Bool("active", b)` — flags

## Sensitive Data

Never log passwords, tokens, full user objects, or API keys. Log identifiers (IDs, emails) only when needed for debugging.
