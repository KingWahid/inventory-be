<!-- TODO this ought to be in repo layer instead and instead of camelcase sortBy, it should have been sort_by - sort_order instead. -->

# How to Implement Sorting with Service-Layer Validation

When your service has list endpoints with sorting, use **service-layer whitelist validation** as the Single Source of Truth (SSOT) for allowed sort values. This provides explicit control and security against malformed requests.

## Table of Contents

- [How to Implement Sorting with Service-Layer Validation](#how-to-implement-sorting-with-service-layer-validation)
  - [Table of Contents](#table-of-contents)
  - [Step 1: Define Sort Parameters in OpenAPI](#step-1-define-sort-parameters-in-openapi)
  - [Step 2: Generate Stub Code](#step-2-generate-stub-code)
  - [Step 3: Define Whitelist Config in Service Layer](#step-3-define-whitelist-config-in-service-layer)
  - [Step 4: Validate and Use in Service Function](#step-4-validate-and-use-in-service-function)
  - [Why This Pattern](#why-this-pattern)
  - [The ValidateAndNormalizeSortParams Function](#the-validateandnormalizesortparams-function)
  - [Frontend Integration](#frontend-integration)
  - [Related Guides](#related-guides)

---

## Step 1: Define Sort Parameters in OpenAPI

```yaml
# In openapi/openapi.yaml
paths:
  /items:
    get:
      parameters:
        - name: sortBy
          in: query
          description: Field to sort by (e.g., name, created_at). Validated server-side.
          required: false
          schema:
            type: string
            example: "created_at"
        - name: sortOrder
          in: query
          description: Sort order direction (asc or desc). Defaults to desc.
          required: false
          schema:
            type: string
            example: "desc"
            # Or you may use reusable generic schema instead of creating new one check other implementation
```

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Step 2: Generate Stub Code

```bash
go generate ./...
```

This generates simple string pointers in `stub/openapi.gen.go`:

```go
type GetItemsParams struct {
    SortBy    *string `form:"sortBy,omitempty" json:"sortBy,omitempty"`
    SortOrder *string `form:"sortOrder,omitempty" json:"sortOrder,omitempty"`
    // ...
}
```

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Step 3: Define Whitelist Config in Service Layer

```go
// In service/items.go
import "github.com/industrix-id/backend/pkg/common/utils"

// itemSortConfig is the SSOT for item sorting.
// This whitelist defines allowed fields and defaults.
var itemSortConfig = utils.SortValidationConfig{
    AllowedFields: map[string]bool{
        "name":       true,
        "created_at": true,
        "is_active":  true,
        "price":      true,
    },
    DefaultField: "created_at",
    DefaultOrder: "DESC",
}
```

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Step 4: Validate and Use in Service Function

```go
func (s *service) ListItems(
    ctx context.Context,
    page, limit *int,
    sortBy, sortOrder *string,  // Simple string pointers from API
) (*ListItemsResponse, error) {
    // Validate and normalize using shared utility with service config (SSOT)
    // Invalid values silently fall back to defaults for security
    sortParams := utils.ValidateAndNormalizeSortParams(sortBy, sortOrder, itemSortConfig)

    // Pass validated params to repository
    items, pagination, err := s.itemRepository.ListItems(
        ctx, page, limit,
        sortParams.Field,  // Guaranteed to be in whitelist
        sortParams.Order,  // Guaranteed to be "ASC" or "DESC"
    )
    // ...
}
```

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Why This Pattern

| Benefit | Explanation |
|---------|-------------|
| **Explicit Security** | Whitelist prevents SQL injection and invalid column access |
| **Graceful Handling** | Invalid input silently falls back to defaults (no 500 errors) |
| **Single Source of Truth** | Service layer defines allowed values, visible in code |
| **Flexible Defaults** | Each resource can have different default field/order |
| **No Code Generation Issues** | OpenAPI enum validation isn't enforced by oapi-codegen |

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## The ValidateAndNormalizeSortParams Function

Located in `pkg/common/utils/sorting.go`, this shared utility:

1. Validates `sortBy` against the `AllowedFields` whitelist
2. Normalizes `sortOrder` to "ASC" or "DESC"
3. Falls back to configured defaults for invalid/missing values
4. Returns a `SortParams` struct with `Field` and `Order`

```go
// Example with custom defaults
config := utils.SortValidationConfig{
    AllowedFields: map[string]bool{"name": true, "price": true},
    DefaultField:  "name",   // Custom default field
    DefaultOrder:  "ASC",    // Custom default order
}
params := utils.ValidateAndNormalizeSortParams(sortBy, sortOrder, config)
```

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Frontend Integration

Send `sortBy` and `sortOrder` as separate query parameters:

```typescript
// In data provider
const params = {
  sortBy: "name",      // Field name
  sortOrder: "asc",    // "asc" or "desc"
};

// GET /api/items?sortBy=name&sortOrder=asc
```

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Related Guides

- [How to Write Service Layer](./how-to-write-service-layer.md)
- [How to Write Handlers](./how-to-write-handlers.md)
- [How to Structure OpenAPI](./how-to-structure-openapi.md)
