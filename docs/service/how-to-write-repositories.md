# How to Write Repositories

Repositories in `pkg/database/repositories/` handle database operations and data access. Each repository is organized in its own directory under `pkg/database/repositories/<entity>/`.

> **📘 Comprehensive Guide:** For complete repository documentation including directory structure, FX module patterns, transaction handling, caching strategies, and best practices, see [`pkg/database/repositories/README.md`](../../../pkg/database/repositories/README.md). It covers:
> - Directory-based structure and file organization
> - FX module pattern and dependency injection
> - Import paths and interface naming conventions
> - Mock generation with Makefile
> - Transaction handling patterns (`transaction.GetDB`)
> - Caching strategies and invalidation
> - UUID conventions and ID types
> - Error handling and custom error codes
> - Testing patterns

> **🤖 AI Reference:** For quick AI-optimized guidance on repository conventions, see [`pkg/database/repositories/README.md`](../../../pkg/database/repositories/README.md) or [`pkg/database/repositories/CLAUDE.md`](../../../pkg/database/repositories/CLAUDE.md).

## Quick Reference: Pagination in Repositories

**Always use `common.PaginationInfo` for pagination in repositories.** Do NOT create custom pagination structs.

```go
// CORRECT: Use common.PaginationInfo
func (r *userRepository) ListUsers(ctx context.Context, page, limit *int) ([]schemas.User, *common.PaginationInfo, error) {
    // ... query logic ...

    pagination := &common.PaginationInfo{
        Page:  actualPage,
        Limit: actualLimit,
        Total: int(total),
    }

    return users, pagination, nil
}

// WRONG: Creating custom pagination struct
type UserPaginationInfo struct {  // DON'T create custom structs
    CurrentPage int
    PerPage     int
    TotalCount  int
}
```

**Why use `common.PaginationInfo`?**
- Consistent across all repositories
- Services and handlers expect this type
- Caching layer uses this type
- Easy to convert to stub types in handlers

## Quick Reference: Transaction Support in Repositories

**Always use `transaction.GetDB(ctx, r.db)` to support context-based transactions.** Repository methods should be transaction-unaware - they automatically participate in service-layer transactions via context.

```go
import "github.com/industrix-id/backend/pkg/database/transaction"

// CORRECT: Repository method supports transactions via context
func (r *userRepository) CreateUser(ctx context.Context, user *schemas.User) error {
    // Extract transaction from context (or use default db)
    db := transaction.GetDB(ctx, r.db)

    // GetDB already applies context; do NOT chain .WithContext(ctx) again
    if err := db.Create(user).Error; err != nil {
        zap.S().With(zap.Error(err)).Error("Failed to create user")
        return common.NewCustomError("Failed to create user").
            WithErrorCode(errorcodes.OtherDatabaseError).
            WithHTTPCode(http.StatusInternalServerError)
    }
    return nil
}

// WRONG: Explicit tx parameter (old pattern - don't use)
func (r *userRepository) CreateUser_OLD(ctx context.Context, tx *gorm.DB, user *schemas.User) error {
    dbConn := r.db
    if tx != nil {
        dbConn = tx
    }
    // ... complex transaction management logic
}
```

**Repository Method Pattern:**
```go
func (r *repository) MethodName(ctx context.Context, ...) error {
    // 1. Get DB connection (tx from context or default db)
    db := transaction.GetDB(ctx, r.db)

    // 2. Use db directly; GetDB already applies context—no redundant .WithContext(ctx)
    return db.Create(...).Error
}
```

**Service Layer Orchestration:**
```go
// Service manages transactions using txManager, repositories participate automatically
func (s *service) CreateUserWithRoles(ctx context.Context, user *schemas.User, roleIDs []uuid.UUID) error {
    err := s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
        // Repository methods automatically use the transaction from context
        if err := s.userRepo.CreateUser(txCtx, user); err != nil {
            return err
        }
        return s.userRoleRepo.AssignRolesToUser(txCtx, user.ID, roleIDs)
    })

    if err != nil {
        return err
    }

    // Cache invalidation AFTER transaction commits
    s.userRepo.InvalidateUserCache(ctx, user.ID)
    return nil
}
```

**Why use context-based transactions?**
- Clean repository method signatures (no `tx *gorm.DB` parameters)
- Service layer controls transactions, repository is transaction-unaware
- Automatic participation when context contains transaction
- Context propagates timeouts, cancellation, and tracing
- Cache invalidation managed by service layer AFTER commits

**Critical Rules:**
- ✅ Use `transaction.GetDB(ctx, r.db)` at the start of repository methods
- ✅ Use the returned `db` directly for operations; do **not** chain `.WithContext(ctx)` after GetDB (it already applies context)
- ❌ Never add `tx *gorm.DB` parameters to repository methods
- ❌ Never manage transactions directly in repositories
- ❌ Never invalidate cache in repository write methods (service layer responsibility)

**See Also:**
- [How to Use Transactions](./how-to-use-transactions.md) - Comprehensive transaction guide
- [Repository Creation Walkthrough](../../walkthroughs/REPOSITORY_CREATION_WALKTHROUGH.md) - Full transaction examples
