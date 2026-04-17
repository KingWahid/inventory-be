# How to Write Service Layer

The service layer (`service/service.go`) contains your business logic. It's separate from HTTP handling.

## Table of Contents

- [Service Structure](#service-structure)
- [Error Handling in Services](#error-handling-in-services)
  - [When to Create Custom Error Codes](#when-to-create-custom-error-codes)
  - [Deactivation Check Pattern](#deactivation-check-pattern)
  - [Error Returning Best Practices](#error-returning-best-practices)
  - [Checking Error Codes](#checking-error-codes)
  - [Logging Best Practices](#logging-best-practices)
- [Transaction Management](#transaction-management)
  - [Context-Based Transaction Pattern](#context-based-transaction-pattern)
  - [Service Layer Orchestration](#service-layer-orchestration)
  - [Cache Invalidation After Commits](#cache-invalidation-after-commits)
  - [Transaction Guidelines](#transaction-guidelines)
- [Handling Stale Cache](#handling-stale-cache)
- [Service Method Parameter Ordering](#service-method-parameter-ordering)
- [Claims vs Auth Tokens Pattern](#claims-vs-auth-tokens-pattern)
- [Common Module Dependencies](#common-module-dependencies)
- [Response Building Patterns](#response-building-patterns)

---

## Service Structure

```go
// 1. Define Service interface (what your service can do)
type Service interface {
    // JWT token decoding - thin wrapper for API layer to extract claims from token
    DecodeAuthenticationToken(ctx context.Context, token string) (*jwt.AuthenticationTokenClaims, error)

    // Business logic methods receive claims directly (NOT auth tokens)
    GetUsers(ctx context.Context, claims *jwt.AuthenticationTokenClaims, orgID uuid.UUID) ([]User, error)
    CreateUser(ctx context.Context, claims *jwt.AuthenticationTokenClaims, orgID uuid.UUID, name, email string) (*User, error)
    // ... more methods
}

// 2. Implement service struct (holds dependencies)
type service struct {
    userRepository     repository.UserRepository
    roleRepository     repository.RoleRepository
    cacheManager       *caches.CacheManager
    jwtEncoder         jwt.Encoder
    // ... more dependencies
}

// 3. Constructor function
func NewService(
    dbHost string, dbPort int, /* ... all config */,
) (Service, error) {
    // Connect to database
    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})

    // Initialize repositories
    userRepo, err := repository.NewUserRepository(db, cacheManager, cachingEnabled)

    // Initialize other dependencies (Redis, MQTT, etc.)

    // Return service instance
    return &service{
        userRepository: userRepo,
        // ... assign all dependencies
    }, nil
}

// 4. DecodeAuthenticationToken - thin wrapper for API layer
// This allows the API layer to decode JWT tokens without importing jwt package directly
func (s *service) DecodeAuthenticationToken(ctx context.Context, token string) (*jwt.AuthenticationTokenClaims, error) {
    return s.jwtEncoder.DecodeAuthenticationToken(ctx, token)
}

// 5. Implement interface methods - receive claims, not auth tokens
func (s *service) GetUsers(ctx context.Context, claims *jwt.AuthenticationTokenClaims, orgID uuid.UUID) ([]User, error) {
    // Business logic here - claims available for user-specific logic
    // Example: Get user's timezone for date formatting
    if claims != nil {
        userTimezone, _ := s.getUserTimezone(ctx, claims.UserID)
        // ... use timezone
    }

    users, err := s.userRepository.FindByOrganizationID(ctx, orgID)
    if err != nil {
        return nil, err
    }
    return users, nil
}
```

**Key Points:**
- Service doesn't know about HTTP - it's pure business logic
- Uses repositories for database access
- Takes `context.Context` for cancellation/timeouts
- **Service methods receive `claims` directly, NOT auth tokens** - JWT decoding happens in API layer
- `DecodeAuthenticationToken` is exposed as a thin wrapper for API layer to use
- Returns **OpenAPI stub types** (from `stub/openapi.gen.go`), not custom response structs
- Service converts repository data (schemas) to stub types – this is the conversion layer

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Error Handling in Services

Services are responsible for **business logic validation errors**. Unlike repositories (which handle data access errors), services validate business rules and return appropriate errors.

### When to Create Custom Error Codes

1. **Business rule violations**: "Cannot disable feature X because feature Y depends on it"
2. **Authorization failures**: "User does not have permission to perform this action"
3. **State validation**: "Cannot delete organization that still has active users"
4. **Cross-entity validation**: "Cannot assign role that belongs to different organization"

**Error code placement summary:**

| Layer | Error Type | Example |
|-------|-----------|---------|
| Repository | Data access errors | `FeatureNotFound`, `FailedToRetrieveFeatures` |
| Repository | Data validation (exists check) | `InvalidFeatureID` |
| Service | Business rule validation | `FeatureDependencyViolation` |
| Service | Authorization | `UserNotAuthorized` |
| Service | State validation | `NotDeactivated` (entity must be deactivated before deletion) |

**Adding new error codes:**

1. Add to `pkg/common/errorcodes/error_codes.go`:
```go
// FeatureDependencyViolation indicates cannot disable feature because another enabled feature depends on it (code 18002).
FeatureDependencyViolation ErrorCode = 18002
```

2. Use with `WithMessageID()` for frontend translations:
```go
common.NewCustomError("Cannot disable feature").
    WithMessageID("error_feature_dependency_violation").  // For i18n
    WithErrorCode(errorcodes.FeatureDependencyViolation).
    WithHTTPCode(http.StatusBadRequest)
```

### Deactivation Check Pattern

For entities that have an `is_active` field, they **must be deactivated before deletion**. This is a critical business rule enforced in the service layer.

```go
// In service/organization.go
func (s *service) DeleteOrganization(ctx context.Context, orgID uuid.UUID, deletedBy *uuid.UUID) error {
    // 1. Fetch the entity first
    org, err := s.organizationRepository.GetOrganizationByID(ctx, orgID)
    if err != nil {
        return err
    }

    // 2. Check if entity has is_active field AND is still active
    if org.IsActive {
        return common.NewCustomError("Organization must be deactivated before deletion").
            WithMessageID("error_must_be_deactivated_first").
            WithErrorCode(errorcodes.NotDeactivated).
            WithHTTPCode(http.StatusBadRequest)
    }

    // 3. Proceed with deletion only if deactivated
    return s.organizationRepository.DeleteOrganization(ctx, orgID, deletedBy)
}
```

**When to apply this pattern:**

| Entity has `is_active` field? | Service must check before deletion? |
|------------------------------|-------------------------------------|
| **Yes** (Organizations, Users, Sites, Devices, Roles) | Yes - return `NotDeactivated` error if active |
| **No** (Notifications, Audit logs, Sessions) | No - can delete directly |

**Error code:** `errorcodes.NotDeactivated` (code 4012)

**Example: Feature dependency validation in service layer:**

```go
// In service/features.go
func (s *service) UpdateOrganizationFeatures(ctx context.Context, orgID uuid.UUID, featureIDs []uuid.UUID) error {
    // Get currently enabled features
    currentlyEnabled, err := s.featureRepository.GetOrganizationEnabledFeatureIDs(ctx, orgID)
    if err != nil {
        return err
    }

    // Find features being disabled
    featuresToDisable := findFeaturesToDisable(currentlyEnabled, featureIDs)

    // Business logic validation: can't disable a feature if another enabled feature depends on it
    for _, disablingID := range featuresToDisable {
        dependentFeatures, err := s.featureRepository.GetFeaturesDependingOn(ctx, disablingID)
        if err != nil {
            return err
        }

        for _, dependentID := range dependentFeatures {
            if slices.Contains(featureIDs, dependentID) {
                // Business rule violation - use custom error code
                return common.NewCustomError("Cannot disable feature because another enabled feature depends on it").
                    WithMessageID("error_feature_dependency_violation").
                    WithErrorCode(errorcodes.FeatureDependencyViolation).
                    WithHTTPCode(http.StatusBadRequest)
            }
        }
    }

    // Perform the update (repository handles data access)
    return s.featureRepository.UpdateOrganizationFeatures(ctx, nil, orgID, featureIDs, nil)
}
```

### Error Returning Best Practices

When calling repositories, JWT encoders, password hashers, or other internal dependencies that already return `*common.CustomError`, **do NOT wrap the error** with `common.NewCustomErrorFromError()`. Just return it directly.

**Why?**

`NewCustomErrorFromError` checks if the error is already a `*CustomError` and returns it unchanged:

```go
func NewCustomErrorFromError(err error) *CustomError {
    cE := &CustomError{}
    if errors.As(err, &cE) {
        return cE  // Already CustomError, return as-is
    }
    return &CustomError{Message: err.Error()}  // Wrap plain error
}
```

If you chain `.WithErrorCode()` or `.WithMessageID()` after wrapping, it **overwrites** the original values, losing valuable information (e.g., turning `PasswordTooShort` into generic `PasswordHashingError`, or replacing a specific `error_user_inactive` with generic `error_user_not_found`).

**Examples:**

```go
// CORRECT: Return error directly from dependencies that return *CustomError
func (s *service) ResetPassword(ctx context.Context, resetToken, newPassword string) error {
    // jwtEncoder already returns *CustomError with proper error codes
    resetPasswordClaim, err := s.jwtEncoder.DecodeResetPasswordToken(ctx, resetToken)
    if err != nil {
        return err  // Just return it
    }

    // repository already returns *CustomError with proper error codes
    user, err := s.userRepo.GetUserByEmail(ctx, resetPasswordClaim.Email)
    if err != nil {
        return err  // Just return it
    }

    // passwordHasher returns *CustomError with specific codes (PasswordTooShort, etc.)
    hashedPassword, err := s.passwordHasher.HashPassword(newPassword)
    if err != nil {
        return err  // Preserves specific error codes like PasswordTooShort
    }

    return nil
}

// WRONG: Unnecessary wrapping that can lose error codes
func (s *service) ResetPassword(ctx context.Context, resetToken, newPassword string) error {
    resetPasswordClaim, err := s.jwtEncoder.DecodeResetPasswordToken(ctx, resetToken)
    if err != nil {
        return common.NewCustomErrorFromError(err)  // Unnecessary - already CustomError
    }

    hashedPassword, err := s.passwordHasher.HashPassword(newPassword)
    if err != nil {
        // DANGEROUS: Overwrites PasswordTooShort with generic PasswordHashingError
        return common.NewCustomErrorFromError(err).
            WithErrorCode(errorcodes.PasswordHashingError).
            WithHTTPCode(http.StatusInternalServerError)
    }

    return nil
}

// WRONG: Overwriting MessageID loses specific error context
func (s *service) GetDeviceUser(ctx context.Context, orgID, userID uuid.UUID, locale string) (*User, error) {
    user, err := s.organizationUserRepository.GetOrganizationUserByID(ctx, orgID, userID, locale)
    if err != nil {
        // DANGEROUS: Repository might return "error_user_inactive" or "error_user_not_in_org"
        // but this overwrites it with generic "error_user_not_found"
        return nil, common.NewCustomErrorFromError(err).WithMessageID("error_user_not_found")
    }
    return user, nil
}
```

**When to use `NewCustomErrorFromError`:**

| Scenario | Use wrapping? | Reason |
|----------|---------------|--------|
| Repository returns error | No | Already `*CustomError` |
| JWT encoder returns error | No | Already `*CustomError` |
| Password hasher returns error | No | Already `*CustomError` with specific codes |
| External API returns plain `error` | Yes | Need to convert to `*CustomError` |
| Standard library returns `error` | Yes | Need to convert to `*CustomError` |

**Rule of thumb:** If the dependency is from `pkg/common`, `pkg/database/repository`, or your service's internal packages, it likely already returns `*CustomError`. Just return `err` directly.

### Checking Error Codes

When you need to check if an error has a specific error code (e.g., to handle certain errors differently), use `common.HasErrorCode()` instead of manually creating a `CustomError` to check the code.

```go
// CORRECT: Use HasErrorCode utility
func (s *service) GetDeviceConfig(ctx context.Context, orgID, deviceID uuid.UUID) (*Config, error) {
    config, err := s.deviceConfigRepository.GetConfig(ctx, deviceID)
    if err != nil {
        if common.HasErrorCode(err, errorcodes.DeviceConfigNotFound) {
            // Handle "not found" case differently (e.g., log warning instead of error)
            zap.S().Warn("Device config not found", zap.String("device_id", deviceID.String()))
            return nil, err
        }

        zap.S().With(zap.Error(err)).Error("Failed to get device config")
        return nil, err
    }
    return config, nil
}

// CORRECT: Negated check
func (s *service) deleteConfigIfExists(ctx context.Context, deviceID uuid.UUID) error {
    err := s.deviceConfigRepository.DeleteConfig(ctx, deviceID)
    if err != nil {
        if !common.HasErrorCode(err, errorcodes.DeviceConfigNotFound) {
            // Only log error if it's NOT "not found"
            zap.S().With(zap.Error(err)).Error("Failed to delete config")
            return err
        }
        // "Not found" is OK - config didn't exist
        zap.S().Debug("No config to delete")
    }
    return nil
}

// WRONG: Creating CustomError just to check code
func (s *service) GetDeviceConfig(ctx context.Context, orgID, deviceID uuid.UUID) (*Config, error) {
    config, err := s.deviceConfigRepository.GetConfig(ctx, deviceID)
    if err != nil {
        // DON'T DO THIS - unnecessary allocation and verbose
        customErr := common.NewCustomErrorFromError(err)
        if customErr.Code() == int(errorcodes.DeviceConfigNotFound) {
            // ...
        }
    }
    return config, nil
}
```

**Why use `HasErrorCode`?**
- Cleaner, more readable code
- No unnecessary object creation
- Uses `errors.As` internally for proper error unwrapping
- Works with wrapped errors

### Logging Best Practices

**Be selective with error logging in service layer.** The central error handler sends error responses but doesn't log CustomError details. Service-layer logs provide valuable debugging context.

```go
// CORRECT: Skip logging when URL path already contains the entity IDs
// URL: GET /devices/{device_id}/config - device_id is in path, no need to log it
func (s *service) GetDeviceConfig(ctx context.Context, orgID, deviceID uuid.UUID) (*Config, error) {
    device, err := s.deviceRepository.GetFuelTankDeviceByID(ctx, organizationID, deviceID)
    if err != nil {
        return nil, err  // device_id already in URL path
    }

    config, err := s.deviceConfigRepository.GetConfig(ctx, device.ID)
    if err != nil {
        return nil, err  // device.ID same as deviceID from URL
    }

    return config, nil
}

// CORRECT: Log when there's unique context not available from URL
// URL: GET /devices/{device_id}/users - but we're fetching internal device_user_ids
func (s *service) GetDeviceUsers(ctx context.Context, deviceID uuid.UUID) ([]User, error) {
    deviceUserIDs := []uuid.UUID{...}  // Internal IDs not in URL

    users, err := s.deviceUserRepository.GetDeviceUsersByIDs(ctx, deviceUserIDs)
    if err != nil {
        // Log because deviceUserIDs are internal context not in URL
        zap.S().With(zap.Error(err)).Error("Failed to bulk fetch device users",
            zap.Any("device_user_ids", deviceUserIDs))
        return nil, err
    }
    return users, nil
}
```

**When to log in service layer:**

| Scenario | Log? | Level | Example |
|----------|------|-------|---------|
| Error with IDs already in URL path | No | - | `GET /devices/{id}` fails - id is in path |
| Error with internal IDs not in URL | Yes | `Error` | Bulk fetch with internal device_user_ids |
| Non-fatal issues (fallback used) | Yes | `Warn` | "Config not found, using defaults" |
| Expected conditions (not errors) | Yes | `Debug` | "No device config to delete" |
| Business-significant events | Yes | `Info` | "Device activated successfully" |

```go
// CORRECT: Warn for non-fatal, Debug for expected conditions
func (s *service) deleteConfigIfExists(ctx context.Context, deviceID uuid.UUID) error {
    err := s.deviceConfigRepository.DeleteConfig(ctx, deviceID)
    if err != nil {
        if !common.HasErrorCode(err, errorcodes.DeviceConfigNotFound) {
            return err  // Real error - let central handler log it
        }
        // Expected condition - use Debug, not Error
        zap.S().Debug("No config to delete", zap.String("device_id", deviceID.String()))
    }
    return nil
}

// CORRECT: Warn when using fallback behavior
func (s *service) GetDeviceTimezone(ctx context.Context, deviceID uuid.UUID) string {
    config, err := s.deviceConfigRepository.GetConfig(ctx, deviceID)
    if err != nil {
        zap.S().Warn("Failed to get device config, using default timezone",
            zap.String("device_id", deviceID.String()))
        return "UTC"  // Fallback - operation continues
    }
    return config.Timezone
}
```

**Guidelines:**
- Skip logging if URL path already provides entity IDs (avoid redundancy)
- Log when there's unique internal context (internal IDs, computed state)
- Always use appropriate log levels (Warn for non-fatal, Debug for expected conditions)
- Error messages in CustomError provide context; don't repeat in logs

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Transaction Management

The service layer is responsible for orchestrating database transactions across multiple repositories. This ensures that multi-table operations are atomic - either all changes succeed or all are rolled back.

### Context-Based Transaction Pattern

**The codebase uses a unified context-based transaction propagation pattern** via the `transaction.Manager`. This approach:
- Provides automatic transaction lifecycle management (begin, commit, rollback)
- Keeps repository method signatures clean (no `tx *gorm.DB` parameters)
- Allows repositories to automatically participate in service-layer transactions
- Propagates context with deadlines, cancellation, and tracing

#### Core Components

**Service Layer: `txManager.RunInTx(ctx, fn)`** - Executes a function within a transaction:
```go
import "github.com/industrix-id/backend/pkg/database/transaction"

// Service layer uses txManager for transaction orchestration
err := s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
    // Repository methods automatically use the transaction
    if err := s.userRepository.CreateUser(txCtx, user); err != nil {
        return err  // Triggers automatic rollback
    }
    return s.orgUserRepository.CreateOrganizationUser(txCtx, orgUser)
})
```

**Repository Layer: `transaction.GetDB(ctx, r.db)`** - Extracts transaction from context or returns default db:
```go
// Repository method extracts transaction from context
func (r *userRepository) CreateUser(ctx context.Context, user *schemas.User) error {
    db := transaction.GetDB(ctx, r.db)  // Uses tx if present, else r.db
    return db.WithContext(ctx).Create(user).Error
}
```

### Service Layer Orchestration

The service layer manages transactions when operations span multiple repositories. Use `s.txManager.RunInTx()` to create atomic operations.

#### Multi-Repository Transaction Pattern

```go
// CORRECT: Service orchestrates multi-repository transaction
func (s *service) DeleteUserFromOrganization(
    ctx context.Context,
    userID, orgID uuid.UUID,
    deletedBy *uuid.UUID,
) error {
    // Execute all operations in transaction
    err := s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
        // 1. Delete user roles (UserRoleRepository)
        if err := s.userRoleRepo.DeleteUserRolesInOrganization(txCtx, userID, orgID); err != nil {
            return err  // Triggers automatic rollback
        }

        // 2. Delete organization user (OrganizationUserRepository)
        if err := s.orgUserRepo.DeleteOrganizationUser(txCtx, userID, orgID); err != nil {
            return err  // Triggers automatic rollback
        }

        // All operations succeeded - transaction will auto-commit
        return nil
    })

    if err != nil {
        return err
    }

    // CRITICAL: Invalidate caches AFTER successful transaction commit
    s.userRoleRepo.InvalidateUserRolesCache(ctx, userID, orgID)
    s.orgUserRepo.InvalidateOrganizationUserCache(ctx, orgID, userID)

    return nil
}
```

#### Single-Repository Operations

Not all operations need explicit transactions. Single-repository write operations can rely on the repository's internal transaction handling:

```go
// CORRECT: Single-repository operation - no explicit transaction needed
func (s *service) UpdateUserProfile(ctx context.Context, userID uuid.UUID, name string) error {
    user, err := s.userRepo.GetUserByID(ctx, userID)
    if err != nil {
        return err
    }

    user.Name = name
    return s.userRepo.UpdateUser(ctx, user)
}
```

### Cache Invalidation After Commits

**CRITICAL RULE: Never invalidate cache inside a transaction.**

Cache invalidation must happen AFTER the transaction successfully commits. If you invalidate cache inside the transaction and the transaction rolls back, the cache will be inconsistent with the database.

```go
// CORRECT: Cache invalidation AFTER transaction commits
func (s *service) UpdateUserRoles(ctx context.Context, userID, orgID uuid.UUID, roleIDs []uuid.UUID) error {
    err := s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
        // Perform all database operations inside transaction
        if err := s.userRoleRepo.DeleteUserRolesInOrganization(txCtx, userID, orgID); err != nil {
            return err
        }
        return s.userRoleRepo.AssignRolesToUser(txCtx, userID, roleIDs)
    })

    if err != nil {
        return err
    }

    // Cache invalidation happens AFTER successful commit
    s.userRoleRepo.InvalidateUserRolesCache(ctx, userID, orgID)
    return nil
}

// WRONG: Cache invalidation inside transaction (race condition!)
func (s *service) UpdateUserRoles_Wrong(ctx context.Context, userID, orgID uuid.UUID, roleIDs []uuid.UUID) error {
    return s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
        if err := s.userRoleRepo.DeleteUserRolesInOrganization(txCtx, userID, orgID); err != nil {
            return err
        }

        // DANGEROUS: Cache invalidated before commit!
        // If transaction rolls back, cache will be inconsistent
        s.userRoleRepo.InvalidateUserRolesCache(ctx, userID, orgID)

        return s.userRoleRepo.AssignRolesToUser(txCtx, userID, roleIDs)
    })
}
```

**Why this matters:**
- **Consistency**: Database is the source of truth - cache must reflect committed data
- **Race conditions**: Other requests may read stale cache before transaction commits
- **Rollback scenarios**: If transaction fails, cache should NOT be invalidated

### Transaction Guidelines

**Service Layer Responsibilities:**
- ✅ Use `s.txManager.RunInTx(ctx, func(txCtx context.Context) error {...})` for multi-repository operations
- ✅ Pass `txCtx` (from the RunInTx callback) to all repository methods within the transaction
- ✅ Invalidate caches AFTER transaction commits successfully (outside the RunInTx callback)
- ✅ Return errors from transaction callback to trigger automatic rollback
- ❌ Never invalidate cache inside a transaction
- ❌ Never manage transactions in repository methods (repositories are transaction-unaware)

**Repository Layer Responsibilities:**
- ✅ Use `db := transaction.GetDB(ctx, r.db)` at the start of each method
- ✅ Always apply `.WithContext(ctx)` to the db for timeout/cancellation propagation
- ✅ Keep method signatures clean - no `tx *gorm.DB` parameters
- ❌ Never manage transactions directly (let service layer orchestrate)
- ❌ Never invalidate cache in repository write methods

**Context Propagation:**
- ✅ Repository methods get transaction from context via `transaction.GetDB(ctx, r.db)`
- ✅ Service layer uses `txManager.RunInTx()` for automatic transaction management
- ✅ All db operations should use `.WithContext(ctx)` for proper timeout/cancellation

#### Complete Real-World Example

```go
// Service: Create user with organization membership and roles
func (s *service) CreateUser(
    ctx context.Context,
    orgID uuid.UUID,
    name, email string,
    roleIDs []uuid.UUID,
) (*schemas.User, error) {
    var user *schemas.User

    // Multi-repository atomic operation
    err := s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
        // 1. Create user record
        user = &schemas.User{
            ID:       uuid.New(),
            Name:     name,
            Email:    email,
            Timezone: "UTC",
        }
        if err := s.userRepo.CreateUser(txCtx, user); err != nil {
            return err  // Triggers automatic rollback
        }

        // 2. Create organization user relationship
        orgUser := &schemas.OrganizationUser{
            UserID:         user.ID,
            OrganizationID: orgID,
            IsActive:       true,
        }
        if err := s.orgUserRepo.CreateOrganizationUser(txCtx, orgUser); err != nil {
            return err  // Triggers automatic rollback
        }

        // 3. Assign roles to user
        if len(roleIDs) > 0 {
            if err := s.userRoleRepo.AssignRolesToUser(txCtx, user.ID, roleIDs); err != nil {
                return err  // Triggers automatic rollback
            }
        }

        // All operations succeeded - transaction will auto-commit
        return nil
    })

    if err != nil {
        return nil, err
    }

    // Invalidate all affected caches AFTER successful transaction
    s.userRepo.InvalidateUserCache(ctx, user.ID)
    s.orgUserRepo.InvalidateOrganizationUserCache(ctx, user.ID, orgID)
    s.userRoleRepo.InvalidateUserRolesCache(ctx, user.ID, orgID)

    return user, nil
}
```

**Key takeaways:**
- Service layer orchestrates transactions using `txManager.RunInTx()`
- Transaction lifecycle (begin, commit, rollback) is fully automatic
- Repository methods automatically participate via `transaction.GetDB(ctx, r.db)`
- Cache invalidation happens AFTER transaction commits, never inside
- Return errors from transaction callback to trigger automatic rollback

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Handling Stale Cache

Repository methods may return cached data. For write operations (create/update), handle stale cache by letting the database be the source of truth:

**Pattern: Handle stale cache at write time**

```go
// CORRECT: Let database handle stale cache
func (s *service) CreateOrUpdateDeviceUser(ctx context.Context, orgID, deviceID, userID uuid.UUID, permissions []string) (*DeviceUser, error) {
    // 1. Check if exists (may return cached data)
    existingUser, _ := s.deviceUserRepo.GetDeviceUserByUserAndDevice(ctx, orgID, userID, deviceID)

    // 2. If found, try to update
    if existingUser != nil {
        existingUser.Permissions = permissions
        if err := s.deviceUserRepo.UpdateDeviceUser(ctx, orgID, existingUser); err != nil {
            // If record was deleted (stale cache), fall through to create
            if common.HasErrorCode(err, errorcodes.SpecifiedResourceDoesNotExists) {
                existingUser = nil
            } else {
                return nil, err
            }
        } else {
            return existingUser, nil
        }
    }

    // 3. Create new record (either didn't exist or was stale)
    newUser := &DeviceUser{UserID: userID, DeviceID: deviceID, Permissions: permissions}
    if err := s.deviceUserRepo.CreateDeviceUser(ctx, orgID, newUser); err != nil {
        return nil, err
    }
    return newUser, nil
}

// WRONG: Double-fetch to validate cache (unnecessary complexity)
func (s *service) CreateOrUpdateDeviceUser_Wrong(ctx context.Context, ...) (*DeviceUser, error) {
    existingUser, err := s.deviceUserRepo.GetDeviceUserByUserAndDevice(ctx, orgID, userID, deviceID)
    if err == nil && existingUser != nil {
        // Unnecessary re-fetch to validate cache
        if freshUser, freshErr := s.deviceUserRepo.GetDeviceUserWithDevice(ctx, existingUser.ID); freshErr == nil {
            existingUser = freshUser
        } else if common.HasErrorCode(freshErr, errorcodes.SpecifiedResourceDoesNotExists) {
            existingUser = nil
        } else {
            return nil, freshErr
        }
    }
    // ... continue with update/create
}
```

**Why this works:**
- The database is the source of truth, not the cache
- If update fails with "not found", we know the cached data was stale
- Falling through to create handles the stale cache case
- One less network round-trip compared to double-fetch

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Service Method Parameter Ordering

Service methods MUST follow this parameter ordering convention for consistency:

```
func (s *service) MethodName(
    ctx context.Context,              // 1. Context - ALWAYS first
    organizationID uuid.UUID,         // 2. Identity - org ID or claims
    resourceID uuid.UUID,             // 3. Resource IDs - deviceID, userID, etc.
    page, limit *int,                 // 4. Pagination - page, then limit
    search *string,                   // 5. Search filter
    additionalFilters []uuid.UUID,    // 6. Additional filters - userIDs, deviceIDs, siteIDs
    locale string,                    // 7. Locale - ALWAYS last (if needed)
) (*stub.ResponseType, error)
```

**Parameter Groups (in order):**

| Order | Group | Examples | Notes |
|-------|-------|----------|-------|
| 1 | Context | `ctx context.Context` | Always first, for cancellation/timeouts |
| 2 | Identity | `organizationID uuid.UUID` | Organization scope for the request |
| 3 | Resource IDs | `deviceID`, `userID`, `quotaID` | Specific entity identifiers |
| 4 | Pagination | `page, limit *int` | Always in this order |
| 5 | Search | `search *string` | Text search filter |
| 6 | Additional filters | `userIDs, deviceIDs []uuid.UUID` | Array filters for narrowing results |
| 7 | Locale | `locale string` | Always last, for translations |

**Examples:**

```go
// CORRECT: Follows parameter ordering convention
func (s *service) GetDeviceUsers(
    ctx context.Context,
    organizationID uuid.UUID,
    page, limit *int,
    search *string,
    userIDs, deviceIDs []uuid.UUID,
    locale string,
) (*stub.DeviceUserInfoList, error)

// CORRECT: With resource ID
func (s *service) GetFuelTankMonitoringDeviceUsers(
    ctx context.Context,
    organizationID, deviceID uuid.UUID,
    page, limit *int,
    search *string,
    locale string,
) (*stub.DeviceUserList, error)

// INCORRECT: locale in wrong position, filters before pagination
func (s *service) GetDeviceUsers(
    ctx context.Context,
    organizationID uuid.UUID,
    locale string,                    // Wrong! Should be last
    userIDs []uuid.UUID,              // Wrong! Filters after pagination
    page, limit *int,
    search *string,
) (*stub.DeviceUserInfoList, error)
```

**Why this convention?**
- **Predictable**: Developers know where to find each parameter type
- **Consistent API calls**: All service methods look similar from the API layer
- **Easy to extend**: New filters go before locale, new resource IDs go after identity

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Claims vs Auth Tokens Pattern

**Why this pattern (claims vs auth tokens)?**

| Old Pattern | New Pattern |
|-------------|-------------|
| `GetUsersByAuthToken(ctx, token)` | `GetUsers(ctx, claims, orgID)` |
| Service decodes JWT internally | API layer decodes JWT once |
| JWT logic duplicated in each method | Single point of JWT decoding |
| Hard to test without mocking JWT | Easy to test with mock claims |

The API layer uses `DecodeAuthenticationToken` once, then passes the claims to all service methods. This keeps JWT handling centralized and makes services easier to test.

**Why `DecodeAuthenticationToken` stays in the service layer (not pkg/common):**

You might consider moving `DecodeAuthenticationToken` to `pkg/common/jwt` as a standalone utility. However, keeping it as a service method is preferred because:

| Concern | Service Method | pkg/common Utility |
|---------|---------------|-------------------|
| **Dependency Injection** | Service owns `jwtEncoder` instance | Where does encoder come from? |
| **Testability** | Mock entire service interface | Need to mock jwt package |
| **Encapsulation** | API layer only depends on service | API needs jwt package + encoder |

The `jwt.Encoder` already exists in `pkg/common/jwt`. The service wrapper exists to provide clean dependency injection and testability. Keep `DecodeAuthenticationToken` in the service layer.

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Common Module Dependencies

The `pkg/common` module provides shared utilities used across all services.

### Key Packages:

#### `pkg/common/initialization`
Provides setup functions:
- `InitSentry()` - Error tracking
- `InitLogger()` - Structured logging (zap)
- `NewEcho()` - Creates Echo instance with middleware
- `BuildPostgresDSN()` - Database connection string builder

#### `pkg/common/jwt`
JWT token handling:
- `jwt.Encoder` - Create/sign tokens
- `jwt.Decoder` - Validate/parse tokens
- `jwt.ExtractUserFromToken()` - Get user info from token

#### `pkg/common/translations`
Internationalization (i18n):
- `LoadServiceLocales()` - Load translation files
- `translations.Translate()` - Get translated messages

#### `pkg/common/caches`
Redis caching:
- `CacheManager` - Manages cache operations
- Cache invalidation helpers

#### `pkg/common/errorcodes`
Standardized error codes:
- `InitializationError`
- `ValidationError`
- `NotFoundError`
- etc.

#### `pkg/database/repository`
Database access layer:
- `UserRepository`, `RoleRepository`, etc.
- Provides CRUD operations
- Handles caching

**How services use it:**
```go
import (
    "github.com/industrix-id/backend/pkg/common/initialization"
    "github.com/industrix-id/backend/pkg/common/jwt"
    "github.com/industrix-id/backend/pkg/database/repository"
)
```

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Response Building Patterns

When building OpenAPI response structs, prefer simpler patterns that reduce intermediate variables.

### OpenAPI Type Aliases vs Distinct Types

Understanding which OpenAPI types are aliases helps write cleaner code:

| OpenAPI Type | Underlying Type | Can use `&` directly? |
|--------------|-----------------|----------------------|
| `openapi_types.UUID` | `uuid.UUID` (type alias) | Yes - `&entity.ID` |
| `openapi_types.Email` | `string` (distinct type) | No - use `utils.StringToOpenAPIEmailPtr()` |
| `openapi_types.Date` | `struct` (distinct type) | No - need conversion |

### Simplifying Response Building

**Use `&` directly for struct fields when types match:**

```go
// CORRECT: Use & directly for UUID (type alias) and string fields
func mapUserToResponse(user *schemas.User) *stub.User {
    defaultActive := true
    return &stub.User{
        Id:       &user.ID,                              // UUID is type alias
        Name:     &user.Name,                            // string field
        Email:    utils.StringToOpenAPIEmailPtr(user.Email), // Email needs conversion
        Timezone: &user.Timezone,                        // string field
        IsActive: &defaultActive,                        // literal needs variable
    }
}

// WRONG: Unnecessary intermediate variables
func mapUserToResponse(user *schemas.User) *stub.User {
    userID := openapi_types.UUID(user.ID)  // Unnecessary conversion
    email := openapi_types.Email(user.Email)
    name := user.Name                       // Unnecessary variable

    return &stub.User{
        Id:    &userID,
        Name:  &name,
        Email: &email,
    }
}
```

**Use `&paginationInfo.Field` directly for pagination:**

```go
// CORRECT: Access struct fields directly
func buildPaginatedResponse(items []stub.Item, paginationInfo *common.PaginationInfo) *stub.ItemList {
    return &stub.ItemList{
        Data:  &items,
        Total: &paginationInfo.Total,
        Page:  &paginationInfo.Page,
        Limit: &paginationInfo.Limit,
    }
}

// WRONG: Extracting to variables first
func buildPaginatedResponse(items []stub.Item, paginationInfo *common.PaginationInfo) *stub.ItemList {
    total, page, limit := paginationInfo.Total, paginationInfo.Page, paginationInfo.Limit
    return &stub.ItemList{
        Data:  &items,
        Total: &total,
        Page:  &page,
        Limit: &limit,
    }
}
```

### When Intermediate Variables ARE Required

Some cases require intermediate variables:

```go
// 1. Function return values - can't take address of return value
actionColor := constants.GetProcessActionByValue(actionValue).Color
responseLog.ActionColor = &actionColor  // Need variable

// 2. Literal values - can't take address of literal
defaultActive := true
user.IsActive = &defaultActive  // Need variable

// 3. Type conversions of non-alias types
email := openapi_types.Email(user.Email)  // Email is NOT a type alias
user.Email = &email  // Need variable OR use utils.StringToOpenAPIEmailPtr()

// 4. Computed values
activeCountInt := int(activeCount)  // Converting int64 to int
response.TotalActive = &activeCountInt  // Need variable
```

### Quick Reference: When to Use What

| Scenario | Pattern |
|----------|---------|
| UUID field from entity | `&entity.ID` |
| String field from entity | `&entity.Name` |
| Email field from entity | `utils.StringToOpenAPIEmailPtr(entity.Email)` |
| Pagination fields | `&paginationInfo.Total` |
| Boolean literal | `isActive := true; &isActive` |
| Function return value | `color := GetColor(); &color` |
| Type conversion (int64→int) | `count := int(countInt64); &count` |

<div align="right"><a href="#table-of-contents">↑ Back to top</a></div>

---

## Related Guides

- [How to Create a Service](./how-to-create-a-service.md)
- [How to Write Handlers](./how-to-write-handlers.md)
- [How to Understand Architecture](./how-to-understand-architecture.md)
