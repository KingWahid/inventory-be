# OpenAPI Structuring Guide

This document outlines best practices for structuring OpenAPI specifications to be cleaner, DRY (Don't Repeat Yourself), and easier to maintain.

## Table of Contents

1. [Request Mapping Quick Reference](#request-mapping-quick-reference)
2. [File Structure](#file-structure)
3. [Schemas](#schemas)
4. [Parameters](#parameters)
5. [Request Bodies](#request-bodies)
6. [Responses](#responses)
7. [Pagination Patterns](#pagination-patterns)
8. [Naming Conventions](#naming-conventions)
9. [What to Keep vs Remove](#what-to-keep-vs-remove)

---

## Request Mapping Quick Reference

Where each part of an HTTP request belongs in OpenAPI:

```
                                          Query Parameters
                                          ┌───────┴───────┐
POST https://example.com/api/users/abc123?search=foo&limit=10
│    └────────┬─────────┘   └──┬──┘└──┬──┘└───┬───┘   └─┬──┘
│             │                │      │       │         │
│             │                │      │       │         └─ parameters:
│             │                │      │       │              name: limit
│             │                │      │       │              in: query
│             │                │      │       │
│             │                │      │       └─────────── parameters:
│             │                │      │                      name: search
│             │                │      │                      in: query
│             │                │      │
│             │                │      └─────────────────── parameters:
│             │                │                             name: userID
│             │                │                             in: path
│             │                │
│             │                └────────────────────────── paths:
│             │                                              /api/users/{userID}:
│             │
│             └────────────────────────────────────────── servers:
│                                                           - url: https://example.com
│
└──────────────────────────────────────────────────────── paths:
                                                            /api/users/{userID}:
                                                              post:

┌─────────────────────────────────────────┐
│ Headers                                 │
│   Authorization: Bearer eyJhbG...       │──── Handled by middleware (not in OpenAPI spec)
│   Content-Type: application/json        │     Define in security: only if needed for codegen
├─────────────────────────────────────────┤
│ Body                                    │
│   {                                     │
│     "data": {                           │──── requestBody:
│       "name": "John",                   │       content:
│       "email": "john@example.com"       │         application/json:
│     }                                   │           schema:
│   }                                     │             $ref: "#/components/schemas/UserCreate"
└─────────────────────────────────────────┘

┌─────────────────────────────────────────┐
│ Response                                │
│   {                                     │
│     "id": "abc123",                     │──── responses:
│     "name": "John",                     │       "201":
│     "email": "john@example.com"         │         content:
│   }                                     │           application/json:
└─────────────────────────────────────────┘             schema:
                                                          $ref: "#/components/schemas/User"
```

---

## File Structure

Organize your OpenAPI spec in this order:

```yaml
openapi: "3.0.0"
info:
  title: Service Name
  version: "1.0.0"

paths:
  # Group by domain/feature with comments
  # ---------------------------------------------------------------------------
  # Feature Name
  # ---------------------------------------------------------------------------
  /endpoint:
    get:
      ...

components:
  parameters:
    # Reusable path/query parameters

  responses:
    # Reusable error responses (high reuse value)

  schemas:
    # All data models, grouped by domain
```

### Use Section Comments

Add clear separators between logical groups:

```yaml
paths:
  # ---------------------------------------------------------------------------
  # Roles
  # ---------------------------------------------------------------------------
  /roles:
    ...

  # ---------------------------------------------------------------------------
  # Users
  # ---------------------------------------------------------------------------
  /users:
    ...

components:
  schemas:
    # -------------------------------------------------------------------------
    # Common/Shared
    # -------------------------------------------------------------------------
    Error:
      ...

    # -------------------------------------------------------------------------
    # Roles
    # -------------------------------------------------------------------------
    Role:
      ...
```

---

## Schemas

### DO: Update Frontend

Update the existing typed data/object in the frontend everytime changing the schema from the backend via the openapi.

### DO: Create Reusable Base Schemas

```yaml
# Good - reusable pagination base
PaginatedResponse:
  type: object
  properties:
    page:
      type: integer
    limit:
      type: integer
    total:
      type: integer

# Good - reusable UUID type
UUID:
  type: string
  format: uuid
  example: "123e4567-e89b-12d3-a456-426614174000"
```

### DO: Use `allOf` for Composition

```yaml
# Good - compose list responses from base
RoleList:
  allOf:
    - $ref: "#/components/schemas/PaginatedResponse"
    - type: object
      properties:
        data:
          type: array
          items:
            $ref: "#/components/schemas/Role"
```

### DO: Extend Base Schemas When Needed

If some endpoints need additional fields, extend the base schema rather than duplicating:

```yaml
# Base pagination schema
PaginatedResponse:
  type: object
  properties:
    page:
      type: integer
    limit:
      type: integer
    total:
      type: integer

# Extended schema for endpoints that need total_pages
PaginatedResponseWithPages:
  allOf:
    - $ref: "#/components/schemas/PaginatedResponse"
    - type: object
      properties:
        total_pages:
          type: integer

# Usage - most endpoints use base
RoleList:
  allOf:
    - $ref: "#/components/schemas/PaginatedResponse"
    - type: object
      properties:
        data:
          type: array
          items:
            $ref: "#/components/schemas/Role"

# Usage - specific endpoint needs total_pages
DeviceRefreshTokenListResponse:
  type: object
  properties:
    refresh_tokens:
      type: array
      items:
        $ref: "#/components/schemas/DeviceRefreshTokenInfo"
    pagination:
      $ref: "#/components/schemas/PaginatedResponseWithPages"
```

### DON'T: Duplicate Similar Schemas

```yaml
# Bad - duplicated pagination fields
RoleList:
  type: object
  properties:
    data:
      type: array
      items:
        $ref: "#/components/schemas/Role"
    page:
      type: integer
    limit:
      type: integer
    total:
      type: integer

UserList:
  type: object
  properties:
    data:
      type: array
      items:
        $ref: "#/components/schemas/User"
    page:
      type: integer      # Duplicated!
    limit:
      type: integer      # Duplicated!
    total:
      type: integer      # Duplicated!
```

### DON'T: Create Circular References

```yaml
# Bad - circular reference
UUID:
  $ref: "#/components/schemas/UUID"  # References itself!

# Good - proper definition
UUID:
  type: string
  format: uuid
```

### DO: Consolidate Identical Types

If multiple fields represent the same concept (e.g., various ID fields), use a single schema:

```yaml
# Good - single UUID schema referenced everywhere
schemas:
  UUID:
    type: string
    format: uuid

  Role:
    properties:
      id:
        $ref: "#/components/schemas/UUID"

  User:
    properties:
      id:
        $ref: "#/components/schemas/UUID"
      role_id:
        $ref: "#/components/schemas/UUID"
```

---

## Parameters

### DO: Create Reusable Path Parameters

```yaml
components:
  parameters:
    UserID:
      name: userID
      in: path
      required: true
      schema:
        $ref: "#/components/schemas/UUID"

    RoleID:
      name: roleID
      in: path
      required: true
      schema:
        $ref: "#/components/schemas/UUID"
```

### DO: Create Reusable Query Parameters

```yaml
components:
  parameters:
    Page:
      name: page
      in: query
      schema:
        type: integer
        default: 1

    Limit:
      name: limit
      in: query
      schema:
        type: integer
        default: 10

    Search:
      name: search
      in: query
      schema:
        type: string
```

### DON'T: Include Parameters Handled by Middleware

If your middleware automatically handles certain headers (e.g., `Accept-Language`, `Authorization`), you may not need them in the OpenAPI spec for validation purposes. However, keep them if:
- You use code generation that needs them
- You want them documented for API consumers

```yaml
# Consider removing if middleware handles it
parameters:
  AcceptLanguage:
    name: Accept-Language
    in: header
    schema:
      type: string
```

### Exception: Service-to-Service Validation Parameters

Some endpoints are designed for service-to-service communication where a token is passed explicitly for validation (not for authenticating the caller). Keep these as query parameters:

```yaml
# Good - explicit token for permission validation (service-to-service)
/endpoint-permissions:
  get:
    summary: Check user permission for an endpoint
    description: Validates if the user (identified by `authToken`) has access to the specified endpoint.
    parameters:
      - $ref: "#/components/parameters/AuthToken"  # Query param, not header
      - $ref: "#/components/parameters/EndpointPath"
      - $ref: "#/components/parameters/EndpointAction"

# The AuthToken parameter definition
parameters:
  AuthToken:
    name: authToken
    in: query
    description: Authentication token (JWT) for permission validation
    required: true
    schema:
      type: string
```

This is different from the `Authorization` header:
- `Authorization` header: Authenticates the **caller** (handled by middleware)
- `authToken` query param: The token being **validated** (passed by the caller for permission checks)

---

## Request Bodies

### DON'T: Use `requestBodies` Section

The `requestBodies` component adds indirection without much benefit since request bodies are typically unique per endpoint.

```yaml
# Bad - unnecessary indirection
components:
  requestBodies:
    UserCreate:
      required: true
      content:
        application/json:
          schema:
            $ref: "#/components/schemas/UserCreate"

paths:
  /users:
    post:
      requestBody:
        $ref: "#/components/requestBodies/UserCreate"
```

### DO: Inline Request Bodies in Paths

```yaml
# Good - direct and clear
paths:
  /users:
    post:
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/UserCreate"
```

---

## Responses

### DO: Keep Reusable Error Responses

Error responses have high reuse value (often 50+ references). Keep them in `components/responses`:

```yaml
components:
  responses:
    BadRequest:
      description: Bad request due to validation errors
      content:
        application/json:
          schema:
            $ref: "#/components/schemas/Error"

    Unauthorized:
      description: Unauthorized - Invalid or missing authentication token
      content:
        application/json:
          schema:
            $ref: "#/components/schemas/Error"

    Forbidden:
      description: Forbidden - Insufficient permissions
      content:
        application/json:
          schema:
            $ref: "#/components/schemas/Error"

    NotFound:
      description: Resource not found
      content:
        application/json:
          schema:
            $ref: "#/components/schemas/Error"

    InternalServerError:
      description: Internal server error
      content:
        application/json:
          schema:
            $ref: "#/components/schemas/Error"
```

### DON'T: Create Response Wrappers for Single-Use Cases

```yaml
# Bad - unnecessary wrapper for a single schema
components:
  responses:
    Role:
      description: Role details
      content:
        application/json:
          schema:
            $ref: "#/components/schemas/Role"

# Good - inline directly in path
paths:
  /roles/{roleID}:
    get:
      responses:
        "200":
          description: Role details
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Role"
```

---

## Pagination Patterns

### Choose ONE Pattern and Be Consistent

#### Option A: Flat Structure (Recommended)

```yaml
# Response: {data: [], page: 1, limit: 10, total: 100}
RoleList:
  allOf:
    - $ref: "#/components/schemas/PaginatedResponse"
    - type: object
      properties:
        data:
          type: array
          items:
            $ref: "#/components/schemas/Role"
```

#### Option B: Nested Meta Structure

```yaml
# Response: {data: [], meta: {page: 1, limit: 10, total: 100}}
RoleList:
  type: object
  properties:
    data:
      type: array
      items:
        $ref: "#/components/schemas/Role"
    meta:
      $ref: "#/components/schemas/PaginatedResponse"
```

### Why Flat is Recommended

1. **Simpler access**: `response.total` vs `response.meta.total`
2. **Less nesting**: Easier to work with in frontend code
3. **Framework compatibility**: Many frameworks (e.g., Refine) expect flat `{data, total}`

### DON'T: Mix Patterns

```yaml
# Bad - inconsistent pagination patterns
RoleList:      # Uses flat: {data, page, limit, total}
UserList:      # Uses flat: {data, page, limit, total}
AccessCardList: # Uses nested: {data, meta: {page, limit, total}}  # Inconsistent!
```

---

## Naming Conventions

### Philosophy: Schemas Describe DATA, Not Operations

Schemas define the **structure of data**, not the HTTP operation that uses them. The HTTP method (GET, POST, PUT) already tells you the operation context.

### Schema Naming Rules

| Type | Convention | Example | Notes |
|------|------------|---------|-------|
| **Entity** | PascalCase noun | `User`, `Role`, `Organization` | The core data model |
| **List** | Entity + `List` | `UserList`, `RoleList` | Paginated collection |
| **Create payload** | Entity + `Create` | `UserCreate`, `RoleCreate` | Fields needed to create |
| **Update payload** | Entity + `Update` | `UserUpdate`, `RoleUpdate` | Fields that can be updated |
| **Action payload** | Action name | `SetActive`, `RevokeAccessCard` | Describes the action |
| **Info/Details** | Entity + `Info` | `DeviceRefreshTokenInfo` | Subset of entity fields |
| **Result** | Context + `Response` | `DeviceAuthTokenResponse` | Only when response differs significantly from entity |

### When to Use `Response` Suffix

**DO use `Response`** when the schema is specifically an API response structure that differs from the entity:

```yaml
# Good - this is specifically a token response, not a token entity
DeviceAuthTokenResponse:
  properties:
    auth_token: string      # The actual JWT
    device_id: UUID
    expires_at: datetime

# Good - response includes metadata not part of the entity
DeviceRefreshTokenListResponse:
  properties:
    refresh_tokens: array
    pagination: object      # Response-specific pagination wrapper
```

**DON'T use `Response`** when it's just wrapping an entity:

```yaml
# Bad - just wrapping the Role entity
RoleResponse:
  $ref: "#/components/schemas/Role"

# Good - return the entity directly
responses:
  "200":
    content:
      application/json:
        schema:
          $ref: "#/components/schemas/Role"
```

### DON'T: Use Redundant Prefixes/Suffixes

```yaml
# Bad - redundant naming
RoleCreateRequest      # "Request" adds nothing - it's in requestBody
RoleListResponse       # "Response" adds nothing - it's in responses
GetActionAuditLogResponse  # "Get" prefix is HTTP verb, not data
UserUpdateRequest      # We know it's a request from context

# Good - clean, semantic naming
RoleCreate             # "Create" describes what data is needed
RoleList               # "List" describes collection structure
ActionAuditLog         # The entity name
UserUpdate             # "Update" describes the payload purpose
```

### DON'T: Use `Request` Suffix

The `Request` suffix is almost always redundant:

```yaml
# Bad
SetActiveRequest
AccessCardCreateRequest

# Good
SetActive              # Action name is sufficient
AccessCardCreate       # Entity + operation is clear
```

### Exception: Pagination Base Schemas

Keep `Response` for base pagination schemas since they specifically describe response structure:

```yaml
# Acceptable - describes response pagination pattern
PaginatedResponse:
  properties:
    page: integer
    limit: integer
    total: integer

PaginatedResponseWithPages:
  allOf:
    - $ref: "#/components/schemas/PaginatedResponse"
    - properties:
        total_pages: integer
```

### Parameter Names

Use camelCase for parameters:

```yaml
parameters:
  userID:           # Not user_id or UserId
  roleID:
  sortOrder:
  includeInactive:
  pageSize:         # Not page_size
```

### Summary: Clean Naming Decision Tree

```
Is it a core entity?
  → Use entity name: User, Role, Permission

Is it a list/collection?
  → Entity + List: UserList, RoleList

Is it for creating something?
  → Entity + Create: UserCreate, RoleCreate

Is it for updating something?
  → Entity + Update: UserUpdate, RoleUpdate

Is it an action payload?
  → Action name: SetActive, RevokeAccessCard

Is it a response with special structure (pagination, metadata)?
  → Context + Response: DeviceRefreshTokenListResponse

Otherwise?
  → Just describe what the data IS
```

---

## What to Keep vs Remove

### KEEP in `components/responses`

| Item | Reason |
|------|--------|
| Error responses (400, 401, 403, 404, 500) | High reuse (50+ references) |

### REMOVE / INLINE

| Item | Reason |
|------|--------|
| `requestBodies` section | Low reuse, adds indirection |
| Single-use response wrappers | No benefit over direct schema ref |
| Duplicate pagination schemas | Use single `PaginatedResponse` |
| Parameters handled by middleware | Unless needed for codegen |

### CONSOLIDATE

| Before | After |
|--------|-------|
| `PaginatedResponse` + `PaginationInfo` | Single `PaginatedResponse` |
| Multiple UUID-like types | Single `UUID` schema |
| Flat + nested pagination | Single consistent pattern |

---

## Checklist for Restructuring

- [ ] Remove `tags` section if not using Swagger UI
- [ ] Remove `tags: [X]` from operations if not using Swagger UI
- [ ] Consolidate duplicate schemas (pagination, UUIDs, etc.)
- [ ] Use `allOf` for schema composition
- [ ] Remove `requestBodies` section, inline in paths
- [ ] Keep only error responses in `components/responses`
- [ ] Ensure consistent pagination pattern (flat recommended)
- [ ] Use clean naming without redundant suffixes
- [ ] Add section comments for organization
- [ ] Remove parameters handled by middleware (if not needed for codegen)
- [ ] Verify no circular references in schemas
- [ ] Regenerate code and verify build passes
- [ ] Update frontend to match any response structure changes

---

## Example: Before and After

### Before (Verbose)

```yaml
components:
  requestBodies:
    UserCreate:
      required: true
      content:
        application/json:
          schema:
            $ref: "#/components/schemas/UserCreateRequest"

  responses:
    User:
      description: User details
      content:
        application/json:
          schema:
            $ref: "#/components/schemas/User"

  schemas:
    UserCreateRequest:
      type: object
      properties:
        name:
          type: string

    UserListResponse:
      type: object
      properties:
        data:
          type: array
          items:
            $ref: "#/components/schemas/User"
        page:
          type: integer
        limit:
          type: integer
        total:
          type: integer

paths:
  /users:
    post:
      tags: [Users]
      requestBody:
        $ref: "#/components/requestBodies/UserCreate"
      responses:
        "201":
          $ref: "#/components/responses/User"
```

### After (Clean)

```yaml
components:
  schemas:
    PaginatedResponse:
      type: object
      properties:
        page:
          type: integer
        limit:
          type: integer
        total:
          type: integer

    UserCreate:
      type: object
      properties:
        name:
          type: string

    UserList:
      allOf:
        - $ref: "#/components/schemas/PaginatedResponse"
        - type: object
          properties:
            data:
              type: array
              items:
                $ref: "#/components/schemas/User"

paths:
  /users:
    post:
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/UserCreate"
      responses:
        "201":
          description: User created
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/User"
```

---

## Related Files

When restructuring OpenAPI, you may need to update:

1. **Generated stubs** - Run code generation after changes
2. **Backend handlers** - Update type references if schema names changed
3. **Frontend data providers** - Update response parsing if structure changed
4. **Tests** - Update expected response formats
