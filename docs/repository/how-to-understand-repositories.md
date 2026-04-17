# How to Understand Repositories

This guide explains the repository layer's role, responsibilities, and core principles in the Industrix backend architecture.

## What is a Repository?

A repository is your **database access abstraction layer**. It sits between the service layer and the database, providing a clean interface for data operations.

```
API Handler → Service Layer → Repository Layer → Database
```

## Repository Responsibilities

### What Repositories SHOULD Do

- **Define an interface** with database-centric methods (IDs, filters, pagination)
- **Implement GORM queries** against tables
- **Integrate caching** for read operations (using `pkg/common/caches`)
- **Apply standardized error handling** with `WithMessageID` for i18n support
- **Return schema types** (`pkg/database/schemas`)

### What Repositories MUST NOT Do

- **No HTTP/Echo/OpenAPI types** - Use schema types only
- **No JWT, auth, or permission checks** - Service layer concern
- **No business rules** - "system admin cannot be removed" belongs in services
- **No cross-aggregate orchestration** - Services coordinate multiple repositories

## Core Principles

### One Repository Per Table

Each database table gets exactly one repository:

**✅ CORRECT:**
```
users table              → UserRepository (user.go)
roles table              → RoleRepository (role.go)
organization_users table → OrganizationUserRepository (organization_user.go)
```

**❌ INCORRECT:**
```
UserRepository touching both users + user_roles tables directly
OrganizationUserRepository managing organization_users + user_roles at the same time
```

**When you need data from multiple tables:**
- Use **JOINs** inside a single repo when the primary table is obvious
- Let **services** coordinate writes across multiple repositories

### Directory Structure

```
backend/pkg/database/repositories/
├── users/
│   ├── MODULE.go              # FX module & dependency injection
│   ├── repo.go                # Repository interface & implementation
│   ├── repo_test.go           # Unit tests with mocked dependencies
│   ├── repo_integration_test.go  # Integration tests with real DB
│   └── mocks/
│       └── Repository.go      # Auto-generated mock (via mockery)
├── organizations/
│   ├── MODULE.go
│   ├── repo.go
│   ├── repo_test.go
│   ├── repo_integration_test.go
│   └── mocks/
├── organizations_users/
│   └── ...
├── roles/
│   └── ...
├── sites/
│   └── ...
├── notifications/
│   └── ...
└── [35+ other repositories following the same pattern]
```

**Directory Naming Conventions:**
- **Directory name**: `<entity>` in plural form with underscores (e.g., `users`, `organizations_users`, `device_tokens`)
- **Package name**: Same as directory name (e.g., `package users`, `package organizations_users`)
- **Interface name**: Always `Repository` (not entity-specific like `UserRepository`)
- **Import paths**: `github.com/industrix-id/backend/pkg/database/repositories/users`

**For comprehensive repository documentation**, see [`pkg/database/repositories/README.md`](../../../pkg/database/repositories/README.md)

## Common Repository Structure

Every repository file (`repo.go`) follows this pattern:

```go
// Package entities provides data access layer for entity management.
package entities

import (
    "context"
    "errors"
    "net/http"

    "github.com/google/uuid"
    "go.uber.org/zap"
    "gorm.io/gorm"

    "github.com/industrix-id/backend/pkg/common"
    "github.com/industrix-id/backend/pkg/common/caches"
    "github.com/industrix-id/backend/pkg/common/errorcodes"
    "github.com/industrix-id/backend/pkg/common/utils"
    "github.com/industrix-id/backend/pkg/database/base"
    "github.com/industrix-id/backend/pkg/database/schemas"
    "github.com/industrix-id/backend/pkg/database/transaction"
)

// Repository interface for entity data access.
// Note: Interface is named "Repository", not "EntityRepository"
type Repository interface {
    // READ methods
    GetEntityByID(ctx context.Context, id uuid.UUID) (*schemas.Entity, error)
    ListEntities(ctx context.Context, page, limit *int, search *string) ([]schemas.Entity, *common.PaginationInfo, error)

    // WRITE methods (use transaction.GetDB(ctx, r.db) internally)
    CreateEntity(ctx context.Context, entity *schemas.Entity) error
    UpdateEntity(ctx context.Context, entityID uuid.UUID, updates map[string]any) error
    DeleteEntity(ctx context.Context, entityID uuid.UUID, deletedBy *uuid.UUID) error

    // CACHE invalidation (called by service layer)
    InvalidateEntityCache(entityID uuid.UUID)
}

type entityRepository struct {
    *base.Repository  // Embedded base repository with caching
    db *gorm.DB
}

// NewRepository creates a new entity repository instance.
func NewRepository(db *gorm.DB, cacheManager *caches.CacheManager, cachingEnabled bool) (Repository, error) {
    if db == nil {
        return nil, common.NewCustomError("Database connection is nil").
            WithMessageID("error_db_connection_nil").
            WithErrorCode(errorcodes.InitializationError).
            WithHTTPCode(http.StatusInternalServerError)
    }

    return &entityRepository{
        Repository: base.NewRepository(cacheManager, cachingEnabled),
        db:         db,
    }, nil
}
```

