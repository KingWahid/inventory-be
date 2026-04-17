# How to Use Database Transactions

This guide explains the context-based transaction pattern used throughout the Industrix backend. It covers both service layer orchestration and repository layer implementation.

## Table of Contents

- [Overview](#overview)
- [When to Use Transactions](#when-to-use-transactions)
- [Transaction Package API](#transaction-package-api)
- [Service Layer Implementation](#service-layer-implementation)
  - [Multi-Repository Transactions](#multi-repository-transactions)
  - [Single-Repository Operations](#single-repository-operations)
  - [Nested Transactions](#nested-transactions)
- [Repository Layer Implementation](#repository-layer-implementation)
- [Cache Invalidation Rules](#cache-invalidation-rules)
- [Common Patterns](#common-patterns)
- [Anti-Patterns to Avoid](#anti-patterns-to-avoid)
- [Troubleshooting](#troubleshooting)
- [Migration from Old Pattern](#migration-from-old-pattern)

---

## Overview

The Industrix backend uses a **context-based transaction propagation pattern** that provides:

- **Clean separation of concerns**: Service layer orchestrates transactions, repository layer is transaction-unaware
- **Type-safe propagation**: Transactions passed via context, not function parameters
- **Automatic participation**: Repository methods automatically join transactions when present
- **Context preservation**: Deadlines, cancellation, and tracing propagate with transactions

### Core Principle

**Service layer orchestrates transactions. Repository layer participates automatically via context.**

```
┌─────────────────────────────────────────────────┐
│              Service Layer                      │
│  • Starts transactions                          │
│  • Wraps tx in context with WithTx()           │
│  • Passes context to repositories               │
│  • Invalidates cache AFTER commit              │
└─────────────────────────────────────────────────┘
                        │
                        ▼ (txCtx)
┌─────────────────────────────────────────────────┐
│            Repository Layer                     │
│  • Extracts tx from context with GetDB()       │
│  • Uses tx if present, else default db         │
│  • Transaction-unaware implementation           │
│  • NO cache invalidation in write methods      │
└─────────────────────────────────────────────────┘
```

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## When to Use Transactions

### Use Transactions When:

✅ **Multiple repositories are involved** - Ensures atomicity across tables
```go
// Create user + organization membership + roles
s.userRepo.CreateUser(txCtx, user)
s.orgUserRepo.CreateOrganizationUser(txCtx, orgUser)
s.userRoleRepo.AssignRolesToUser(txCtx, user.ID, roleIDs)
```

✅ **Business logic requires atomicity** - All-or-nothing operations
```go
// Transfer device ownership (must update device + old owner + new owner)
s.deviceRepo.UpdateDevice(txCtx, device)
s.deviceUserRepo.RemoveUser(txCtx, oldOwnerID)
s.deviceUserRepo.AddUser(txCtx, newOwnerID)
```

✅ **Dependent operations** - Later operations depend on earlier ones
```go
// Delete organization and all related data
s.userRoleRepo.DeleteOrgUserRoles(txCtx, orgID)
s.orgUserRepo.DeleteOrgUsers(txCtx, orgID)
s.organizationRepo.DeleteOrganization(txCtx, orgID)
```

### Don't Use Transactions When:

❌ **Single repository write operation** - Unnecessary overhead
```go
// Simple update - no transaction needed
user.Name = newName
s.userRepo.UpdateUser(ctx, user)
```

❌ **Read-only operations** - Transactions add no value
```go
// Just fetching data - no transaction needed
user, err := s.userRepo.GetUserByID(ctx, userID)
```

❌ **Independent operations that can fail separately** - Don't force atomicity
```go
// Sending notification can fail without affecting user creation
user, err := s.userRepo.CreateUser(ctx, user)
// Send notification separately (not in transaction)
s.notificationService.SendWelcomeEmail(ctx, user.Email)
```

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Transaction Package API

Import the transaction package:
```go
import "github.com/industrix-id/backend/pkg/database/transaction"
```

### `txManager.RunInTx(ctx, fn) error`

**Purpose**: Executes a function within a transaction (service layer use)

**Parameters**:
- `ctx context.Context` - The original context
- `fn func(context.Context) error` - Function to execute within transaction

**Returns**: Error from function execution or transaction management

**Usage**:
```go
// Recommended: Use txManager (available in service struct)
err := s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
    return s.userRepo.CreateUser(txCtx, user)
})
```

**How it works**: `RunInTx` internally wraps the GORM transaction in the context using `transaction.WithTx`, so you don't need to do it manually.

### `transaction.WithTx(ctx, tx) context.Context`

**Purpose**: Wraps a GORM transaction in a context (internal use by txManager)

**Note**: This is called internally by `txManager.RunInTx`. You typically don't need to call this directly.

**Parameters**:
- `ctx context.Context` - The original context
- `tx *gorm.DB` - The GORM transaction to wrap

**Returns**: A new context containing the transaction

### `transaction.GetDB(ctx, defaultDB) *gorm.DB`

**Purpose**: Extracts transaction from context or returns default DB (repository layer use)

**Parameters**:
- `ctx context.Context` - The context that may contain a transaction
- `defaultDB *gorm.DB` - The default database connection to use if no transaction

**Returns**: Transaction from context if present, otherwise defaultDB

**Usage**:
```go
func (r *userRepository) CreateUser(ctx context.Context, user *schemas.User) error {
    db := transaction.GetDB(ctx, r.db)  // Get tx or default db
    return db.WithContext(ctx).Create(user).Error
}
```

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Service Layer Implementation

### Multi-Repository Transactions

Use `s.txManager.RunInTx()` to orchestrate atomic operations across multiple repositories.

**Pattern**:
```go
func (s *service) OperationName(ctx context.Context, ...) error {
    // Execute repositories within transaction
    err := s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
        // Call repository methods with txCtx
        if err := s.repo1.Method1(txCtx, ...); err != nil {
            return err  // Triggers rollback
        }

        if err := s.repo2.Method2(txCtx, ...); err != nil {
            return err  // Triggers rollback
        }

        // All succeeded - transaction commits
        return nil
    })

    if err != nil {
        return err
    }

    // Invalidate caches AFTER successful commit
    s.repo1.InvalidateCache(ctx, ...)
    s.repo2.InvalidateCache(ctx, ...)

    return nil
}
```

**Real Example**:
```go
func (s *service) CreateUserWithRoles(
    ctx context.Context,
    orgID uuid.UUID,
    name, email string,
    roleIDs []uuid.UUID,
) (*schemas.User, error) {
    var user *schemas.User

    err := s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
        // 1. Create user
        user = &schemas.User{
            ID:    uuid.New(),
            Name:  name,
            Email: email,
        }
        if err := s.userRepo.CreateUser(txCtx, user); err != nil {
            return err
        }

        // 2. Create organization user
        orgUser := &schemas.OrganizationUser{
            UserID:         user.ID,
            OrganizationID: orgID,
            IsActive:       true,
        }
        if err := s.orgUserRepo.CreateOrganizationUser(txCtx, orgUser); err != nil {
            return err
        }

        // 3. Assign roles
        if len(roleIDs) > 0 {
            if err := s.userRoleRepo.AssignRolesToUser(txCtx, user.ID, roleIDs); err != nil {
                return err
            }
        }

        return nil
    })

    if err != nil {
        return nil, err
    }

    // Cache invalidation AFTER transaction commits
    s.userRepo.InvalidateUserCache(ctx, user.ID)
    s.orgUserRepo.InvalidateOrganizationUserCache(ctx, user.ID, orgID)
    s.userRoleRepo.InvalidateUserRolesCache(ctx, user.ID, orgID)

    return user, nil
}
```

### Single-Repository Operations

Single-repository operations don't need explicit transactions:

```go
func (s *service) UpdateUserProfile(ctx context.Context, userID uuid.UUID, name string) error {
    user, err := s.userRepo.GetUserByID(ctx, userID)
    if err != nil {
        return err
    }

    user.Name = name
    return s.userRepo.UpdateUser(ctx, user)
}
```

### Nested Transactions

**IMPORTANT**: GORM doesn't support true nested transactions. Use savepoints if needed.

```go
// Top-level transaction
err := s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
    // Calling RunInTx again will reuse the same transaction (nested transaction flattens)
    err := s.txManager.RunInTx(txCtx, func(innerTxCtx context.Context) error {
        // This is the SAME transaction as outer txCtx
        return s.userRepo.CreateUser(innerTxCtx, user)
    })

    return err
})
```

**Avoid nested transactions** - design service methods to accept context and let the caller manage transactions.

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Repository Layer Implementation

Repository methods should use `transaction.GetDB()` to automatically participate in transactions.

### Standard Repository Method Pattern

```go
func (r *repository) WriteMethod(ctx context.Context, ...) error {
    // Extract transaction from context (or use default db)
    db := transaction.GetDB(ctx, r.db)

    // Use db.WithContext(ctx) for all operations
    return db.WithContext(ctx).Create(...).Error
}

func (r *repository) ReadMethod(ctx context.Context, ...) (*Schema, error) {
    // Same pattern for reads (though transactions less important)
    db := transaction.GetDB(ctx, r.db)

    var result Schema
    err := db.WithContext(ctx).Where(...).First(&result).Error
    return &result, err
}
```

### Complete Repository Example

```go
type userRepository struct {
    *BaseRepository
    db *gorm.DB
}

func (r *userRepository) CreateUser(ctx context.Context, user *schemas.User) error {
    db := transaction.GetDB(ctx, r.db)

    if err := db.WithContext(ctx).Create(user).Error; err != nil {
        zap.S().With(zap.Error(err)).Error("Failed to create user")
        return common.NewCustomError("Failed to create user").
            WithErrorCode(errorcodes.OtherDatabaseError).
            WithHTTPCode(http.StatusInternalServerError)
    }
    return nil
}

func (r *userRepository) UpdateUser(ctx context.Context, user *schemas.User) error {
    db := transaction.GetDB(ctx, r.db)

    if err := db.WithContext(ctx).Save(user).Error; err != nil {
        zap.S().With(zap.Error(err)).Error("Failed to update user")
        return common.NewCustomError("Failed to update user").
            WithErrorCode(errorcodes.OtherDatabaseError).
            WithHTTPCode(http.StatusInternalServerError)
    }
    return nil
}

func (r *userRepository) DeleteUser(ctx context.Context, userID uuid.UUID, deletedBy *uuid.UUID) error {
    db := transaction.GetDB(ctx, r.db)

    updates := map[string]interface{}{
        "deleted_at": time.Now(),
    }
    if deletedBy != nil {
        updates["deleted_by"] = *deletedBy
    }

    return db.WithContext(ctx).
        Model(&schemas.User{}).
        Where("id = ?", userID).
        Updates(updates).Error
}

func (r *userRepository) GetUserByID(ctx context.Context, userID uuid.UUID) (*schemas.User, error) {
    db := transaction.GetDB(ctx, r.db)

    var user schemas.User
    err := db.WithContext(ctx).Where("id = ?", userID).First(&user).Error
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, common.NewCustomError("User not found").
                WithErrorCode(errorcodes.UserNotFound).
                WithHTTPCode(http.StatusNotFound)
        }
        return nil, common.NewCustomError("Database error").
            WithErrorCode(errorcodes.OtherDatabaseError).
            WithHTTPCode(http.StatusInternalServerError)
    }
    return &user, nil
}
```

### Key Points

- ✅ **Always use** `db := transaction.GetDB(ctx, r.db)` at the start
- ✅ **Always chain** `.WithContext(ctx)` to propagate timeouts/cancellation
- ✅ **Clean signatures** - no `tx *gorm.DB` parameters needed
- ❌ **Never manage transactions** directly in repositories
- ❌ **Never invalidate cache** in repository write methods (service layer responsibility)

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Cache Invalidation Rules

### Critical Rule: Never Invalidate Cache Inside Transactions

**WRONG** ❌:
```go
err := s.txManager.RunInTx(ctx, func(txCtx context.Context) error {

    if err := s.userRepo.UpdateUser(txCtx, user); err != nil {
        return err
    }

    // DANGEROUS: Cache invalidated before commit!
    s.userRepo.InvalidateUserCache(ctx, user.ID)

    return nil
})
```

**CORRECT** ✅:
```go
err := s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
    return s.userRepo.UpdateUser(txCtx, user)
})

if err != nil {
    return err
}

// Cache invalidation AFTER successful commit
s.userRepo.InvalidateUserCache(ctx, user.ID)
```

### Why This Matters

1. **Database is source of truth**: Cache should only be invalidated after database changes commit
2. **Race conditions**: Other requests may read invalidated cache before transaction commits
3. **Rollback scenarios**: If transaction rolls back, cache should NOT be invalidated

### Cache Invalidation Placement

| Layer | Responsibility | When to Invalidate |
|-------|---------------|-------------------|
| **Service Layer** | Orchestrate cache invalidation | AFTER transaction commits successfully |
| **Repository Layer** | Provide invalidation methods | NEVER call directly in write methods |

```go
// Service layer orchestrates cache invalidation
func (s *service) UpdateUserRoles(ctx context.Context, userID, orgID uuid.UUID, roleIDs []uuid.UUID) error {
    err := s.txManager.RunInTx(ctx, func(txCtx context.Context) error {

        // Delete existing roles
        if err := s.userRoleRepo.DeleteUserRolesInOrganization(txCtx, userID, orgID); err != nil {
            return err
        }

        // Assign new roles
        return s.userRoleRepo.AssignRolesToUser(txCtx, userID, roleIDs)
    })

    if err != nil {
        return err
    }

    // Service layer invalidates cache AFTER commit
    s.userRoleRepo.InvalidateUserRolesCache(ctx, userID, orgID)
    return nil
}

// Repository provides invalidation method (but never calls it internally)
func (r *userRoleRepository) InvalidateUserRolesCache(ctx context.Context, userID, orgID uuid.UUID) {
    cacheKey, err := caches.BuildUserRolesCacheKey(userID, orgID)
    if err == nil {
        r.InvalidateCache(ctx, cacheKey)
    }
}
```

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Common Patterns

### Pattern 1: Create Entity with Relations

```go
func (s *service) CreateDevice(
    ctx context.Context,
    orgID uuid.UUID,
    name string,
    userIDs []uuid.UUID,
) (*schemas.Device, error) {
    var device *schemas.Device

    err := s.txManager.RunInTx(ctx, func(txCtx context.Context) error {

        // Create device
        device = &schemas.Device{
            ID:             uuid.New(),
            OrganizationID: orgID,
            Name:           name,
        }
        if err := s.deviceRepo.CreateDevice(txCtx, device); err != nil {
            return err
        }

        // Create device users
        for _, userID := range userIDs {
            deviceUser := &schemas.DeviceUser{
                DeviceID: device.ID,
                UserID:   userID,
            }
            if err := s.deviceUserRepo.CreateDeviceUser(txCtx, deviceUser); err != nil {
                return err
            }
        }

        return nil
    })

    if err != nil {
        return nil, err
    }

    // Invalidate caches
    s.deviceRepo.InvalidateDeviceCache(ctx, device.ID)
    for _, userID := range userIDs {
        s.deviceUserRepo.InvalidateUserDevicesCache(ctx, userID)
    }

    return device, nil
}
```

### Pattern 2: Update with Cascade

```go
func (s *service) DeactivateUserEverywhere(
    ctx context.Context,
    userID uuid.UUID,
) error {
    err := s.txManager.RunInTx(ctx, func(txCtx context.Context) error {

        // Deactivate user
        if err := s.userRepo.DeactivateUser(txCtx, userID); err != nil {
            return err
        }

        // Deactivate all organization memberships
        if err := s.orgUserRepo.DeactivateUserInAllOrganizations(txCtx, userID); err != nil {
            return err
        }

        // Revoke all device access
        if err := s.deviceUserRepo.RemoveUserFromAllDevices(txCtx, userID); err != nil {
            return err
        }

        return nil
    })

    if err != nil {
        return err
    }

    // Invalidate all affected caches
    s.userRepo.InvalidateUserCache(ctx, userID)
    s.orgUserRepo.InvalidateUserOrganizationsCache(ctx, userID)
    s.deviceUserRepo.InvalidateUserDevicesCache(ctx, userID)

    return nil
}
```

### Pattern 3: Delete with Dependencies

```go
func (s *service) DeleteOrganization(
    ctx context.Context,
    orgID uuid.UUID,
    deletedBy *uuid.UUID,
) error {
    // Check deactivation first (outside transaction)
    org, err := s.organizationRepo.GetOrganizationByID(ctx, orgID)
    if err != nil {
        return err
    }
    if org.IsActive {
        return common.NewCustomError("Organization must be deactivated before deletion").
            WithErrorCode(errorcodes.NotDeactivated).
            WithHTTPCode(http.StatusBadRequest)
    }

    // Delete in transaction
    err = s.txManager.RunInTx(ctx, func(txCtx context.Context) error {

        // Delete in reverse dependency order
        if err := s.userRoleRepo.DeleteAllOrgUserRoles(txCtx, orgID); err != nil {
            return err
        }
        if err := s.orgUserRepo.DeleteAllOrgUsers(txCtx, orgID); err != nil {
            return err
        }
        if err := s.deviceRepo.DeleteAllOrgDevices(txCtx, orgID); err != nil {
            return err
        }
        return s.organizationRepo.DeleteOrganization(txCtx, orgID, deletedBy)
    })

    if err != nil {
        return err
    }

    // Invalidate caches
    s.organizationRepo.InvalidateOrganizationCache(ctx, orgID)
    s.orgUserRepo.InvalidateOrganizationUsersCache(ctx, orgID)
    s.deviceRepo.InvalidateOrganizationDevicesCache(ctx, orgID)

    return nil
}
```

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Anti-Patterns to Avoid

### ❌ Anti-Pattern 1: Passing `tx *gorm.DB` to Repositories

```go
// WRONG: Explicit tx parameter
func (r *userRepository) CreateUser(ctx context.Context, tx *gorm.DB, user *schemas.User) error {
    dbConn := r.db
    if tx != nil {
        dbConn = tx
    }
    return dbConn.Create(user).Error
}

// CORRECT: Use transaction.GetDB
func (r *userRepository) CreateUser(ctx context.Context, user *schemas.User) error {
    db := transaction.GetDB(ctx, r.db)
    return db.WithContext(ctx).Create(user).Error
}
```

### ❌ Anti-Pattern 2: Managing Transactions in Repository

```go
// WRONG: Repository manages transaction
func (r *userRepository) CreateUserWithRoles(ctx context.Context, user *schemas.User, roleIDs []uuid.UUID) error {
    return r.db.Transaction(func(tx *gorm.DB) error {
        // Repository should NOT manage transactions
        if err := tx.Create(user).Error; err != nil {
            return err
        }
        // ...
    })
}

// CORRECT: Service manages transaction
func (s *service) CreateUserWithRoles(ctx context.Context, user *schemas.User, roleIDs []uuid.UUID) error {
    err := s.txManager.RunInTx(ctx, func(txCtx context.Context) error {

        if err := s.userRepo.CreateUser(txCtx, user); err != nil {
            return err
        }
        return s.userRoleRepo.AssignRolesToUser(txCtx, user.ID, roleIDs)
    })

    if err != nil {
        return err
    }

    s.userRepo.InvalidateUserCache(ctx, user.ID)
    return nil
}
```

### ❌ Anti-Pattern 3: Cache Invalidation Inside Transaction

```go
// WRONG: Cache invalidated inside transaction
err := s.txManager.RunInTx(ctx, func(txCtx context.Context) error {

    if err := s.userRepo.UpdateUser(txCtx, user); err != nil {
        return err
    }

    // WRONG: Inside transaction callback
    s.userRepo.InvalidateUserCache(ctx, user.ID)

    return nil
})

// CORRECT: Cache invalidated after commit
err := s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
    return s.userRepo.UpdateUser(txCtx, user)
})

if err != nil {
    return err
}

s.userRepo.InvalidateUserCache(ctx, user.ID)  // After commit
```

### ❌ Anti-Pattern 4: Forgetting `.WithContext(ctx)`

```go
// WRONG: Missing .WithContext(ctx) - timeouts won't work
func (r *userRepository) CreateUser(ctx context.Context, user *schemas.User) error {
    db := transaction.GetDB(ctx, r.db)
    return db.Create(user).Error  // Missing .WithContext(ctx)
}

// CORRECT: Always chain .WithContext(ctx)
func (r *userRepository) CreateUser(ctx context.Context, user *schemas.User) error {
    db := transaction.GetDB(ctx, r.db)
    return db.WithContext(ctx).Create(user).Error
}
```

### ❌ Anti-Pattern 5: Using Original Context Instead of txCtx

```go
// WRONG: Using original ctx instead of txCtx
err := s.txManager.RunInTx(ctx, func(txCtx context.Context) error {

    // WRONG: Passing ctx instead of txCtx
    if err := s.userRepo.CreateUser(ctx, user); err != nil {
        return err
    }
    return nil
})

// CORRECT: Pass txCtx to all repository methods
err := s.txManager.RunInTx(ctx, func(txCtx context.Context) error {

    // CORRECT: Passing txCtx
    if err := s.userRepo.CreateUser(txCtx, user); err != nil {
        return err
    }
    return nil
})
```

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Troubleshooting

### Issue: Transaction not propagating to repository

**Symptom**: Changes not atomic, partial commits happening

**Cause**: Forgot to use `txCtx` when calling repository methods

**Solution**:
```go
// WRONG
err := s.txManager.RunInTx(ctx, func(txCtx context.Context) error {

    // Using ctx instead of txCtx
    return s.userRepo.CreateUser(ctx, user)  // ❌
})

// CORRECT
err := s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
    return s.userRepo.CreateUser(txCtx, user)  // ✅
})
```

### Issue: Cache inconsistency after transaction

**Symptom**: Stale data returned from cache after update

**Cause**: Cache invalidated inside transaction or not at all

**Solution**:
```go
// Cache invalidation AFTER transaction commits
err := s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
    return s.userRepo.UpdateUser(txCtx, user)
})

if err != nil {
    return err
}

// Invalidate AFTER commit
s.userRepo.InvalidateUserCache(ctx, user.ID)
```

### Issue: Context deadline exceeded in transaction

**Symptom**: `context.DeadlineExceeded` errors

**Cause**: Transaction taking too long or not propagating context

**Solution**:
```go
// Ensure .WithContext(ctx) on both transaction and db operations
err := s.txManager.RunInTx(ctx, func(txCtx context.Context) error {

    db := transaction.GetDB(txCtx, r.db)
    return db.WithContext(txCtx).Create(user).Error  // Propagates deadline
})
```

### Issue: Duplicate key errors when re-assigning relationships

**Symptom**: `duplicate key value violates unique constraint`

**Cause**: Join tables using soft delete instead of hard delete

**Solution**: Use hard delete (`.Unscoped()`) for join tables
```go
// CORRECT: Hard delete for join tables
tx.Unscoped().Where("user_id = ? AND role_id IN ?", userID, roleIDs).
    Delete(&schemas.UserRole{})
```

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Migration from Old Pattern

### Old Pattern (Before)

```go
// Repository with explicit tx parameter
func (r *userRepository) CreateUser(ctx context.Context, tx *gorm.DB, user *schemas.User) error {
    dbConn := r.db
    if tx != nil {
        dbConn = tx
    }
    manageTx := tx == nil

    var err error
    if manageTx {
        err = dbConn.Transaction(func(tx *gorm.DB) error {
            return tx.Create(user).Error
        })
    } else {
        err = dbConn.Create(user).Error
    }

    if err == nil {
        r.InvalidateUserCache(ctx, user.ID)  // Cache invalidated in repository
    }
    return err
}

// Service passing tx explicitly
func (s *service) CreateUserWithRoles(ctx context.Context, user *schemas.User, roleIDs []uuid.UUID) error {
    return s.gorm.Transaction(func(tx *gorm.DB) error {
        if err := s.userRepo.CreateUser(ctx, tx, user); err != nil {
            return err
        }
        return s.userRoleRepo.AssignRolesToUser(ctx, tx, user.ID, roleIDs)
    })
}
```

### New Pattern (After)

```go
// Repository using context-based transaction
func (r *userRepository) CreateUser(ctx context.Context, user *schemas.User) error {
    db := transaction.GetDB(ctx, r.db)  // Extract from context

    if err := db.WithContext(ctx).Create(user).Error; err != nil {
        return err
    }
    return nil  // No cache invalidation in repository
}

// Service using txManager.RunInTx
func (s *service) CreateUserWithRoles(ctx context.Context, user *schemas.User, roleIDs []uuid.UUID) error {
    err := s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
        if err := s.userRepo.CreateUser(txCtx, user); err != nil {
            return err
        }
        return s.userRoleRepo.AssignRolesToUser(txCtx, user.ID, roleIDs)
    })

    if err != nil {
        return err
    }

    // Cache invalidation AFTER commit
    s.userRepo.InvalidateUserCache(ctx, user.ID)
    s.userRoleRepo.InvalidateUserRolesCache(ctx, user.ID, orgID)

    return nil
}
```

### Migration Checklist

**Repository Layer:**
- [ ] Remove `tx *gorm.DB` parameter from method signature
- [ ] Add `db := transaction.GetDB(ctx, r.db)` at the start
- [ ] Replace `r.db` usage with `db` variable
- [ ] Ensure `.WithContext(ctx)` is chained on all db operations
- [ ] Remove transaction management logic (if manageTx)
- [ ] Move cache invalidation to service layer

**Service Layer:**
- [ ] Update service to use `s.txManager.RunInTx(ctx, func(txCtx context.Context) error {})`
- [ ] Remove manual transaction handling (`s.gorm.WithContext(ctx).Transaction` and `transaction.WithTx`)
- [ ] Ensure cache invalidation happens AFTER transaction commits

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Related Guides

- [How to Write Service Layer](./how-to-write-service-layer.md)
- [How to Write Repositories](./how-to-write-repositories.md)
- [Repository Creation Walkthrough](../../walkthroughs/REPOSITORY_CREATION_WALKTHROUGH.md)
