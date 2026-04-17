# How to Use Utilities

This guide covers shared utility functions from `pkg/common/utils` and `pkg/database/db_utils` that simplify repository implementations.

## Pagination Utilities

### GetPaginationValues

Returns validated page, limit, and offset in one call:

```go
import "github.com/industrix-id/backend/pkg/common/utils"

func (r *repo) ListEntities(ctx context.Context, page, limit *int) ([]Entity, *common.PaginationInfo, error) {
    // Returns page, limit, AND offset
    p, l, offset := utils.GetPaginationValues(page, limit)

    query := r.db.WithContext(ctx).
        Model(&schemas.Entity{}).
        Offset(offset).  // Use offset directly
        Limit(l)

    return entities, common.NewPaginationInfo(p, l, total), nil
}
```

**Default values:**
- `page`: defaults to `1` if `nil` or `<= 0`
- `limit`: defaults to `10` if `nil` or `<= 0`
- `offset`: automatically calculated as `(page - 1) * limit`

**❌ WRONG - Manual calculation:**
```go
p := 1
if page != nil && *page > 0 {
    p = *page
}
l := 10
if limit != nil && *limit > 0 {
    l = *limit
}
offset := (p - 1) * l  // Manual calculation
```

**✅ CORRECT - Use utility:**
```go
p, l, offset := utils.GetPaginationValues(page, limit)
```

### NewPaginationInfo

Creates pagination response with automatic type conversion:

```go
// Accepts int64 for total (matches GORM's Count() return type)
return entities, common.NewPaginationInfo(p, l, total), nil

// BEFORE - manual conversion
pagination := &common.PaginationInfo{
    Page:  p,
    Limit: l,
    Total: int(total),  // Manual int64 to int conversion
}
```

## Pointer Utilities

### ToValue

Safely dereferences optional parameters, returns zero value if nil:

```go
import "github.com/industrix-id/backend/pkg/common/utils"

func (r *repo) ListEntities(ctx context.Context, search *string) ([]Entity, error) {
    // Returns "" if search is nil, otherwise *search
    searchTerm := utils.ToValue(search)

    if searchTerm != "" {
        query = query.Where("name ILIKE ?", "%"+searchTerm+"%")
    }
}
```

**❌ WRONG - Manual nil check:**
```go
searchTerm := ""
if search != nil {
    searchTerm = *search
}
```

**✅ CORRECT - Use utility:**
```go
searchTerm := utils.ToValue(search)
```

### ToValueOr

Returns custom default if nil:

```go
func (r *repo) ListEntities(ctx context.Context, sortBy *string) ([]Entity, error) {
    // Returns "created_at DESC" if sortBy is nil
    orderBy := utils.ToValueOr(sortBy, "created_at DESC")

    query = query.Order(orderBy)
}
```

**Common use case - locale defaulting:**
```go
// Always default to English for translations
loc := utils.ToValueOr(locale, "en")

query = query.
    Joins("LEFT JOIN translations t ON t.entity_id = e.id AND t.locale = ?", loc).
    Select("COALESCE(t.name, e.name) as name")
```

**Why default locale to "en"?**
- Eliminates repeated `locale != nil && *locale != ""` checks
- Ensures translations are always fetched
- Simplifies query building (no conditional joins)

## UUID Utilities

### UUIDsToStrings

Converts UUID arrays to string arrays for cache key building:

```go
import "github.com/industrix-id/backend/pkg/common/utils"

func (r *repo) GetActivities(ctx context.Context, userIDs, deviceIDs []uuid.UUID) ([]Activity, error) {
    // Convert UUID arrays to string arrays
    userIDStrs := utils.UUIDsToStrings(userIDs)
    deviceIDStrs := utils.UUIDsToStrings(deviceIDs)

    // Build cache key with string arrays
    cacheKey, err := caches.BuildActivitiesCacheKey(orgID.String(), userIDStrs, deviceIDStrs)
}
```

**❌ WRONG - Manual conversion:**
```go
userIDStrs := make([]string, len(userIDs))
for i, id := range userIDs {
    userIDStrs[i] = id.String()
}
```

**✅ CORRECT - Use utility:**
```go
userIDStrs := utils.UUIDsToStrings(userIDs)
```

**Why this matters:**
- Reduces code duplication
- Returns empty slice for nil/empty input (no nil pointer issues)
- Consistent conversion pattern
- Cache key builders require string arrays for wildcard patterns

## Sorting Utilities

### SortParams

Builds ORDER BY clauses from pre-validated sort parameters:

```go
import "github.com/industrix-id/backend/pkg/common/utils"

func (r *repo) ListEntities(
    ctx context.Context,
    sortField, sortOrder string,  // Pre-validated by service layer
) ([]Entity, error) {
    // Build ORDER BY clause
    sortParams := utils.SortParams{Field: sortField, Order: sortOrder}
    orderByClause := sortParams.BuildOrderByClause()
    // Returns: "field_name ASC" or "field_name DESC"

    query = query.Order(orderByClause)
}
```