## Key Components

### BaseRepository

All repositories embed `*base.Repository` (from `pkg/database/base/`) which provides:
- **Cache helpers**: `GetFromCacheOrDB`, `InvalidateCache`, `InvalidateCachePattern`
- **Soft delete**: `Delete` method with `deleted_by` tracking
- **Context utilities**: `ContextWithDeletedBy` for propagating user IDs

### Query Helpers

Use shared query helpers from `pkg/database/db_utils/`:

```go
// Execute count query
ExecuteCountQuery(query, &total, (&schemas.Entity{}).TableName())

// Execute paginated query
ExecutePaginatedQuery(query, "created_at DESC", offset, limit, &results, (&schemas.Entity{}).TableName())

// Handle find errors (single record)
HandleFindError(err, entityTableName, "Entity not found", "error_entity_not_found", "error_retrieve_entity_failed")

// Handle query errors (list queries)
HandleQueryError(err, "count", entityTableName, "error_count_entities_failed")
```

### Utility Functions

Use shared utilities from `pkg/common/utils`:

```go
// Pagination with offset
p, l, offset := utils.GetPaginationValues(page, limit)

// Pointer dereferencing
searchTerm := utils.ToValue(search)          // Returns "" if nil
sortBy := utils.ToValueOr(sortBy, "created_at DESC")  // Custom default

// UUID array conversion for cache keys
userIDStrs := utils.UUIDsToStrings(userIDs)
```

## Schema Field Types

In `pkg/database/schemas/`, field types indicate database nullability:

| Go Type | Database | When to Use |
|---------|----------|-------------|
| `*string`, `*uuid.UUID`, `*time.Time` | **Nullable** (can be NULL) | Optional fields |
| `string`, `uuid.UUID`, `time.Time`, `bool` | **NOT NULL** | Required fields |

**Determine which to use from GORM tag:**

```go
type User struct {
    // Nullable fields - use pointers (no "not null" in tag)
    ProfilePictureURL        *string    `gorm:"column:profile_picture_url;size:512"`
    DeletedBy                *uuid.UUID `gorm:"column:deleted_by"`

    // NOT NULL fields - use non-pointers (has "not null" in tag)
    Email    string    `gorm:"column:email;size:255;not null;unique"`
    Name     string    `gorm:"column:name;size:255;not null"`
    ID       uuid.UUID `gorm:"column:id;type:uuid;primaryKey"`
}
```

## Transaction Pattern

Repositories use **context-based transaction propagation**. Transactions are passed via context, not as explicit parameters:

```go
import "github.com/industrix-id/backend/pkg/database/transaction"

func (r *repo) CreateEntity(ctx context.Context, entity *schemas.Entity) error {
    // GetDB extracts transaction from context, or uses r.db as default
    db := transaction.GetDB(ctx, r.db)

    if err := db.Create(entity).Error; err != nil {
        return r.handleCreateEntityError(err, entity)
    }
    return nil
}
```

**Key principles:**
- Use `transaction.GetDB(ctx, r.db)` to get the database connection
- If a transaction is active in context, it returns the transaction
- If no transaction, it returns the default connection
- No `tx *gorm.DB` parameter needed in repository methods

## Cache Invalidation

**Cache invalidation is the service layer's responsibility**, not the repository's.

Repositories expose invalidation helpers that services call after successful operations:

```go
// Repository exposes helper
func (r *repo) InvalidateEntityCache(entityID uuid.UUID) {
    // Invalidate various caches in goroutines
}

// Service calls after success
func (s *service) CreateEntity(ctx context.Context, ...) error {
    if err := s.repo.CreateEntity(ctx, entity); err != nil {
        return err
    }
    // Only invalidate after confirmed success
    s.repo.InvalidateEntityCache(entity.ID)
    return nil
}
```

## Next Steps

- [How to Create a Repository](./how-to-create-a-repository.md) - Step-by-step creation guide
- [How to Implement Queries](./how-to-implement-queries.md) - Read operations and pagination
- [How to Handle Errors](./how-to-handle-errors.md) - Error patterns with WithMessageID
- [How to Use Utilities](./how-to-use-utilities.md) - Shared utilities reference
