# How to Handle Transactions

This guide covers write operations, transactions, soft delete, and the deactivation-before-deletion pattern.

## Context-Based Transaction Pattern

Repositories use context-based transaction propagation. Transactions are passed via context, not as explicit parameters.

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

## Create Operations

```go
func (r *repo) CreateEntity(ctx context.Context, entity *schemas.Entity) error {
    db := transaction.GetDB(ctx, r.db)

    if err := db.Create(entity).Error; err != nil {
        return r.handleCreateEntityError(err, entity)
    }
    return nil
}
```

**Note**: Caller is responsible for cache invalidation via `InvalidateEntityCache`.

## Update Operations

### Update with map[string]any

```go
func (r *repo) UpdateEntity(
    ctx context.Context,
    entityID uuid.UUID,
    updates map[string]any,
) error {
    db := transaction.GetDB(ctx, r.db)

    result := db.Model(&schemas.Entity{}).
        Where("id = ? AND deleted_at IS NULL", entityID).
        Updates(updates)

    if result.Error != nil {
        return r.handleUpdateEntityError(result.Error)
    }

    if result.RowsAffected == 0 {
        return NewNotFoundError("Entity not found", "error_entity_not_found")
    }

    return nil
}
```

**Always check `RowsAffected`** - a successful query with 0 rows updated means the record doesn't exist.

### Returning Updated Records

After updates, reload from database instead of manual field mapping:

```go
// CORRECT - Reload from database
func (r *repo) UpdateEntity(ctx context.Context, entityID uuid.UUID, updates map[string]any) (*schemas.Entity, error) {
    db := transaction.GetDB(ctx, r.db)
    var entity schemas.Entity

    // Find first
    if err := db.Where("id = ? AND deleted_at IS NULL", entityID).First(&entity).Error; err != nil {
        return nil, HandleFindError(err, entityTableName, "Entity not found", "error_entity_not_found", "error_retrieve_entity_failed")
    }

    // Update
    if err := db.Model(&entity).Updates(updates).Error; err != nil {
        return nil, r.handleUpdateEntityError(err)
    }

    // Reload to get actual DB state
    if err := db.First(&entity, entityID).Error; err != nil {
        return nil, HandleFindError(err, entityTableName, "Entity not found", "error_entity_not_found", "error_retrieve_entity_failed")
    }

    return &entity, nil
}
```

**Why reload?**
- Simpler than manual field mapping
- Returns actual database state (including defaults, triggers, computed columns)
- Consistent pattern across all updates
- One extra query is negligible for correctness

## Delete Operations (Soft Delete)

### Using BaseRepository.Delete

```go
func (r *repo) DeleteEntity(
    ctx context.Context,
    entityID uuid.UUID,
    deletedBy *uuid.UUID,
) error {
    err := r.Delete(
        ctx,
        r.db,
        &schemas.Entity{},
        deletedBy,
        "id = ? AND deleted_at IS NULL",
        entityID,
    )

    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return NewNotFoundError("Entity not found", "error_entity_not_found")
        }
        zap.S().With(zap.Error(err)).Error("Failed to delete entity")
        return NewDatabaseError("Failed to delete entity", "error_delete_entity_failed")
    }

    return nil
}
```

### deleted_by via Context

Instead of passing `deletedBy` to every delete call, use context:

```go
// Service layer sets it once
ctx = base.ContextWithDeletedBy(ctx, actingUserID)

// Repository can pass nil - taken from context automatically
err := repo.DeleteEntity(ctx, entityID, nil)
```

**BaseRepository.Delete will:**
1. Check `deletedBy` parameter first
2. If `nil`, extract from context via `deletedByFromContext(ctx)`
3. Set `deleted_by` and `deleted_at` fields

## Deactivation Before Deletion

**Important rule:** Entities with `is_active` field **must be deactivated** (`is_active: false`) before deletion.

### Why This Rule Exists

- Prevents accidental deletion of active resources
- Ensures proper cleanup workflows are followed
- Gives users a chance to review before permanent deletion
- Maintains data integrity for dependent resources

### Implementation (Service Layer Responsibility)

The **service layer** checks this rule, not the repository:

```go
// Service layer (e.g., service/organization.go)
func (s *service) DeleteOrganization(ctx context.Context, orgID uuid.UUID, deletedBy uuid.UUID) error {
    // 1. Fetch the entity first
    org, err := s.organizationRepository.GetOrganizationByID(ctx, orgID)
    if err != nil {
        return err
    }

    // 2. Check if still active
    if org.IsActive {
        return common.NewCustomError("Organization must be deactivated before deletion").
            WithMessageID("error_must_be_deactivated_first").
            WithErrorCode(errorcodes.NotDeactivated).
            WithHTTPCode(http.StatusBadRequest)
    }

    // 3. Proceed with deletion only if deactivated
    ctx = base.ContextWithDeletedBy(ctx, deletedBy)
    if err := s.organizationRepository.DeleteOrganization(ctx, orgID, nil); err != nil {
        return err
    }

    // 4. Invalidate caches after successful deletion
    s.organizationRepository.InvalidateOrganizationCache(orgID)
    s.organizationRepository.InvalidateOrganizationsListCache()

    return nil
}
```

### When to Apply

| Entity has `is_active`? | Deletion behavior |
|------------------------|-------------------|
| **Yes** (orgs, users, sites, devices, roles) | Must check `is_active == false` before deletion |
| **No** (notifications, audit logs, sessions) | Can delete directly |

**Error code to use:** `errorcodes.NotDeactivated` (code 4012)

**Why not in repository?**
- It's a business rule, not a data access concern
- Different services might have different deactivation requirements
- Some bulk operations might need to bypass with proper authorization

## Transaction Coordination (Service Layer)

Services coordinate transactions using the transaction manager:

```go
import "github.com/industrix-id/backend/pkg/database/transaction"

// Service coordinates multi-repo transaction
func (s *service) CreateUserWithRole(ctx context.Context, user *schemas.User, roleID uuid.UUID) error {
    err := s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
        // Create user - transaction is in txCtx
        if err := s.userRepository.CreateUser(txCtx, user); err != nil {
            return err  // Auto-rollback
        }

        // Assign role - uses same transaction from txCtx
        userRole := &schemas.UserRole{
            UserID: user.ID,
            RoleID: roleID,
        }
        if err := s.userRoleRepository.CreateUserRole(txCtx, userRole); err != nil {
            return err  // Auto-rollback
        }

        return nil  // Auto-commit
    })

    if err != nil {
        return err
    }

    // Transaction committed - now invalidate caches
    s.userRepository.InvalidateUserCache(user.ID)
    s.userRepository.InvalidateUsersListCache()
    return nil
}
```

**Key points:**
- Service owns the transaction via `s.txManager.RunInTx(ctx, fn)`
- Pass `txCtx` (context with transaction) to all repository methods
- Repositories extract transaction via `transaction.GetDB(txCtx, r.db)`
- Cache invalidation happens **after** transaction succeeds

### Transaction Manager Setup

In service initialization:

```go
type service struct {
    txManager   transaction.Manager
    userRepo    repository.UserRepository
    // ...
}

func NewService(db *gorm.DB, ...) *service {
    return &service{
        txManager: transaction.NewManager(db),
        // ...
    }
}
```

## Documentation Standards

Write methods should document cache invalidation responsibility:

```go
// CreateEntity creates a new entity.
// Note: Caller is responsible for cache invalidation via InvalidateEntityCache.
func (r *repo) CreateEntity(ctx context.Context, entity *schemas.Entity) error {
    // ...
}

// UpdateEntity updates entity fields.
// Note: Caller is responsible for cache invalidation via InvalidateEntityCache.
func (r *repo) UpdateEntity(ctx context.Context, entityID uuid.UUID, updates map[string]any) error {
    // ...
}

// DeleteEntity soft deletes an entity.
// Note: Service layer should verify deactivation (if applicable) before calling.
// Note: Caller is responsible for cache invalidation via InvalidateEntityCache.
func (r *repo) DeleteEntity(ctx context.Context, entityID uuid.UUID, deletedBy *uuid.UUID) error {
    // ...
}
```

## Best Practices

### DO

- Use `transaction.GetDB(ctx, r.db)` to get database connection
- Use `txManager.RunInTx()` for multi-repository transactions
- Check `RowsAffected` after updates/deletes
- Use dedicated error handlers for each operation
- Document cache invalidation responsibility
- Reload records after updates instead of manual mapping
- Check deactivation in service layer before delete

### DON'T

- Don't pass `tx *gorm.DB` as method parameters (old pattern)
- Don't invalidate cache inside repository methods
- Don't put business rules in repository (deactivation checks belong in services)
- Don't use manual field mapping helpers
- Don't manage transactions inside repository methods

## Next Steps

- [How to Handle Errors](./how-to-handle-errors.md) - Error handling patterns
- [How to Invalidate Caches](./how-to-invalidate-caches.md) - Cache invalidation strategies
