# How to Implement Queries

This guide covers read operations, pagination, sorting, and list caching patterns.

## Query Helper Functions

Use shared helpers from `pkg/database/db_utils/errors.go` for consistent error handling:

```go
// Execute count query
ExecuteCountQuery(query *gorm.DB, count *int64, entityName string) error

// Execute paginated query
ExecutePaginatedQuery(query *gorm.DB, orderBy string, offset, limit int, dest interface{}, entityName string) error

// Execute non-paginated query
ExecuteQuery(query *gorm.DB, dest interface{}, entityName string) error

// Handle single record retrieval errors
HandleFindError(err error, entityName, notFoundMsg, notFoundMsgID, retrieveFailedMsgID string) error

// Handle list query errors
HandleQueryError(err error, operation, entityName, messageID string) error
```

## Single Record Retrieval

### Basic Pattern with Caching

```go
func (r *repo) GetEntityByID(ctx context.Context, entityID uuid.UUID) (*schemas.Entity, error) {
    var entity schemas.Entity

    cacheKey, err := caches.BuildEntityByIDCacheKey(entityID.String())
    if err != nil {
        return nil, NewDatabaseError("Failed to build cache key", "error_build_cache_key_failed")
    }

    err = r.GetFromCacheOrDB(ctx, cacheKey, caches.EntityCacheTTL, &entity, func() error {
        err := r.db.WithContext(ctx).
            Where("id = ? AND deleted_at IS NULL", entityID).
            First(&entity).Error

        if err != nil {
            return HandleFindError(err, entityTableName,
                "Entity not found",
                "error_entity_not_found",
                "error_retrieve_entity_failed")
        }
        return nil
    })

    if err != nil {
        return nil, err
    }

    return &entity, nil
}
```

### With Joins

```go
func (r *repo) GetEntityWithRelations(ctx context.Context, entityID uuid.UUID) (*schemas.Entity, error) {
    var entity schemas.Entity

    err := r.db.WithContext(ctx).
        Preload("Related").  // Eager load relations
        Where("id = ? AND deleted_at IS NULL", entityID).
        First(&entity).Error

    if err != nil {
        return nil, HandleFindError(err, entityTableName,
            "Entity not found",
            "error_entity_not_found",
            "error_retrieve_entity_failed")
    }

    return &entity, nil
}
```

## List Queries with Pagination

### Standard Pattern

```go
func (r *repo) ListEntities(
    ctx context.Context,
    page, limit *int,
    search *string,
) ([]schemas.Entity, *common.PaginationInfo, error) {
    // Get pagination values with offset
    p, l, offset := utils.GetPaginationValues(page, limit)

    // Handle optional filter
    searchTerm := utils.ToValue(search)

    // Build cache key
    cacheKey, err := caches.BuildEntitiesListCacheKey(p, l, searchTerm)
    if err != nil {
        return nil, nil, NewDatabaseError("Failed to build cache key", "error_build_cache_key_failed")
    }

    // Method-level struct for caching
    type cachedResult struct {
        Entities []schemas.Entity
        Total    int64
    }
    var result cachedResult

    err = r.GetFromCacheOrDB(ctx, cacheKey, caches.EntitiesListCacheTTL, &result, func() error {
        // Base query
        query := r.db.WithContext(ctx).
            Model(&schemas.Entity{}).
            Where("deleted_at IS NULL")

        // Apply search filter
        if searchTerm != "" {
            query = query.Where("name ILIKE ?", "%"+searchTerm+"%")
        }

        // Count total
        if err := ExecuteCountQuery(query, &result.Total, entityTableName); err != nil {
            return err
        }

        // Fetch paginated data
        return ExecutePaginatedQuery(
            query,
            "created_at DESC",
            offset,
            l,
            &result.Entities,
            entityTableName,
        )
    })

    if err != nil {
        return nil, nil, err
    }

    return result.Entities, common.NewPaginationInfo(p, l, result.Total), nil
}
```

**Key components:**
- `utils.GetPaginationValues(page, limit)` → returns `(page, limit, offset)`
- `utils.ToValue(search)` → safely dereferences optional parameter
- Method-level `cachedResult` struct (scoped to this method)
- `ExecuteCountQuery` and `ExecutePaginatedQuery` from query helpers
- `common.NewPaginationInfo` handles `int64` → `int` conversion

