# Codebase Conventions

This project follows specific patterns and conventions. Review code against these established practices.

## Error Handling

### CustomError Pattern
This codebase uses a fluent CustomError pattern with method chaining:

```go
return common.NewCustomError("Database connection is nil").
    WithMessageID("error_db_connection_nil").
    WithErrorCode(errorcodes.InitializationError).
    WithHTTPCode(http.StatusInternalServerError)
```

**Key conventions:**
- Always use `common.NewCustomError()` or `common.NewCustomErrorFromError()`
- Chain methods for error code, HTTP code, and message ID
- Use `WithMessageID()` for translation support
- Use `WithErrorCode()` with predefined constants from `errorcodes` package
- Use `WithHTTPCode()` to set appropriate HTTP status

**Common mistakes to catch:**
- Returning plain `errors.New()` or `fmt.Errorf()` instead of CustomError
- Missing error codes on CustomError
- Incorrect HTTP status codes (e.g., 500 for validation errors)

### Error Translation
Errors support internationalization via message IDs:

```go
ce.WithMessageID("error_user_not_found").
   WithMessageData(map[string]interface{}{"userId": userID.String()})
```

## Logging

### Structured Logging with Zap
Always use structured logging with `zap.S()`:

```go
zap.S().With(zap.Error(err)).Error("Failed to build access cards cache key")
zap.S().Info("User logged in", "userId", userID, "organizationId", orgID)
```

**Key conventions:**
- Use `zap.S()` (sugared logger) for most logging
- Use `.With(zap.Error(err))` to include error context
- Use structured key-value pairs, not string concatenation
- Log levels: `Error()`, `Warn()`, `Info()`, `Debug()`

**Common mistakes:**
- Using `fmt.Printf()` or `log.Println()` instead of zap
- Logging sensitive data (passwords, tokens, full user objects)
- Missing context (userID, requestID) in error logs

## Context Management

### Context as First Parameter
Always pass `context.Context` as the first parameter:

```go
func GetAccessCardByID(ctx context.Context, organizationID, cardID uuid.UUID) (*schemas.UserAccessCard, error)
```

### Transaction Context
Transactions are passed via context and automatically used by repositories:

```go
// In service layer
ctx = transaction.WithTx(ctx, tx)

// Repository detects transaction from context
err := r.CreateAccessCard(ctx, card)
```

**Key conventions:**
- Use `transaction.WithTx(ctx, tx)` to attach transaction
- Repositories automatically use transaction if present in context
- Don't pass `*gorm.DB` or `*sql.Tx` as method parameters

## Repository Pattern

### Interface + Implementation
All repositories follow interface + private implementation pattern:

```go
// Public interface
type AccessCardRepository interface {
    GetAccessCardByID(ctx context.Context, organizationID, cardID uuid.UUID) (*schemas.UserAccessCard, error)
}

// Private implementation
type accessCardRepository struct {
    *BaseRepository
    db *gorm.DB
}
```

### Organization Scoping
Most methods validate organization ownership:

```go
// Scoped to organization (normal operations)
GetAccessCardByID(ctx context.Context, organizationID, cardID uuid.UUID) (*schemas.UserAccessCard, error)

// Unscoped (admin operations)
GetAccessCardByIDUnscoped(ctx context.Context, cardID uuid.UUID) (*schemas.UserAccessCard, error)
```

**Review for:**
- Missing organization checks in scoped methods
- Using unscoped methods where scoped is appropriate
- Admin-only methods lacking documentation or access controls

## Type Converters

### Stub ↔ Domain Type Conversion

HTTP handlers use converters to transform between OpenAPI-generated stub types and domain types. This pattern keeps `pkg/services` independent of API stubs, preventing circular dependencies.

```go
// In services/*/api/converters.go

// =============================================================================
// Request Converters (stub → domain)
// =============================================================================

// FromStubCreateRequest converts stub.CreateRequest to types.CreateRequest.
func FromStubCreateRequest(r *stub.CreateRequest) *types.CreateRequest {
    if r == nil {
        return nil
    }
    return &types.CreateRequest{
        Name:  r.Name,
        Email: r.Email,
    }
}

// =============================================================================
// Response Converters (domain → stub)
// =============================================================================

// ToStubResponse converts types.Response to stub.Response.
func ToStubResponse(r *types.Response) *stub.Response {
    if r == nil {
        return nil
    }
    return &stub.Response{
        ID:   stub.UUID(r.ID),
        Name: r.Name,
    }
}

// =============================================================================
// Internal Helper Converters
// =============================================================================

// toStubItems converts []types.Item to []stub.Item (unexported helper).
func toStubItems(items []types.Item) []stub.Item {
    result := make([]stub.Item, len(items))
    for i := range items {
        result[i] = toStubItem(&items[i])
    }
    return result
}
```