**With prefix for multi-column sorting:**
```go
// Build with prefix
sortParams := utils.SortParams{Field: sortField, Order: sortOrder}
orderByClause := sortParams.BuildOrderByClauseWithPrefix("is_admin DESC")
// Returns: "is_admin DESC, field_name ASC"

query = query.Order(orderByClause)
```

**Note:** Repository does NOT validate sort fields - service layer does this using OpenAPI enums.

## Query Helper Functions

From `pkg/database/db_utils/errors.go`:

### ExecuteCountQuery

Executes count query with standardized error handling:

```go
var total int64

query := r.db.WithContext(ctx).
    Model(&schemas.Entity{}).
    Where("deleted_at IS NULL")

// Use schema's TableName() for entity name
if err := ExecuteCountQuery(query, &total, (&schemas.Entity{}).TableName()); err != nil {
    return err
}
```

**Handles:**
- `context.DeadlineExceeded` → 504 Gateway Timeout
- Other errors → 500 with message ID

### ExecutePaginatedQuery

Executes paginated query with ORDER BY:

```go
var results []schemas.Entity

query := r.db.WithContext(ctx).
    Model(&schemas.Entity{}).
    Where("deleted_at IS NULL")

if err := ExecutePaginatedQuery(
    query,
    "created_at DESC",  // ORDER BY clause
    offset,
    limit,
    &results,
    (&schemas.Entity{}).TableName(),
); err != nil {
    return err
}
```

**Applies:**
1. ORDER BY clause
2. OFFSET
3. LIMIT
4. Standardized error handling

### ExecuteQuery

Executes non-paginated query:

```go
var results []schemas.Entity

query := r.db.WithContext(ctx).
    Model(&schemas.Entity{}).
    Where("is_active = ?", true)

if err := ExecuteQuery(query, &results, (&schemas.Entity{}).TableName()); err != nil {
    return err
}
```

**Use for:**
- Queries without pagination
- Loading all records of a small set
- Internal operations

### HandleFindError

Handles errors from single record retrieval (First):

```go
err := r.db.WithContext(ctx).
    Where("id = ? AND deleted_at IS NULL", entityID).
    First(&entity).Error

if err != nil {
    return nil, HandleFindError(err, entityTableName,
        "Entity not found",                  // Not found message
        "error_entity_not_found",            // Not found message ID
        "error_retrieve_entity_failed")      // Generic error message ID
}
```

**Handles:**
- `gorm.ErrRecordNotFound` → 404 with `notFoundMsgID`
- `context.DeadlineExceeded` → 504 with `error_request_deadline_exceeded`
- Other errors → 500 with `retrieveFailedMsgID`

### HandleQueryError

Handles errors from list queries (Find):

```go
err := r.db.WithContext(ctx).
    Model(&schemas.Entity{}).
    Find(&entities).Error

if err != nil {
    return nil, HandleQueryError(err, "find", entityTableName, "error_retrieve_entities_failed")
}
```

**Handles:**
- `context.DeadlineExceeded` → 504 Gateway Timeout
- Other errors → 500 with provided message ID
- **Does NOT** treat "not found" as error (lists can be empty)

## Error Helper Functions

From `pkg/database/db_utils/errors.go`:

### NewDeadlineExceededError

```go
if errors.Is(err, context.DeadlineExceeded) {
    return NewDeadlineExceededError()
    // Returns 504 with "error_request_deadline_exceeded"
}
```

### NewDatabaseError

```go
return NewDatabaseError(
    "Failed to retrieve entity",
    "error_retrieve_entity_failed")
// Returns 500 Internal Server Error
```

### NewDuplicateError

```go
return NewDuplicateError(
    "An entity with this ID already exists",
    "error_entity_duplicate")
// Returns 409 Conflict
```

### NewInvalidRequestError

```go
return NewInvalidRequestError(
    "A required field is missing",
    "error_entity_required_field_missing")
// Returns 400 Bad Request
```

### NewForeignKeyError

```go
return NewForeignKeyError(
    "The selected organization does not exist",
    "error_entity_organization_not_found")
// Returns 400 Bad Request
```

### NewNotFoundError

```go
return NewNotFoundError(
    "Entity not found",
    "error_entity_not_found")
// Returns 404 Not Found
```

## Package-Level Table Name Pattern

Define table name once at package level:

```go
// At package level
var entityTableName = (&schemas.Entity{}).TableName()

// Use throughout repository
func (r *repo) GetEntityByID(ctx context.Context, entityID uuid.UUID) (*schemas.Entity, error) {
    // ...
    return nil, HandleFindError(err, entityTableName, ...)
}

func (r *repo) ListEntities(ctx context.Context, page, limit *int) ([]Entity, *common.PaginationInfo, error) {
    // ...
    if err := ExecuteCountQuery(query, &total, entityTableName); err != nil {
        return nil, nil, err
    }
}
```

**Why?**
- Consistent with schema definition
- No hardcoded strings
- Error messages reference actual table names
- Single point of change if table name changes

## Complete Example

Putting it all together:

```go
package entities

import (
    "context"
    "github.com/google/uuid"
    "gorm.io/gorm"

    "github.com/industrix-id/backend/pkg/common"
    "github.com/industrix-id/backend/pkg/common/caches"
    "github.com/industrix-id/backend/pkg/common/utils"
    "github.com/industrix-id/backend/pkg/database/db_utils"
    "github.com/industrix-id/backend/pkg/database/schemas"
)

// Package-level table name
var entityTableName = (&schemas.Entity{}).TableName()

func (r *entityRepository) ListEntities(
    ctx context.Context,
    orgID uuid.UUID,
    page, limit *int,
    search *string,
    locale *string,
    isActive *bool,
    sortField, sortOrder string,
) ([]schemas.Entity, *common.PaginationInfo, error) {
    // Pagination with offset
    p, l, offset := utils.GetPaginationValues(page, limit)

    // Optional parameter handling
    searchTerm := utils.ToValue(search)
    loc := utils.ToValueOr(locale, "en")

    // Sorting
    sortParams := utils.SortParams{Field: sortField, Order: sortOrder}
    orderByClause := sortParams.BuildOrderByClause()

    // Build cache key
    cacheKey, _ := caches.BuildEntitiesListCacheKey(orgID.String(), p, l, searchTerm, loc, isActive, sortField, sortOrder)

    // Method-level struct
    type cachedResult struct {
        Entities []schemas.Entity
        Total    int64
    }
    var result cachedResult

    err := r.GetFromCacheOrDB(ctx, cacheKey, caches.EntitiesListCacheTTL, &result, func() error {
        query := r.db.WithContext(ctx).
            Model(&schemas.Entity{}).
            Joins("LEFT JOIN translations t ON t.entity_id = entities.id AND t.locale = ?", loc).
            Select("entities.*, COALESCE(t.name, entities.name) as name").
            Where("entities.organization_id = ? AND entities.deleted_at IS NULL", orgID)

        // Apply search filter
        if searchTerm != "" {
            query = query.Where("name ILIKE ?", "%"+searchTerm+"%")
        }

        // Apply is_active filter
        if isActive != nil {
            query = query.Where("entities.is_active = ?", *isActive)
        }

        // Count using helper
        if err := db_utils.ExecuteCountQuery(query, &result.Total, entityTableName); err != nil {
            return err
        }

        // Fetch using helper
        return db_utils.ExecutePaginatedQuery(query, orderByClause, offset, l, &result.Entities, entityTableName)
    })

    if err != nil {
        return nil, nil, err
    }

    return result.Entities, common.NewPaginationInfo(p, l, result.Total), nil
}
```

## Quick Reference

| Utility | Package | Purpose |
|---------|---------|---------|
| `GetPaginationValues` | `pkg/common/utils` | Returns page, limit, offset |
| `NewPaginationInfo` | `pkg/common` | Creates pagination response |
| `ToValue` | `pkg/common/utils` | Dereferences pointer (zero value if nil) |
| `ToValueOr` | `pkg/common/utils` | Dereferences pointer (custom default if nil) |
| `UUIDsToStrings` | `pkg/common/utils` | Converts UUID arrays to strings |
| `SortParams.BuildOrderByClause` | `pkg/common/utils` | Builds ORDER BY clause |
| `ExecuteCountQuery` | `pkg/database/db_utils` | Executes count with error handling |
| `ExecutePaginatedQuery` | `pkg/database/db_utils` | Executes paginated query |
| `ExecuteQuery` | `pkg/database/db_utils` | Executes non-paginated query |
| `HandleFindError` | `pkg/database/db_utils` | Handles single record errors |
| `HandleQueryError` | `pkg/database/db_utils` | Handles list query errors |
| `NewDeadlineExceededError` | `pkg/database/db_utils` | Creates 504 error |
| `NewDatabaseError` | `pkg/database/db_utils` | Creates 500 error |
| `NewDuplicateError` | `pkg/database/db_utils` | Creates 409 error |
| `NewInvalidRequestError` | `pkg/database/db_utils` | Creates 400 error |
| `NewForeignKeyError` | `pkg/database/db_utils` | Creates 400 FK error |
| `NewNotFoundError` | `pkg/database/db_utils` | Creates 404 error |

## Best Practices

### DO

- ✅ Use `GetPaginationValues` for all paginated queries
- ✅ Use `ToValue`/`ToValueOr` for optional parameters
- ✅ Default locale to "en" with `ToValueOr`
- ✅ Use query helpers for consistent error handling
- ✅ Use error builders for standardized responses
- ✅ Define table name at package level
- ✅ Use `UUIDsToStrings` for cache keys

### DON'T

- ❌ No manual pagination calculation
- ❌ No manual pointer nil checks
- ❌ No inline ORDER BY construction
- ❌ No manual UUID array conversion
- ❌ No custom error wrapping (use helpers)
- ❌ No hardcoded table names in errors

## Next Steps

- [How to Understand Repositories](./how-to-understand-repositories.md) - Repository overview
- [How to Create a Repository](./how-to-create-a-repository.md) - Creation guide
- [How to Implement Queries](./how-to-implement-queries.md) - Query patterns