### With Multiple Filters

```go
func (r *repo) ListEntitiesFiltered(
    ctx context.Context,
    orgID uuid.UUID,
    page, limit *int,
    search *string,
    isActive *bool,
) ([]schemas.Entity, *common.PaginationInfo, error) {
    p, l, offset := utils.GetPaginationValues(page, limit)
    searchTerm := utils.ToValue(search)

    cacheKey, err := caches.BuildEntitiesListCacheKey(orgID.String(), p, l, searchTerm, isActive)
    if err != nil {
        return nil, nil, NewDatabaseError("Failed to build cache key", "error_build_cache_key_failed")
    }

    type cachedResult struct {
        Entities []schemas.Entity
        Total    int64
    }
    var result cachedResult

    err = r.GetFromCacheOrDB(ctx, cacheKey, caches.EntitiesListCacheTTL, &result, func() error {
        query := r.db.WithContext(ctx).
            Model(&schemas.Entity{}).
            Where("organization_id = ? AND deleted_at IS NULL", orgID)

        // Apply optional search
        if searchTerm != "" {
            query = query.Where("name ILIKE ?", "%"+searchTerm+"%")
        }

        // Apply optional is_active filter
        if isActive != nil {
            query = query.Where("is_active = ?", *isActive)
        }

        // Count and fetch
        if err := ExecuteCountQuery(query, &result.Total, entityTableName); err != nil {
            return err
        }

        return ExecutePaginatedQuery(query, "created_at DESC", offset, l, &result.Entities, entityTableName)
    })

    if err != nil {
        return nil, nil, err
    }

    return result.Entities, common.NewPaginationInfo(p, l, result.Total), nil
}
```

## Sorting

Repositories receive **pre-validated** sort parameters from the service layer. The service validates using OpenAPI enums.

> See [How to Implement Sorting](../service/how-to-implement-sorting.md) for the full pattern.

**Repository's role is simple:**

```go
func (r *repo) ListEntities(
    ctx context.Context,
    page, limit *int,
    sortField, sortOrder string,  // Pre-validated by service
) ([]schemas.Entity, *common.PaginationInfo, error) {
    p, l, offset := utils.GetPaginationValues(page, limit)

    // Build ORDER BY clause
    sortParams := utils.SortParams{Field: sortField, Order: sortOrder}
    orderByClause := sortParams.BuildOrderByClause()

    // Or with prefix for multi-column sorting
    orderByClause := sortParams.BuildOrderByClauseWithPrefix("is_admin DESC")

    cacheKey, _ := caches.BuildEntitiesListCacheKey(p, l, sortField, sortOrder)

    type cachedResult struct {
        Entities []schemas.Entity
        Total    int64
    }
    var result cachedResult

    r.GetFromCacheOrDB(ctx, cacheKey, caches.EntitiesListCacheTTL, &result, func() error {
        query := r.db.WithContext(ctx).
            Model(&schemas.Entity{}).
            Where("deleted_at IS NULL")

        if err := ExecuteCountQuery(query, &result.Total, entityTableName); err != nil {
            return err
        }

        return ExecutePaginatedQuery(query, orderByClause, offset, l, &result.Entities, entityTableName)
    })

    return result.Entities, common.NewPaginationInfo(p, l, result.Total), nil
}
```

**Key points:**
- Repository does NOT validate sort fields (service does)
- Use `utils.SortParams` to build ORDER BY clauses
- Include sort params in cache keys

## JOINs and Aliases

Use short aliases for joined tables:

```go
// Use alias "ou" for organization_users
query := r.db.WithContext(ctx).
    Joins("JOIN common.organization_users ou ON ou.user_id = common.users.id").
    Where("ou.organization_id = ?", orgID)

// Common aliases:
// ou - organization_users
// ur - user_roles
// rp - role_permissions
```

**Note**: Use `Model()` for the main table, not `Table()` with aliases:

```go
// ✅ CORRECT
query := r.db.WithContext(ctx).
    Model(&schemas.User{}).
    Joins("JOIN common.organization_users ou ON ou.user_id = common.users.id")

// ❌ WRONG - Table() doesn't support aliases
query := r.db.WithContext(ctx).
    Table("common.users u").  // Will fail
    Joins("...")
```

## Translations with Locale

**Always default locale to "en"** for optional locale parameters:

```go
func (r *repo) ListEntitiesWithTranslations(
    ctx context.Context,
    locale *string,
    page, limit *int,
) ([]schemas.Entity, *common.PaginationInfo, error) {
    // Default to English
    loc := utils.ToValueOr(locale, "en")

    p, l, offset := utils.GetPaginationValues(page, limit)
    cacheKey, _ := caches.BuildEntitiesListCacheKey(p, l, loc)

    type cachedResult struct {
        Entities []schemas.Entity
        Total    int64
    }
    var result cachedResult

    r.GetFromCacheOrDB(ctx, cacheKey, caches.EntitiesListCacheTTL, &result, func() error {
        query := r.db.WithContext(ctx).
            Model(&schemas.Entity{}).
            Joins("LEFT JOIN translations t ON t.entity_id = entities.id AND t.locale = ?", loc).
            Select("entities.*, COALESCE(t.name, entities.name) as name").
            Where("entities.deleted_at IS NULL")

        if err := ExecuteCountQuery(query, &result.Total, entityTableName); err != nil {
            return err
        }

        return ExecutePaginatedQuery(query, "created_at DESC", offset, l, &result.Entities, entityTableName)
    })

    return result.Entities, common.NewPaginationInfo(p, l, result.Total), nil
}
```

**Why default to "en":**
- Eliminates repeated `locale != nil && *locale != ""` checks
- Ensures translations are always fetched consistently
- Simplifies query building (no conditional joins)

## Method-Level Caching Structs

For list queries that need both data and total count, use method-level structs:

```go
func (r *repo) ListEntities(ctx context.Context, page, limit *int) ([]schemas.Entity, *common.PaginationInfo, error) {
    p, l, offset := utils.GetPaginationValues(page, limit)
    cacheKey, _ := caches.BuildEntitiesListCacheKey(p, l)

    // ✅ Method-level struct (scoped to this method)
    type cachedResult struct {
        Entities []schemas.Entity
        Total    int64
    }
    var result cachedResult

    r.GetFromCacheOrDB(ctx, cacheKey, caches.EntitiesListCacheTTL, &result, func() error {
        // Populate result.Entities and result.Total
    })

    return result.Entities, common.NewPaginationInfo(p, l, result.Total), nil
}
```

**Why method-level?**
- No namespace pollution (not visible outside the method)
- Clear that it's only for this method's caching
- Self-documenting (struct definition and usage are together)

**Don't use package-level structs for single-method caching:**

```go
// ❌ WRONG - Pollutes package namespace
type entitiesWithTotal struct {
    Entities []schemas.Entity
    Total    int64
}
```

## Query Best Practices

### Always Use Context

```go
// ✅ CORRECT
r.db.WithContext(ctx).Where(...)

// ❌ WRONG - Loses deadline/cancellation
r.db.Where(...)
```

### Always Check Soft Delete

```go
// ✅ CORRECT
Where("deleted_at IS NULL")

// ❌ WRONG - Includes deleted records
// (no deleted_at check)
```

### Use TableName() for Entity Names

```go
// Package-level for reuse
var entityTableName = (&schemas.Entity{}).TableName()

// Use in error messages
return HandleFindError(err, entityTableName, ...)
```

### Count Before Fetch

```go
// ✅ CORRECT - Count first, then fetch
if err := ExecuteCountQuery(query, &total, entityTableName); err != nil {
    return err
}
return ExecutePaginatedQuery(query, orderBy, offset, limit, &results, entityTableName)

// ❌ WRONG - Never compute total from len(results)
results := []Entity{}
query.Find(&results)
total := len(results)  // Wrong! This is paginated count, not total
```

## Next Steps

- [How to Handle Errors](./how-to-handle-errors.md) - Error handling patterns
- [How to Handle Transactions](./how-to-handle-transactions.md) - Write operations
- [How to Invalidate Caches](./how-to-invalidate-caches.md) - Cache patterns