### Handler Usage Pattern

In HTTP handlers, convert at the boundary and dereference when calling services:

```go
func (h *Handler) PostResource(ctx echo.Context) error {
    var req stub.CreateRequest
    if err := common.BindRequestBody(ctx, &req); err != nil {
        return err
    }

    // Convert stub → domain (pass pointer, get pointer back)
    domainReq := FromStubCreateRequest(&req)

    // Call service with dereferenced value
    response, err := h.service.Create(ctxTimeout, *domainReq)
    if err != nil {
        return err
    }

    // Convert domain → stub for response
    return ctx.JSON(http.StatusCreated, ToStubResponse(response))
}
```

**Key conventions:**
- **Pointer parameters**: All `FromStub*` functions take pointer params (`*stub.X`)
- **Nil checks**: Always check for nil and return nil early
- **Pointer returns**: Return pointers (`*types.X`) for consistency
- **Dereference at call site**: Use `*domainReq` when passing to service methods that expect values
- **Three sections**: Organize converters into Request, Response, and Internal Helper sections
- **Naming**: `FromStub*` for stub→domain, `ToStub*` for domain→stub
- **Unexported helpers**: Internal recursive/helper converters are lowercase (`toStubItems`)

**File locations:**
- `services/*/api/converters.go` - Service-specific converters
- `pkg/services/types/*.go` - Domain types (no stub dependencies)

**Common mistakes:**
- Importing stubs in `pkg/services/*` (creates circular dependency)
- Forgetting nil checks on pointer parameters
- Not dereferencing converter results when service expects value types
- Mixing pointer and value semantics inconsistently

## Caching

### Cache-Aside Pattern
The codebase extensively uses caching with a cache-aside pattern:

```go
err = r.GetFromCacheOrDB(
    ctx,
    cacheKey,
    caches.AccessCardListCacheTTL,
    &result,
    func() error {
        // DB query logic here
        return query.Find(&result.Cards).Error
    },
)
```

**Key conventions:**
- Use `GetFromCacheOrDB()` helper from BaseRepository
- Build cache keys with `caches.Build*CacheKey()` functions
- Invalidate caches on write operations via `Invalidate*Cache()` methods
- Cache TTLs are defined as constants (e.g., `caches.AccessCardListCacheTTL`)

**Common mistakes:**
- Forgetting to invalidate cache on updates/deletes
- Caching data that changes frequently
- Missing cache key generation errors handling

## Database Operations

### GORM Patterns
This project uses GORM v2:

```go
// Always use WithContext
query := r.db.WithContext(ctx).Model(&schemas.UserAccessCard{})

// Use placeholder queries, not string concatenation
query.Where("user_id = ? AND deleted_at IS NULL", userID)

// Handle soft deletes explicitly
query.Where("deleted_at IS NULL")
```

**Common mistakes to catch:**
- Missing `WithContext(ctx)`
- String concatenation in WHERE clauses (SQL injection risk)
- Forgetting soft delete filters (`deleted_at IS NULL`)
- Not checking `result.Error` after queries

### Transactions
When operations need atomicity, use explicit transactions:

```go
err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
    if err := tx.Create(&record).Error; err != nil {
        return err
    }
    // More operations...
    return nil
})
```

## Validation

### UUID Validation
UUIDs are used extensively:

```go
import "github.com/google/uuid"

// Parse and validate
userID, err := uuid.Parse(userIDStr)
if err != nil {
    return common.NewCustomError("Invalid user ID")
}
```

**Common mistakes:**
- Not validating UUID strings before parsing
- Comparing UUIDs as strings instead of using `uuid.UUID` type

### Input Validation
Use validator package for struct validation:

```go
import "github.com/go-playground/validator/v10"

type CreateUserRequest struct {
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=8"`
}
```

## Code Organization

### File Structure
- `/pkg/database/repositories/` - Data access layer
- `/pkg/database/schemas/` - Database models (GORM)
- `/pkg/database/constants/` - Database constant definitions (enums, status types, background process types)
- `/pkg/database/base/` - Base repository with shared operations (caching, soft delete, pagination)
- `/pkg/database/db_utils/` - Error classification, query helpers, pagination utilities
- `/pkg/common/` - Shared utilities, errors, validation
- `/pkg/common/caches/` - Caching logic
- `/pkg/scheduler/` - Scheduled job engine (cron scheduler, handler registry, distributed locking)
- `/pkg/eventbus/` - Redis Streams event bus (consumer, idempotency, streams, events)
- `/infra/database/` - Migrations, seeds

### Naming Conventions
- **Interfaces**: `AccessCardRepository` (no "I" prefix)
- **Implementations**: `accessCardRepository` (private, lowercase)
- **Methods**: Use clear action verbs (`GetAccessCardByID`, not `FetchCard`)
- **Scoped variants**: Add `Unscoped` suffix for admin methods
- **FX module files**: Always `MODULE.go` (uppercase) — contains `Params`, `Result`, `Provide`, and `Module` variable

## Testing

### Test Files
- Unit tests: `*_test.go` alongside source files
- Integration tests: `*_integration_test.go`
- Use testify/suite for test organization
- Mock interfaces with mockery

## Security

### Critical Patterns
- **Never concatenate SQL**: Use parameterized queries
- **Always validate organization scope**: Except in explicit unscoped methods
- **Don't log sensitive data**: No passwords, tokens, full user objects
- **Validate UUIDs**: Parse before using in queries
- **Check error returns**: Especially for Close(), Rollback()

## Floating-Point Precision

### Use float64 for Domain Types

**CRITICAL:** Always use `float64` (not `float32`) for floating-point fields in domain types. `float32` causes precision loss for values that cannot be exactly represented in IEEE 754 single precision.

**Problem demonstration:**
```go
// float32 cannot represent 0.35 exactly
val := float32(0.35)
fmt.Printf("%.20f\n", val)  // Output: 0.34999999403953552246

// float64 has sufficient precision
val64 := float64(0.35)
fmt.Printf("%.20f\n", val64)  // Output: 0.35000000000000000000
```

### Domain Types Pattern

Domain types in `pkg/services/types/` must use `float64`:

```go
// ✓ Good: Domain types use float64
type DeviceConfig struct {
    KFactor              *float64  `json:"k_factor,omitempty"`
    CriticalThresholdBar *float64  `json:"critical_threshold_bar,omitempty"`
}

type Site struct {
    Latitude  *float64  `json:"latitude,omitempty"`
    Longitude *float64  `json:"longitude,omitempty"`
}

// ✗ Bad: float32 causes precision loss
type DeviceConfig struct {
    KFactor              *float32  `json:"k_factor,omitempty"`  // WRONG!
}
```

### API Boundary Conversion

OpenAPI/AsyncAPI stubs may use `float32` due to spec definitions. Convert at the API boundary only:

```go
// services/*/api/converters.go

// Request: Convert float32 (stub) → float64 (domain)
func FromStubRequest(r *stub.Request) *types.Request {
    result := &types.Request{}
    if r.Value != nil {
        val := float64(*r.Value)  // Convert up to float64
        result.Value = &val
    }
    return result
}

// Response: Convert float64 (domain) → float32 (stub)
func ToStubResponse(r *types.Response) *stub.Response {
    result := &stub.Response{}
    if r.Value != nil {
        val := float32(*r.Value)  // Convert down only at API boundary
        result.Value = &val
    }
    return result
}
```

### Precision-Sensitive Fields

Fields that commonly require float64 precision:
- **GPS coordinates** (latitude, longitude) - sub-meter precision requires float64
- **Financial values** (prices, amounts) - use `string` or `decimal.Decimal` for exact precision
- **Sensor readings** (flow rates, volumes, thresholds) - measurement precision matters
- **Percentages/ratios** (growth rates, discounts) - calculation accuracy

### Wire Protocol Exceptions

MQTT/AsyncAPI may define float32 for bandwidth efficiency with embedded devices. In these cases:
1. Accept float32 from wire protocol
2. Convert to float64 immediately at service boundary
3. Process internally with float64
4. Convert back to float32 only when sending to devices

```go
// sync-services handler receiving from MQTT
func (h *Handler) HandleFlowData(ctx context.Context, msg asyncapi.FlowDataMessage) error {
    // Convert float32 (wire format) → float64 (service layer)
    err := h.svc.HandleFlowData(
        ctx,
        deviceID,
        float64(msg.Payload.FlowRate),  // Convert immediately
        float64(msg.Payload.Volume),
    )
    return err
}
```

## Go Conventions

### Standard Practices
- Error handling: Check immediately, don't defer
- Defer for cleanup: `defer rows.Close()`, `defer file.Close()`
- Return early: Avoid deep nesting
- Exported vs unexported: Capital for public, lowercase for private
