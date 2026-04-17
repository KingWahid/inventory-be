# Understanding the Architecture: Core Components

A deep dive into the core components used across all services.

## Table of Contents

- [Zap (Structured Logging)](#zap-structured-logging)
- [Echo (Web Framework)](#echo-web-framework)
- [Translation System](#translation-system)
- [Sentry (Error Tracking)](#sentry-error-tracking)
- [Middleware](#middleware)
- [Context (Request Context)](#context-request-context)
- [Key Concepts Summary](#key-concepts-summary)

---

## Zap (Structured Logging)

**What is Zap?** Zap is Uber's high-performance structured logging library. Instead of `fmt.Printf`, you use structured fields.

### How it works in your service:

```go
// In main.go, InitLogger sets up the global logger
initialization.InitLogger(cfg.LogLevel, cfg.Env, cfg.Service, sentryClient)

// This creates a logger with:
// - Level: "debug", "info", "warn", "error" (from config)
// - Development mode: prettier output in dev
// - Console encoding: human-readable format
// - Sentry integration: errors automatically sent to Sentry
```

### What is `zap.S()`?

`zap.S()` returns the **"Sugared Logger"** - a convenience wrapper around Zap's structured logger. The "S" stands for "Sugar" (sweeter/easier API).

Zap provides two logging APIs:

1. **Structured Logger** (`zap.L()`) - Type-safe, zero-allocation, but verbose
2. **Sugared Logger** (`zap.S()`) - Convenient, printf-style, but slightly slower

### How it's set up:

```go
// In InitLogger
logger, _ := zap.Config{...}.Build()
zap.ReplaceGlobals(logger)  // Sets up both zap.L() and zap.S()

// Now you can use:
zap.L().Info("message")  // Structured logger
zap.S().Info("message")  // Sugared logger (easier to use)
```

### Using Zap in your code:

```go
// Simple logging (sugared logger - easy!)
zap.S().Info("Service started")
zap.S().Error("Something went wrong")

// Printf-style formatting (sugared logger)
zap.S().Infof("User %s logged in", username)
zap.S().Errorf("Failed to connect: %v", err)
zap.S().Debugf("Processing request: %s", requestID)

// Structured logging (adds fields)
zap.S().With(
    zap.String("user_id", userID.String()),
    zap.Int("count", 42),
).Info("User processed")

// Error logging with context
zap.S().With(zap.Error(err)).Error("Failed to connect to database")

// Alternative: Structured logger (zap.L()) - more verbose but faster
zap.L().Info("Service started",
    zap.String("user_id", userID.String()),
    zap.Int("count", 42),
)
```

### Why use `zap.S()` (Sugared Logger)?

- **Easier to use** - Printf-style formatting like `fmt.Printf`
- **Less verbose** - No need to specify field types
- **Good for most cases** - Performance difference is negligible for most apps

### When to use `zap.L()` (Structured Logger)?

- **High-performance paths** - Zero-allocation logging
- **Type safety** - Compile-time checking of field types
- **Hot loops** - When logging millions of times per second

**In your codebase:** Most code uses `zap.S()` because it's easier and the performance difference doesn't matter for typical web services.

### What you see in logs:

```
2024-01-15T10:30:45.123Z    INFO    Service started
2024-01-15T10:30:46.456Z    ERROR   Failed to connect to database    {"error": "connection refused"}
2024-01-15T10:30:47.789Z    DEBUG   Processing request    {"request_id": "abc123", "user_id": "user-456"}
```

**Why Zap?**
- Fast (zero-allocation in hot paths)
- Structured (easy to parse/search)
- Levels (filter by severity)
- Integrates with Sentry automatically

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Echo (Web Framework)

**What is Echo?** Echo is a high-performance, minimalist Go web framework. Think Express.js but for Go.

### How it works:

```go
// Create Echo instance
app := echo.New()

// Register routes
app.GET("/users", getUsersHandler)
app.POST("/users", createUserHandler)

// Start server
app.Start(":8080")
```

### In your service, `NewEcho()` sets up:

```go
app := initialization.NewEcho(&cfg.DashboardUIHostname)
// This creates Echo with:
// 1. Custom error handler (translates errors, formats JSON)
// 2. Locale middleware (extracts language from headers)
// 3. Logging middleware (logs every request)
// 4. Sentry middleware (captures panics/errors)
// 5. CORS middleware (allows cross-origin requests)
```

### Echo Context (`echo.Context`):

Every handler receives an `echo.Context` which provides:

```go
func (h *ServerHandler) GetUsers(ctx echo.Context) error {
    // Get request info
    method := ctx.Request().Method
    path := ctx.Path()
    headers := ctx.Request().Header

    // Get query parameters
    page := ctx.QueryParam("page")
    limit := ctx.QueryParam("limit")

    // Get path parameters
    userID := ctx.Param("id")  // from /users/:id

    // Get request body (use common.BindRequestBody for proper error handling)
    var req CreateUserRequest
    common.BindRequestBody(ctx, &req)  // parses JSON and returns CustomError on failure

    // Get context (for timeouts, cancellation)
    goContext := ctx.Request().Context()

    // Set response
    ctx.JSON(200, response)  // JSON response
    ctx.String(200, "text")  // Plain text
    ctx.NoContent(204)        // No content

    // Get locale (from middleware)
    locale := translations.GetLocaleFromEchoContext(ctx)

    return nil  // or return error
}
```

### Middleware Chain:

Middleware runs in order before your handler:

```
Request arrives
  ↓
LocaleMiddleware()      // Extracts Accept-Language header, stores in context
  ↓
ZapSugaredLoggerMiddleware()  // Logs request details
  ↓
SentryMiddleware()     // Captures panics, sends to Sentry
  ↓
CORSMiddleware()       // Handles CORS headers
  ↓
Your Handler           // Your actual code
  ↓
Response sent
```

### Error Handling:

Echo has a custom error handler that:
- Catches all errors from handlers
- Translates error messages based on locale
- Formats errors as JSON
- Sets appropriate HTTP status codes

```go
// In handler, return error
return common.NewCustomError("User not found").
    WithErrorCode(errorcodes.NotFoundError).
    WithHTTPCode(http.StatusNotFound).
    WithMessageID("error_user_not_found")

// Error handler automatically:
// 1. Gets locale from context
// 2. Translates message using messageID
// 3. Returns JSON: {"message": "User not found", "code": "NOT_FOUND"}
```

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Translation System

### LoadServiceLocales

**What it does:** Loads translation files from your service into the global translation bundle.

### How it works:

```go
// In main.go
//go:embed locales/*
var rbacLocalesFS embed.FS

// Load translations
err = translations.LoadServiceLocales(rbacLocalesFS, "rbac")
```

### Step-by-step:

1. **Embed files at compile time:**
   ```go
   //go:embed locales/*
   var rbacLocalesFS embed.FS
   ```
   This embeds all files in `locales/` into the binary. No need to ship separate files.

2. **Load into bundle:**
   ```go
   translations.LoadServiceLocales(rbacLocalesFS, "rbac")
   ```
   - Reads `locales/en/rbac.yaml`
   - Reads `locales/id/rbac.yaml`
   - Parses YAML into translation bundle
   - Merges with existing translations (from `pkg/common`)

3. **Translation file format:**
   ```yaml
   # locales/en/rbac.yaml
   error_user_not_found:
     description: "User not found"
     other: "User not found"

   # locales/id/rbac.yaml
   error_user_not_found:
     description: "User not found"
     other: "Pengguna tidak ditemukan"
   ```

4. **Using translations:**
   ```go
   // In your code
   locale := translations.GetLocaleFromEchoContext(ctx)
   message := translations.TranslateByLocale(locale, "error_user_not_found", nil)
   // Returns: "User not found" (en) or "Pengguna tidak ditemukan" (id)
   ```

### Getting Locale in Handlers (IMPORTANT):

**Always use `translations.GetLocaleFromEchoContext(ctx)`** to get the locale in your handlers. The `LocaleMiddleware` has already parsed the `Accept-Language` header and stored the locale in the Echo context.

```go
// CORRECT: Use GetLocaleFromEchoContext (standard pattern)
func (h *ServerHandler) GetFeatures(ctx echo.Context, _ stub.GetFeaturesParams) error {
    // Get locale from Echo context (set by LocaleMiddleware)
    locale := translations.GetLocaleFromEchoContext(ctx)

    // Pass to service layer
    resp, err := h.service.GetFeatures(ctxTimeout, locale)
    // ...
}

// INCORRECT: Don't manually parse Accept-Language header
func (h *ServerHandler) GetFeatures(ctx echo.Context, params stub.GetFeaturesParams) error {
    // DON'T DO THIS - the middleware already handles this
    locale := translations.ParseAcceptLanguage(params.AcceptLanguage)
    // ...
}
```

**Why use `GetLocaleFromEchoContext`?**
- Consistent across all handlers
- Middleware already did the parsing work
- No need to pass `Accept-Language` through OpenAPI params
- Works even if OpenAPI spec doesn't define the header parameter

### Locale Detection:

The `LocaleMiddleware` extracts locale from HTTP headers:

```go
// Client sends:
Accept-Language: en-US,en;q=0.9,id;q=0.8

// Middleware:
locale := translations.ParseAcceptLanguage(header)
// Returns: "en" (best match)

// Stored in context:
ctx := context.WithValue(ctx, translations.ContextLocaleKey, "en")
```

### Translation Flow:

```
1. Client sends: Accept-Language: en
   ↓
2. LocaleMiddleware extracts "en" → stores in context
   ↓
3. Handler gets locale: translations.GetLocaleFromEchoContext(ctx)
   ↓
4. Handler passes locale to service layer
   ↓
5. Service/Repository uses locale for translations
   ↓
6. Response returned with translated content
```

**Why this system?**
- Single source of truth (YAML files)
- Automatic locale detection via middleware
- Fallback to default locale
- Embedded in binary (no external files needed)
- Consistent pattern across all handlers

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Sentry (Error Tracking)

**What is Sentry?** Sentry is a service that tracks errors, exceptions, and performance issues in production.

### How it's initialized:

```go
// In main.go
flush, err := initialization.InitSentry(cfg.SentryDsn, cfg.Env)
// defer flush ensures pending Sentry error reports are sent before service shuts down.
// Sentry sends errors asynchronously (in background). Without flush, errors queued
// during shutdown might be lost when the process exits immediately.
// The 2 second timeout gives Sentry time to send all pending events.
defer flush(2 * time.Second)
```

### Why `defer flush` is critical:

Sentry sends errors asynchronously (non-blocking). When your service shuts down:
- Without `defer flush`: Process exits immediately → pending errors lost
- With `defer flush`: Waits up to 2 seconds → all errors sent

**What happens:**
1. Service runs, errors occur, Sentry queues them
2. Service receives shutdown signal (Ctrl+C, SIGTERM, docker stop, etc.)
3. `main()` function is about to exit
4. `defer flush(2 * time.Second)` executes automatically
5. Sentry sends all pending errors (waits max 2 seconds)
6. `main()` actually exits

### What InitSentry does:

```go
sentry.Init(sentry.ClientOptions{
    Dsn:              cfg.SentryDsn,        // Your Sentry project URL
    EnableTracing:    isDevelopment,        // Performance monitoring
    TracesSampleRate: 1.0,                  // 100% in dev, 50% in prod
    Environment:      cfg.Env,              // "development", "production"
})
```

### Integration with Zap:

When you log an error with Zap, it automatically goes to Sentry:

```go
zap.S().With(zap.Error(err)).Error("Database connection failed")
// This:
// 1. Logs to console (via Zap)
// 2. Sends to Sentry (via zapsentry integration)
// 3. Includes stack trace, context, tags
```

### Sentry Middleware:

```go
app.Use(sentryecho.New(sentryecho.Options{
    Repanic: true,  // Re-throw panic after logging
}))
```

This middleware:
- Captures panics in handlers
- Sends to Sentry with request context
- Re-throws panic (so Echo's error handler can process it)

### What Sentry receives:

- Error message
- Stack trace
- Request details (method, path, headers)
- User context (if available)
- Environment tags
- Custom tags (service name, etc.)

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Middleware

**What is Middleware?** Functions that run before/after your handlers. They can:
- Modify requests
- Add data to context
- Log requests
- Handle errors
- Short-circuit requests (return early)

### How Echo middleware works:

```go
// Middleware is a function that returns a HandlerFunc
func MyMiddleware() echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            // BEFORE handler runs
            start := time.Now()

            // Call next middleware/handler
            err := next(c)

            // AFTER handler runs
            duration := time.Since(start)
            log.Printf("Request took %v", duration)

            return err
        }
    }
}

// Register middleware
app.Use(MyMiddleware())
```

### Middleware Order Matters:

```go
app.Use(LocaleMiddleware())        // Runs first
app.Use(ZapSugaredLoggerMiddleware())  // Runs second
app.Use(SentryMiddleware())        // Runs third
```

Each middleware can:
- Read/modify the request
- Add data to context
- Call `next(c)` to continue
- Return early (skip remaining middleware/handler)

### Example: LocaleMiddleware

```go
func LocaleMiddleware() echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            // Extract locale from header
            header := c.Request().Header.Get("Accept-Language")
            locale := translations.ParseAcceptLanguage(header)

            // Store in context
            ctx := context.WithValue(
                c.Request().Context(),
                translations.ContextLocaleKey,
                locale,
            )
            c.SetRequest(c.Request().WithContext(ctx))

            // Continue to next middleware/handler
            return next(c)
        }
    }
}
```

### Example: Logging Middleware

```go
func ZapSugaredLoggerMiddleware(sugar *zap.SugaredLogger) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            start := time.Now()

            // Call handler
            err := next(c)

            // Log after handler completes
            latency := time.Since(start)
            sugar.With(
                "method", c.Request().Method,
                "path", c.Request().RequestURI,
                "status", c.Response().Status,
                "duration_ms", latency.Milliseconds(),
            ).Debugf("Traffic Log")

            return err
        }
    }
}
```

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Context (Request Context)

**What is Context?** `context.Context` is Go's way of passing request-scoped data and cancellation signals.

**Why it's important:**
- Request timeouts
- Cancellation (client disconnects)
- Passing data (locale, user info, etc.)

### How it flows:

```
HTTP Request
  ↓
Echo creates context with timeout
  ↓
Middleware adds data (locale, etc.)
  ↓
Handler receives context
  ↓
Service layer uses context
  ↓
Repository uses context for DB queries
```

### Using Context:

```go
// In handler
func (h *ServerHandler) GetUsers(ctx echo.Context) error {
    // Get request context
    reqCtx := ctx.Request().Context()

    // Create context with timeout
    ctxTimeout, cancel := context.WithTimeout(
        reqCtx,
        constants.EndpointTimeout,  // e.g., 30 seconds
    )
    defer cancel()  // Important: always cancel!

    // Pass to service
    users, err := h.service.GetUsers(ctxTimeout, orgID)

    return ctx.JSON(200, users)
}

// In service
func (s *service) GetUsers(ctx context.Context, orgID uuid.UUID) ([]User, error) {
    // Context automatically cancels if:
    // - Timeout expires
    // - Client disconnects
    // - Parent context cancels

    // Pass to repository
    return s.userRepository.FindByOrganizationID(ctx, orgID)
}

// In repository
func (r *userRepository) FindByOrganizationID(
    ctx context.Context,
    orgID uuid.UUID,
) ([]User, error) {
    // GORM uses context for query cancellation
    var users []User
    err := r.db.WithContext(ctx).
        Where("organization_id = ?", orgID).
        Find(&users).Error

    return users, err
}
```

### Context Values (storing data):

```go
// Store in context
ctx := context.WithValue(
    parentCtx,
    translations.ContextLocaleKey,  // key
    "en",                          // value
)

// Retrieve from context
locale := ctx.Value(translations.ContextLocaleKey).(string)
```

### Context Best Practices:

- Always pass context as first parameter
- Always use `defer cancel()` for timeouts
- Don't store context in structs (pass as parameter)
- Use context for cancellation/timeouts, not just data storage

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Key Concepts Summary

- **Viper**: Configuration management (env vars, flags, files)
- **OpenAPI**: API specification that generates type-safe code
- **Echo**: Web framework (like Express.js)
- **Zap**: Structured logging with Sentry integration
- **Sentry**: Error tracking and monitoring
- **Translations**: i18n system with locale detection
- **Context**: Request-scoped data and cancellation
- **Middleware**: Functions that run before/after handlers
- **Service Layer**: Business logic (separate from HTTP)
- **Repository Pattern**: Database access abstraction
- **Common Module**: Shared utilities across services
- **Code Generation**: OpenAPI spec → Go types and routes

This architecture provides:
- Type safety (generated types)
- Separation of concerns (handlers vs service vs repository)
- Reusability (common module)
- Consistency (all services follow same pattern)
- Maintainability (single source of truth for API spec)
- Observability (logging, error tracking)
- Internationalization (multi-language support)

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Related Guides

- [How to Create a Service](./how-to-create-a-service.md)
- [How to Write Service Layer](./how-to-write-service-layer.md)
- [How to Write Handlers](./how-to-write-handlers.md)
- [How to Use Configuration](./how-to-use-configuration.md)
