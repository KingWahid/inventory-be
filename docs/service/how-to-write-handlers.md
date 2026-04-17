# How to Write API Handlers

Handlers in the `api/` directory translate HTTP requests to service calls.

## Table of Contents

- [How to Write API Handlers](#how-to-write-api-handlers)
  - [Table of Contents](#table-of-contents)
  - [Handler File Organization](#handler-file-organization)
  - [API Layer Responsibilities](#api-layer-responsibilities)
  - [Handler Structure](#handler-structure)
  - [Handler Initialization Order](#handler-initialization-order)
  - [Error Handling in Handlers](#error-handling-in-handlers)
  - [Validation Placement](#validation-placement)
  - [Reusing pkg/common Utilities](#reusing-pkgcommon-utilities)
  - [Request Flow](#request-flow)
  - [JWT Middleware Configuration](#jwt-middleware-configuration)
  - [Related Guides](#related-guides)

---

## Handler File Organization

**Split handlers into multiple files by resource/domain** for maintainability. Each file should contain handlers for related functionality.

**Required file structure:**

```
api/
├── handlers.go          # Base struct, constructor, and Ping ONLY
├── auth.go              # Authentication handlers (SignIn, SignOut)
├── passwords.go         # Password management (ForgotPassword, ResetPassword)
├── users.go             # User CRUD handlers
├── devices.go           # Device handlers
└── ...                  # One file per resource domain
```

**handlers.go (base file) - ALWAYS contains only these:**

```go
// Package api provides HTTP handlers for [service-name] service endpoints
package api

import (
    "net/http"

    "github.com/labstack/echo/v4"

    "[service]/service"
    "[service]/stub"
)

// ServerHandler implements the [service-name] API handlers.
type ServerHandler struct {
    service service.Service
}

// NewServerHandler creates a new server handler with the provided service.
func NewServerHandler(svc service.Service) stub.ServerInterface {
    return &ServerHandler{
        service: svc,
    }
}

// Ping handles the health check endpoint.
func (h *ServerHandler) Ping(ctx echo.Context) error {
    return ctx.JSON(http.StatusOK, map[string]string{
        "message": "pong",
    })
}
```

**Resource-specific handler file (e.g., users.go):**

```go
package api

import (
    "context"
    "net/http"

    "github.com/labstack/echo/v4"

    "github.com/industrix-id/backend/pkg/common/constants"
    "github.com/industrix-id/backend/pkg/common/errorcodes"
    "github.com/industrix-id/backend/pkg/common/jwt"
    "[service]/stub"
)

// GetUsers handles GET /users endpoint.
func (h *ServerHandler) GetUsers(ctx echo.Context, params stub.GetUsersParams) error {
    // Handler implementation...
}

// CreateUser handles POST /users endpoint.
func (h *ServerHandler) CreateUser(ctx echo.Context) error {
    // Handler implementation...
}
```

**Why split handlers?**

| Benefit | Explanation |
|---------|-------------|
| **Maintainability** | Each file focuses on one resource domain |
| **Code Review** | Changes to users don't pollute device file diffs |
| **Navigation** | Easy to find handlers by resource name |
| **Conflict Reduction** | Multiple devs can work on different resources |
| **Consistency** | Same pattern across all services |

**Naming convention:**

| Resource | File Name | Contains |
|----------|-----------|----------|
| Base | `handlers.go` | ServerHandler struct, NewServerHandler, Ping |
| Authentication | `auth.go` | SignIn, SignOut, token refresh |
| Passwords | `passwords.go` | ForgotPassword, ResetPassword, ChangePassword |
| Users | `users.go` | User CRUD operations |
| Devices | `devices.go` | Device CRUD operations |
| Dashboard | `dashboard.go` | Dashboard/analytics endpoints |

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## API Layer Responsibilities

The API layer (handlers) should be **thin** and only handle HTTP-specific concerns:

| API Layer (handlers) | Service Layer |
|---------------------|---------------|
| Request binding (`common.BindRequestBody()`) | Business logic |
| JWT token extraction | Validation of business rules |
| HTTP response formatting | Authorization checks |
| HTTP status code mapping | Data transformation |
| **Simple type conversion** (e.g., `openapi_types.UUID` → `uuid.UUID`) | **Complex validation** (e.g., "is this feature valid?") |
| **Pass optional params directly** (e.g., `params.IsActive`) | **Handle nil defaults** (e.g., `if isActive == nil { ... }`) |

**Important: Keep handlers thin - avoid unnecessary intermediate variables.**

```go
// CORRECT: Simple inline conversion for type aliases
func (h *ServerHandler) GetDeviceConfig(ctx echo.Context, id openapi_types.UUID, params stub.GetDeviceConfigParams) error {
    // ... JWT, timeout, claims setup ...

    capabilityOnly := params.CapabilityOnly != nil && *params.CapabilityOnly
    
    response, err := h.service.GetDeviceConfig(
        ctxTimeout,
        claims.OrganizationID,
        uuid.UUID(id),                                          // Inline conversion - openapi_types.UUID is alias for uuid.UUID
        capabilityOnly, 
        locale,
    )
    // ...
}

// WRONG: Unnecessary intermediate variables
func (h *ServerHandler) GetDeviceConfig(ctx echo.Context, id openapi_types.UUID, params stub.GetDeviceConfigParams) error {
    // ... JWT, timeout, claims setup ...
    
    deviceID := uuid.UUID(id) // DON'T DO THIS - unnecessary variable
    capabilityOnly := params.CapabilityOnly != nil && *params.CapabilityOnly

    response, err := h.service.GetDeviceConfig(
        ctxTimeout,
        claims.OrganizationID,
        deviceID,
        capabilityOnly,
        locale,
    )
    // ...
}
```

**When to pass pointers vs values to service:**

| Scenario | Pass to service | Example |
|----------|-----------------|---------|
| Bool where nil = false | `bool` (dereference inline) | `capabilityOnly` - nil and false behave the same |
| Bool where nil ≠ false | `*bool` (pass directly) | Feature flag where default is true |
| Pagination params | `*int` (pass directly) | nil = use server default, vs explicit value |

**Important: Keep validation in the service layer, not the API layer.**

```go
// WRONG: Validation logic in API handler
func (h *ServerHandler) CreateOrganization(ctx echo.Context, _ stub.CreateOrgParams) error {
    var req OrganizationCreateExtended
    common.BindRequestBody(ctx, &req)

    // DON'T DO THIS - validation belongs in service layer
    if req.FeatureIDs != nil {
        for _, idStr := range *req.FeatureIDs {
            featureID, err := uuid.Parse(idStr)
            if err != nil {
                return ctx.JSON(http.StatusBadRequest, stub.Error{Message: "Invalid feature ID"})
            }
            // Checking if feature exists - THIS IS BUSINESS LOGIC!
            exists, _ := h.featureRepo.Exists(featureID)
            if !exists {
                return ctx.JSON(http.StatusBadRequest, stub.Error{Message: "Feature not found"})
            }
        }
    }
    // ...
}

// CORRECT: API layer just passes data to service, service validates
func (h *ServerHandler) CreateOrganization(ctx echo.Context, _ stub.CreateOrgParams) error {
    token, err := jwt.GetJWTFromEchoContext(ctx)
    if err != nil {
        return err
    }

    ctxTimeout, cancel := context.WithTimeout(ctx.Request().Context(), constants.EndpointTimeout)
    defer cancel()

    claims, err := h.service.DecodeAuthenticationToken(ctxTimeout, token)
    if err != nil {
        return err
    }

    var req OrganizationCreateExtended
    if err := common.BindRequestBody(ctx, &req); err != nil {
        return err
    }

    // Pass raw data to service - service handles all validation
    result, err := h.service.CreateOrganization(
        ctxTimeout,
        claims.OrganizationID,
        &req,
    )
    if err != nil {
        return err
    }

    return ctx.JSON(http.StatusCreated, result)
}
```

**Why this matters:**

1. **Single Responsibility**: API layer handles HTTP, service handles business logic
2. **Testability**: Service logic can be unit tested without HTTP
3. **Reusability**: Same service method can be called from different contexts (API, CLI, queue worker)
4. **Consistency**: All validation errors come from one place with consistent error codes
5. **Security**: Business rules are enforced regardless of how the service is called

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Handler Structure

```go
// 1. Handler struct holds service reference
type ServerHandler struct {
    service service.Service
}

// 2. Constructor
func NewServerHandler(svc service.Service) stub.ServerInterface {
    return &ServerHandler{service: svc}
}

// 3. Implement ServerInterface methods (from generated stub)
func (h *ServerHandler) GetUsers(ctx echo.Context, params stub.GetUsersParams) error {
    // 1. Extract JWT token from request
    token, err := jwt.GetJWTFromEchoContext(ctx)
    if err != nil {
        return err  // Already returns proper CustomError
    }

    // 2. Create context with timeout
    ctxTimeout, cancel := context.WithTimeout(ctx.Request().Context(), constants.EndpointTimeout)
    defer cancel()

    // 3. Decode JWT token to get claims (using service's thin wrapper)
    claims, err := h.service.DecodeAuthenticationToken(ctxTimeout, token)
    if err != nil {
        return err
    }

    // 4. Call service layer with claims (NOT token)
    users, err := h.service.GetUsers(ctxTimeout, claims, claims.OrganizationID, params.Page, params.Limit)
    if err != nil {
        return err  // Service returns CustomError
    }

    // 5. Return JSON response (service already returns stub types)
    return ctx.JSON(http.StatusOK, users)
}
```

**Key Points:**
- Handlers implement `stub.ServerInterface` (from generated code)
- Extract JWT token and decode claims in API layer
- **Use `DecodeAuthenticationToken` to get claims, then pass claims to service methods**
- Call service layer methods with claims (NOT auth tokens)
- Service returns stub types directly - no conversion needed
- Handle errors using `common.CustomError` (NOT `stub.Error`)

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Handler Initialization Order

**IMPORTANT: Follow this exact order for handler initialization steps.**

Every handler should follow this order:

1. **JWT Token Extraction** - First thing after function signature
2. **Context Timeout Setup** - Create timeout context for service calls
3. **JWT Claims Decoding** - Decode token to get claims using service wrapper
4. **Locale Extraction** - If the handler needs localized responses
5. **Request Body Binding** - Parse and validate request body (if any)

```go
// CORRECT: Proper initialization order
func (h *ServerHandler) CreateUser(ctx echo.Context) error {
    // 1. JWT extraction FIRST - fail fast if not authenticated
    token, err := jwt.GetJWTFromEchoContext(ctx)
    if err != nil {
        return err
    }

    // 2. Context timeout setup
    ctxTimeout, cancel := context.WithTimeout(ctx.Request().Context(), constants.EndpointTimeout)
    defer cancel()

    // 3. Decode JWT to get claims
    claims, err := h.service.DecodeAuthenticationToken(ctxTimeout, token)
    if err != nil {
        return err
    }

    // 4. Locale extraction (if needed for translations)
    locale := translations.GetLocaleFromEchoContext(ctx)

    // 5. Request body binding LAST
    var req stub.UserCreate
    if err := common.BindRequestBody(ctx, &req); err != nil {
        return err
    }

    // Now call service with claims (NOT token)
    result, err := h.service.CreateUser(ctxTimeout, claims, claims.OrganizationID, &req, locale)
    if err != nil {
        return err
    }

    return ctx.JSON(http.StatusCreated, result)
}

// WRONG: Request binding before authentication
func (h *ServerHandler) CreateUser(ctx echo.Context) error {
    var req stub.UserCreate
    if err := common.BindRequestBody(ctx, &req); err != nil {  // DON'T do this first!
        return err
    }

    token, err := jwt.GetJWTFromEchoContext(ctx)
    // ...
}
```

**Why this order matters:**

| Order | Step | Reason |
|-------|------|--------|
| 1 | JWT Token | Fail fast - reject unauthenticated requests before doing any work |
| 2 | Context Timeout | Must be set up before any service calls (including JWT decoding) |
| 3 | JWT Claims | Decode token to get claims - needed to extract orgID and userID |
| 4 | Locale | Needed for translations (after timeout context is ready) |
| 5 | Request Body | Only parse if authenticated and ready to process |

**Handlers without request body:**

For GET, DELETE, or other handlers without request body, skip step 5:

```go
func (h *ServerHandler) GetUsers(ctx echo.Context, params stub.GetUsersParams) error {
    // 1. JWT extraction
    token, err := jwt.GetJWTFromEchoContext(ctx)
    if err != nil {
        return err
    }

    // 2. Context timeout
    ctxTimeout, cancel := context.WithTimeout(ctx.Request().Context(), constants.EndpointTimeout)
    defer cancel()

    // 3. Decode JWT to get claims
    claims, err := h.service.DecodeAuthenticationToken(ctxTimeout, token)
    if err != nil {
        return err
    }

    // 4. Locale extraction
    locale := translations.GetLocaleFromEchoContext(ctx)

    // 5. No body binding needed - call service directly with claims
    result, err := h.service.GetUsers(ctxTimeout, claims, claims.OrganizationID, locale, params.Page, params.Limit)
    if err != nil {
        return err
    }

    return ctx.JSON(http.StatusOK, result)
}
```

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Error Handling in Handlers

**IMPORTANT: Never manually create `stub.Error` in handlers.** Always use `common.CustomError` and let the central error handler format the response.

**Why?**
- Central error handler auto-translates errors based on locale
- Consistent error format across all services
- Less code duplication
- Error codes and HTTP status managed in one place

**Standard Error Pattern:**

```go
// CORRECT: Return common.CustomError, let central handler format it
func (h *ServerHandler) CreateUser(ctx echo.Context) error {
    // 1. JWT extraction FIRST - fail fast if not authenticated
    token, err := jwt.GetJWTFromEchoContext(ctx)
    if err != nil {
        return err  // Just return it - already has MessageID for translation
    }

    // 2. Context timeout setup
    ctxTimeout, cancel := context.WithTimeout(ctx.Request().Context(), constants.EndpointTimeout)
    defer cancel()

    // 3. Decode JWT to get claims
    claims, err := h.service.DecodeAuthenticationToken(ctxTimeout, token)
    if err != nil {
        return err
    }

    // 4. Request body binding LAST
    var req stub.UserCreate
    if err := common.BindRequestBody(ctx, &req); err != nil {
        return err  // common.BindRequestBody already returns proper CustomError
    }

    // Service call with claims (NOT auth token)
    result, err := h.service.CreateUser(ctxTimeout, claims, claims.OrganizationID, &req)
    if err != nil {
        return err  // Just return it
    }

    return ctx.JSON(http.StatusCreated, result)
}

// WRONG: Manually creating stub.Error
func (h *ServerHandler) CreateUser(ctx echo.Context) error {
    token, err := jwt.GetJWTFromEchoContext(ctx)
    if err != nil {
        // DON'T DO THIS - bypasses translation and central error handling
        return ctx.JSON(http.StatusUnauthorized, stub.Error{
            Message:           "Missing authorization header",
            TranslatedMessage: translations.TranslateByLocale(locale, "error_...", nil),
            Code:              int32(errorcodes.Unauthorized),
        })
    }
    // ...
}
```

**JWT Token Extraction:**

`jwt.GetJWTFromEchoContext()` already returns a properly formatted `CustomError` with `MessageID`, so handlers just need:

```go
token, err := jwt.GetJWTFromEchoContext(ctx)
if err != nil {
    return err  // That's it! Error is auto-translated by central handler
}
```

**Common Error Patterns:**

| Scenario | Pattern |
|----------|---------|
| Request binding fails | `return err` (`common.BindRequestBody` already returns proper CustomError) |
| JWT extraction fails | `return err` (jwt package already returns proper CustomError) |
| Service error | `return err` (service already returns proper CustomError) |
| Pagination validation | `return err` (`utils.ValidatePaginationParamsWithDefaults` already returns proper CustomError) |

**Error Schema (OpenAPI):**

All services use the same standardized Error schema:

```yaml
Error:
  type: object
  required: [message, translated_message, code]
  properties:
    message:
      type: string
      description: English error message (for developers/logs)
    translated_message:
      type: string
      description: Localized error message (for end-user display)
    code:
      type: integer
      format: int32
      description: Application-specific error code
    details:
      type: array
      items:
        type: string
      nullable: true
```

**Note:** `http_code` is NOT included in the JSON body - it's in the HTTP response status header.

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Validation Placement

**Request format validation belongs in the API layer. Business logic validation belongs in the service layer.**

| Validation Type | Layer | Examples |
|----------------|-------|----------|
| Request format | API | Page >= 1, limit 1-100, valid UUID format |
| Business logic | Service | User has permission, quota not exceeded, resource exists |

**Why request format validation stays in API layer:**

1. **Fail fast** - Reject malformed requests before calling service
2. **API concern** - Validating HTTP parameters is part of "translating requests to service calls"
3. **Not business logic** - Format rules (page >= 1) are API contract, not domain rules
4. **Separation** - Service layer trusts it receives valid inputs

```go
// CORRECT: Request format validation in API handler
func (h *ServerHandler) ListItems(ctx echo.Context, params stub.ListItemsParams) error {
    // ... JWT, timeout, claims setup ...

    // Request format validation - API layer responsibility
    if err := utils.ValidatePaginationParamsWithDefaults(params.Page, params.Limit); err != nil {
        return err
    }

    // Service receives validated inputs
    result, err := h.service.ListItems(ctxTimeout, claims.OrganizationID, params.Page, params.Limit)
    // ...
}

// CORRECT: Business logic validation in service layer
func (s *service) ListItems(ctx context.Context, orgID uuid.UUID, page, limit *int) (*stub.ItemList, error) {
    // Business validation - service layer responsibility
    if !s.hasPermission(ctx, orgID, "items:read") {
        return nil, common.NewCustomError("Permission denied")...
    }
    // ...
}
```

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Reusing pkg/common Utilities

**IMPORTANT: Always check `pkg/common` for existing utilities before creating new ones.**

The `pkg/common` module provides shared utilities that should be reused across all services. Before implementing any helper function, check if it already exists in:

- `pkg/common/utils/` - General utilities (sorting, OpenAPI type conversions, etc.)
- `pkg/common/jwt/` - JWT token handling
- `pkg/common/translations/` - Locale and translation helpers
- `pkg/common/errorcodes/` - Standardized error codes
- `pkg/common/constants/` - Shared constants

**Common utilities available:**

| Utility | Location | Purpose |
|---------|----------|---------|
| `utils.IntPtr()` | `pkg/common/utils/openapi.go` | Convert OpenAPI int pointer to `*int` |
| `utils.StringPtr()` | `pkg/common/utils/openapi.go` | Convert OpenAPI string pointer to `*string` |
| `utils.ValidatePaginationParamsWithDefaults()` | `pkg/common/utils/pagination.go` | Validate page/limit params with default config |
| `utils.ValidatePaginationParams()` | `pkg/common/utils/pagination.go` | Validate page/limit params with custom config |
| `utils.OpenAPIUUIDSlicePtrToUUIDSlice()` | `pkg/common/utils/openapi.go` | Convert `*[]types.UUID` to `[]uuid.UUID` |
| `utils.OpenAPIUUIDSliceToUUIDSlice()` | `pkg/common/utils/openapi.go` | Convert `[]types.UUID` to `[]uuid.UUID` |
| `utils.ParseCommaSeparatedUUIDs()` | `pkg/common/utils/openapi.go` | Parse comma-separated UUID string |
| `utils.ValidateAndNormalizeSortParams()` | `pkg/common/utils/sorting.go` | Validate and normalize sort parameters |
| `utils.StringToOpenAPIEmailPtr()` | `pkg/common/utils/openapi.go` | Convert `string` to `*openapi_types.Email` |
| `utils.UUIDToOpenAPIUUIDPtr()` | `pkg/common/utils/openapi.go` | Convert `uuid.UUID` to `*openapi_types.UUID` |
| `common.BindRequestBody()` | `pkg/common/request.go` | Bind and validate request body |
| `jwt.GetJWTFromEchoContext()` | `pkg/common/jwt/` | Extract JWT from Echo context |
| `translations.GetLocaleFromEchoContext()` | `pkg/common/translations/` | Extract locale from Echo context |

**Example: Using OpenAPI type conversion utilities**

```go
// CORRECT: Use pkg/common utilities
import "github.com/industrix-id/backend/pkg/common/utils"

func (h *ServerHandler) ListItems(ctx echo.Context, params stub.ListItemsParams) error {
    // Use utility for pointer-to-slice conversion
    userIDs := utils.OpenAPIUUIDSlicePtrToUUIDSlice(params.UserIds)
    deviceIDs := utils.OpenAPIUUIDSlicePtrToUUIDSlice(params.DeviceIds)
    // ...
}

// WRONG: Creating local helper when pkg/common has one
func parseUUIDs(ids *[]openapi_types.UUID) []uuid.UUID {
    // Don't create this - use utils.OpenAPIUUIDSlicePtrToUUIDSlice instead
}
```

**When to add new utilities to pkg/common:**

1. The utility is needed by multiple services
2. It handles a common pattern (type conversion, validation, etc.)
3. It doesn't contain service-specific business logic

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Request Flow

Here's how a request flows through your service:

```
1. HTTP Request arrives
   ↓
2. Echo router (from stub.RegisterHandlers)
   ↓
3. Handler method (api/handlers.go)
   - Extract JWT token
   - Decode token to get claims (via service.DecodeAuthenticationToken)
   - Parse request body/params
   ↓
4. Service layer (service/service.go)
   - Receives claims directly (NOT auth token)
   - Business logic
   - Validation
   - Call repositories
   ↓
5. Repository (pkg/database/repository)
   - Database queries
   - Cache operations
   ↓
6. Response flows back up
   Handler → JSON response → Client
```

**Example: GET /users**

```
Client → GET /users
  ↓
Echo routes to GetUsers handler
  ↓
Handler extracts Authorization header (jwt.GetJWTFromEchoContext)
  ↓
Handler decodes JWT via service.DecodeAuthenticationToken()
  ↓
Handler calls service.GetUsers(ctx, claims, orgID, ...)
  ↓
Service uses claims.UserID for user-specific logic (e.g., timezone)
  ↓
Service calls userRepository.FindByOrganizationID()
  ↓
Repository queries database
  ↓
Results flow back: DB → Repo → Service → Handler
  ↓
Service returns stub types directly (no conversion needed)
  ↓
Handler returns ctx.JSON(200, response)
  ↓
Client receives JSON response
```

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## JWT Middleware Configuration

**All services must configure JWT middleware in `main.go` to extract JWT tokens into the Echo context.**

The JWT middleware extracts the Authorization header and stores the token in the Echo context, enabling handlers to use `jwt.GetJWTFromEchoContext()` to retrieve the token.

**Required setup in main.go:**

```go
import (
    "github.com/industrix-id/backend/pkg/common/initialization"
)

func main() {
    // ... config, logger setup ...

    app := initialization.NewEcho(&cfg.DashboardUIHostname)

    // Apply JWT middleware to protected routes
    // Skip paths that don't use Authorization header
    app.Use(initialization.JWTAuthMiddleware([]string{
        "/ping",                      // Health check - no auth needed
        "/endpoint-permissions",      // Uses auth_token query param
        "/device-tokens/exchange",    // Uses refresh_token in body
        "/device-tokens/authenticate", // Uses api_key in body
    }))

    // ... service instantiation, handler registration ...
}
```

**Skip paths are required for endpoints that:**

| Scenario | Example | Reason |
|----------|---------|--------|
| Health checks | `/ping` | Must be accessible without authentication for load balancers |
| Query param auth | `/endpoint-permissions?auth_token=...` | Token passed as query param, not Authorization header |
| Body-based auth | `/device-tokens/authenticate` | Uses API key in request body, not JWT |
| Token exchange | `/device-tokens/exchange` | Uses refresh token in body to get new access token |

**How it works:**

1. **Middleware receives request** - Checks if path is in skip list
2. **If skipped** - Request passes through without extracting token
3. **If not skipped** - Extracts `Authorization: Bearer <token>` header and stores token in Echo context
4. **Handler calls `jwt.GetJWTFromEchoContext()`** - Retrieves token from context

```
Request → JWTAuthMiddleware → Handler
                ↓
         Extract "Bearer <token>"
                ↓
         Store in Echo context
                ↓
         jwt.GetJWTFromEchoContext() retrieves it
```

**Important notes:**

- Without this middleware, `jwt.GetJWTFromEchoContext()` will return error "JWT token not found in context"
- The middleware does NOT validate the token - it only extracts and stores it
- Token validation happens in the service layer via `DecodeAuthenticationToken()`
- Kong API gateway handles initial auth validation before requests reach the service

**Example error without middleware:**

```json
{
  "message": "JWT token not found in context",
  "code": 3008
}
```

This error means the JWT middleware is either:
1. Not applied to the service
2. The endpoint is incorrectly added to the skip list

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Related Guides

- [How to Create a Service](./how-to-create-a-service.md)
- [How to Write Service Layer](./how-to-write-service-layer.md)
- [How to Structure OpenAPI](./how-to-structure-openapi.md)
- [How to Understand Architecture](./how-to-understand-architecture.md)
