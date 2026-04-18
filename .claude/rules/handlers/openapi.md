---
paths:
  - "services/*/openapi/**"
  - "services/*/stub/**"
  - "services/*/api/**"
  - "services/*/generate.go"
---

# OpenAPI Conventions

## File Structure

```
services/{service}/
├── openapi/
│   └── openapi.yaml          # OpenAPI spec (hand-written)
├── stub/
│   └── openapi.gen.go        # Generated types & interfaces (never edit)
├── api/
│   ├── handlers.go           # HTTP handlers implementing stub.ServerInterface
│   └── converters.go         # stub ↔ domain type converters
└── generate.go               # //go:generate directive for oapi-codegen
```

## Spec Header

```yaml
openapi: 3.0.3
info:
  title: {Service} Service
  description: API for {service} operations
  version: 1.0.0

servers:
  - url: "{{backendUrl}}/{service-path}"
```

Note: `{{backendUrl}}/{service-path}` — the actual prefix is defined in `infra/kong/kong.template.yml`, not in the OpenAPI spec. See `.claude/rules/businesslogic/kong.md`.

## Naming Conventions

| Element | Convention | Example |
|---------|-----------|---------|
| URL paths | kebab-case | `/site-categories`, `/fuel-tank-monitoring-devices` |
| Path parameters | camelCase | `{siteID}`, `{organizationID}`, `{subscriptionID}` |
| Query parameters | snake_case | `sort_by`, `sort_order`, `from_date`, `site_category_ids` |
| JSON fields | snake_case | `site_category_id`, `is_active`, `created_at`, `translated_message` |
| Tags | PascalCase or quoted | `Users`, `Health`, `"Admin: Organizations"` |
| Schema names | PascalCase | `SiteCreate`, `PaginatedResponse`, `Error` |

## Standard Components

Every service should reuse these standard components via `$ref`.

### Parameters

```yaml
components:
  parameters:
    Authorization:
      name: Authorization
      in: header
      required: true
      schema:
        type: string
      example: "Bearer {{authToken}}"

    AcceptLanguage:
      name: Accept-Language
      in: header
      required: false
      schema:
        type: string
        default: en
      example: "en"

    Page:
      name: page
      in: query
      required: false
      schema:
        type: integer
        minimum: 1
        default: 1

    Limit:
      name: limit
      in: query
      required: false
      schema:
        type: integer
        minimum: 1
        maximum: 100
        default: 10

    Search:
      name: search
      in: query
      required: false
      schema:
        type: string

    SortBy:
      name: sort_by
      in: query
      required: false
      schema:
        type: string

    SortOrder:
      name: sort_order
      in: query
      required: false
      schema:
        type: string
        enum: [asc, desc]
```

### Responses

```yaml
components:
  responses:
    BadRequest:
      description: Bad request - Invalid parameters
      content:
        application/json:
          schema:
            $ref: "#/components/schemas/Error"

    Unauthorized:
      description: Unauthorized - Invalid or missing auth token
      content:
        application/json:
          schema:
            $ref: "#/components/schemas/Error"

    Forbidden:
      description: Access forbidden
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

### Schemas

```yaml
components:
  schemas:
    UUID:
      type: string
      format: uuid
      example: "550e8400-e29b-41d4-a716-446655440000"

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

    Pong:
      type: object
      properties:
        message:
          type: string
          example: "pong"

    PaginatedResponse:
      type: object
      properties:
        page:
          type: integer
        limit:
          type: integer
        total:
          type: integer

  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
```

## Response Patterns

### Paginated List

Use `allOf` to compose pagination with data:

```yaml
SiteList:
  allOf:
    - $ref: "#/components/schemas/PaginatedResponse"
    - type: object
      properties:
        data:
          type: array
          items:
            $ref: "#/components/schemas/Site"
```

### Single Resource

```yaml
responses:
  "200":
    content:
      application/json:
        schema:
          $ref: "#/components/schemas/Site"
```

### No Content

```yaml
responses:
  "204":
    description: Successfully deleted
```

## HTTP Methods

| Method | Use | Status Code |
|--------|-----|-------------|
| `GET /resource` | List | 200 |
| `POST /resource` | Create | 201 |
| `GET /resource/{id}` | Get detail | 200 |
| `PUT /resource/{id}` | Update | 200 |
| `DELETE /resource/{id}` | Delete | 204 |
| `POST /resource/{id}/action` | Custom action | 200 or 201 |

## Organization Scoping

**Implicit** (most common) — organization determined from JWT claims:
```yaml
/sites:
  get:
    description: List sites for the current user's organization
```

**Explicit** — admin endpoints with org in path:
```yaml
/admin/organizations/{organizationID}/subscriptions:
  get:
    parameters:
      - $ref: "#/components/parameters/OrganizationID"
```

## Tags

```yaml
tags:
  - name: Health
    description: Health check endpoints
  - name: Sites
    description: Site management
  - name: "Admin: Organizations"
    description: Organization management (admin only)
```

Admin-only operations use `"Admin: "` prefix. Every endpoint must have at least one tag.

## Nullable Fields

Use `nullable: true` for optional fields. Generated Go types use pointers:

```yaml
# OpenAPI
latitude:
  type: number
  format: double
  nullable: true
```

```go
// Generated Go
Latitude *float32 `json:"latitude,omitempty"`
```

## Descriptions

Include behavioral notes in endpoint descriptions:

```yaml
description: |
  Retrieve a paginated list of bills for the current user's organization.

  **Notes:**
  - Organization is determined from the authenticated user's JWT claims
  - Draft invoices are excluded
  - Overdue status is calculated dynamically based on due_date
```

Document possible error codes when relevant:

```yaml
description: >
  Possible errors:
  - PasswordNoUpperCase: 22
  - PasswordNoLowerCase: 23
```

## Code Generation

Generator: `oapi-codegen/v2`. Config is in `generate.go`:

```go
//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen -package stub -o stub/openapi.gen.go openapi/openapi.yaml
```

**Never edit `stub/openapi.gen.go`** — regenerate with:
```bash
make openapi-gen
```

## Converter Pattern

Converters live in `services/{service}/api/converters.go`. Two directions:

**Request (stub → domain):**
```go
func FromStubSiteCreate(req stub.SiteCreate) types.CreateSiteRequest {
    result := types.CreateSiteRequest{
        Name: req.Name,
    }
    if req.SiteCategoryId != nil {
        id := uuid.UUID(*req.SiteCategoryId)
        result.SiteCategoryID = &id
    }
    return result
}
```

**Response (domain → stub):**
```go
func ToStubSite(site *types.Site) *stub.Site {
    if site == nil {
        return nil
    }
    id := stub.UUID(site.ID)
    return &stub.Site{
        Id:   &id,
        Name: &site.Name,
    }
}
```

Naming: `FromStub*` for request converters, `ToStub*` for response converters. Unexported helpers use camelCase (`toStubUUIDPtr`).

### Float Precision at API Boundary

OpenAPI `double` generates `*float32` in Go stubs. Convert at the boundary:

```go
// stub float32 → domain float64
if r.Latitude != nil {
    val := float64(*r.Latitude)
    result.Latitude = &val
}

// domain float64 → stub float32
if r.Latitude != nil {
    val := float32(*r.Latitude)
    result.Latitude = &val
}
```

## Health Check Endpoint

Every service must have a `/ping` endpoint:

```yaml
/ping:
  get:
    tags: [Health]
    operationId: Ping
    responses:
      "200":
        description: Service is healthy
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/Pong"
```
