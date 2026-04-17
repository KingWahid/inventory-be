# How to Invalidate Caches

This guide covers cache invalidation patterns, when to invalidate, and how to implement invalidation methods.

## Core Principle: Service Layer Responsibility

**Cache invalidation happens in the service layer, NOT inside repository methods.**

### Why Service Layer?

- **Transaction safety**: Cache only invalidated after confirmed success
- **Multi-repo operations**: Can batch all invalidations together
- **Explicit**: Clear what's being invalidated
- **Simpler repositories**: Just data access, no mixed concerns

### Pattern

```go
// Repository exposes invalidation helper
func (r *repo) InvalidateEntityCache(entityID uuid.UUID) {
    // Invalidate various caches in goroutines
}

// Service calls after success
func (s *service) CreateEntity(ctx context.Context, ...) error {
    // 1. Create entity
    if err := s.repo.CreateEntity(ctx, nil, entity); err != nil {
        return err
    }

    // 2. Invalidate caches after confirmed success
    s.repo.InvalidateEntityCache(entity.ID)

    return nil
}
```

## Invalidation Patterns

### Per-Entity Cache Invalidation

For single-entity caches (e.g., `GetEntityByID`):

```go
// InvalidateEntityCache invalidates caches for a specific entity.
//
// Invalidates:
//   - Individual entity cache (by entityID)
//
// Cache invalidation runs in goroutines with copied parameters to avoid blocking.
func (r *repo) InvalidateEntityCache(entityID uuid.UUID) {
    if !r.cachingEnabled || r.cacheManager == nil {
        return
    }

    idStr := entityID.String()

    if cacheKey, err := caches.BuildEntityByIDCacheKey(idStr); err == nil {
        go func(key string) {
            r.InvalidateCache(context.Background(), key)
        }(cacheKey)
    }
}
```

**When to use:**
- After `UpdateEntity` or `DeleteEntity` for a single record
- When you know the exact cache key (usually built from entity ID)

### List Cache Pattern Invalidation

For list endpoints (e.g., `ListEntities`):

```go
// InvalidateEntitiesListCache invalidates the entities list cache pattern.
//
// Invalidates:
//   - All paginated list caches (all pages/limits/filters)
//
// Cache invalidation runs in goroutines to avoid blocking.
func (r *repo) InvalidateEntitiesListCache() {
    if !r.cachingEnabled || r.cacheManager == nil {
        return
    }

    if pattern, err := caches.BuildEntitiesListCachePattern(); err == nil {
        go func(p string) {
            r.InvalidateCachePattern(context.Background(), p)
        }(pattern)
    }
}
```

**When to use:**
- After **any** create/update/delete that affects list results
- Always call **in addition to** per-entity invalidation

**Pattern builders return patterns like:**
- `entities:list:*` (matches all list keys)
- `entities:list:org:{orgID}:*` (scoped to organization)

### Complete Invalidation Example

```go
// Create operation
func (s *service) CreateEntity(ctx context.Context, entity *schemas.Entity) error {
    if err := s.repo.CreateEntity(ctx, nil, entity); err != nil {
        return err
    }

    // Invalidate list cache (new entity appears in lists)
    s.repo.InvalidateEntitiesListCache()

    return nil
}

// Update operation
func (s *service) UpdateEntity(ctx context.Context, entityID uuid.UUID, updates map[string]any) error {
    if err := s.repo.UpdateEntity(ctx, nil, entityID, updates); err != nil {
        return err
    }

    // Invalidate both entity + list caches
    s.repo.InvalidateEntityCache(entityID)  // Per-entity
    s.repo.InvalidateEntitiesListCache()     // List pattern

    return nil
}

// Delete operation
func (s *service) DeleteEntity(ctx context.Context, entityID uuid.UUID, deletedBy uuid.UUID) error {
    ctx = base.ContextWithDeletedBy(ctx, deletedBy)
    if err := s.repo.DeleteEntity(ctx, entityID, nil); err != nil {
        return err
    }

    // Invalidate both entity + list caches
    s.repo.InvalidateEntityCache(entityID)
    s.repo.InvalidateEntitiesListCache()

    return nil
}
```

## Hierarchical Cache Invalidation (Deactivation Only)

**ONLY for deactivation (is_active = false), NOT for regular CRUD operations.**

When an entity is deactivated, database triggers cascade the status change to child entities. Cache invalidation must match this cascade.

### Pattern

```go
// Service layer - check if deactivating
func (s *service) UpdateOrganization(ctx context.Context, orgID uuid.UUID, updates map[string]any) error {
    if err := s.orgRepo.UpdateOrganization(ctx, nil, orgID, updates); err != nil {
        return err
    }

    // Check if this is a deactivation
    if isActive, ok := updates["is_active"].(bool); ok && !isActive {
        // Deactivation triggers hierarchical cascade
        s.orgRepo.InvalidateOnOrganizationDeactivation(orgID)
    } else {
        // Regular update - simple cache invalidation
        s.orgRepo.InvalidateOrganizationCache(orgID)
        s.orgRepo.InvalidateOrganizationsListCache()
    }

    return nil
}
```

### Available Cascade Methods

From `pkg/common/caches` `HierarchicalCacheInvalidator`:

```go
// Organization deactivation cascade
InvalidateOnOrganizationDeactivation(orgID uuid.UUID)
// Cascades: org → org_users → access_cards → sites → devices → FTM

// Site deactivation cascade
InvalidateOnSiteDeactivation(orgID, siteID uuid.UUID)
// Cascades: site → devices → FTM

// Device deactivation cascade
InvalidateOnDeviceDeactivation(orgID, deviceID uuid.UUID)
// Cascades: device → device_users → FTM

// FTM quota deactivation cascade
InvalidateOnFTMQuotaDeactivation(quotaID uuid.UUID)
// Cascades: quota → daily usage
```

### When to Use Hierarchical Invalidation

**ONLY when:**
- ✅ `is_active` changes to `false` (deactivation)

**NOT for:**
- ❌ Regular updates, creates, or deletes
- ❌ Simple entities without dependencies (notifications, audit logs)
- ❌ Activation (`is_active: true`)

## Cache Key Patterns

Follow tightly scoped patterns from `docs/plans/automatic-cache-invalidation.md`:

### Naming Conventions

**Use underscores (`_`) in prefixes:**

```go
// ✅ CORRECT
device_config:active:{{.deviceID}}
device_users:{{.orgID}}:{{.deviceID}}
fuel_tank_monitoring_device:{{.deviceID}}:org:{{.orgID}}

// ❌ WRONG - Don't mix dashes and underscores
fuel-tank-monitoring-device:{{.deviceID}}  // Bad
device-config:active:{{.deviceID}}         // Bad
```

**Tightly scoped patterns:**

```go
// ✅ CORRECT - Tightly scoped
listKey := `entities:list:org:{{.orgID}}:p:{{.page}}:l:{{.limit}}`
listPattern := `entities:list:org:{{.orgID}}:*`

// ❌ WRONG - Too broad
listPattern := `*entities*{{.orgID}}*`  // Invalidates unrelated caches
```

### Cache Key Builders

All cache keys must use builders from `pkg/common/caches`:

```go
// ✅ CORRECT - Use caches package
cacheKey, err := caches.BuildEntityByIDCacheKey(entityID.String())

// ❌ WRONG - Inline construction
cacheKey := fmt.Sprintf("entity:%s", entityID.String())
```

**Why use builders?**
- Consistent key format across read and invalidation
- Templates defined once in `pkg/common/caches/*.go`
- Enables proper pattern-based invalidation
- Prevents cache key mismatches

## Documentation Standards

Invalidation methods should document what IS and IS NOT invalidated:

```go
// InvalidateDeviceConfigCache invalidates caches affected by device config changes.
//
// Invalidates:
//   - Device config caches (active config, config by device ID, FTM config)
//   - Device detail caches (device, device record, device host)
//   - Device user caches (capabilities may change with config)
//   - Flow meter and live flow rate caches
//
// Does NOT invalidate: device list cache, available users cache.
//
// Cache invalidation runs in goroutines with copied parameters to avoid blocking.
func (r *deviceConfigRepository) InvalidateDeviceConfigCache(deviceID, orgID uuid.UUID) {
    // Implementation...
}
```

**Key elements:**
- List what IS invalidated (with categories/grouping)
- List what is NOT invalidated (to prevent confusion)
- Note that goroutines are used for non-blocking execution

## Implementation Pattern

### Single Invalidation Method

For repositories with simple cache needs:

```go
// InvalidateEntityCache invalidates caches affected by entity changes.
//
// Invalidates:
//   - Individual entity cache (by entityID)
//   - List caches containing this entity
//
// Cache invalidation runs in goroutines with copied parameters to avoid blocking.
func (r *repo) InvalidateEntityCache(entityID uuid.UUID) {
    if !r.cachingEnabled || r.cacheManager == nil {
        return
    }

    idStr := entityID.String()

    // Invalidate individual cache
    if cacheKey, err := caches.BuildEntityByIDCacheKey(idStr); err == nil {
        go func(key string) {
            r.InvalidateCache(context.Background(), key)
        }(cacheKey)
    }

    // Invalidate list pattern
    if pattern, err := caches.BuildEntitiesListCachePattern(); err == nil {
        go func(p string) {
            r.InvalidateCachePattern(context.Background(), p)
        }(pattern)
    }
}
```

### Multiple Invalidation Methods

For complex repositories with different scopes:

```go
// Invalidate single entity
func (r *repo) InvalidateEntityCache(entityID uuid.UUID) {
    // Invalidate entity:id:{entityID}
}

// Invalidate list for organization
func (r *repo) InvalidateEntitiesListCache(orgID uuid.UUID) {
    // Invalidate entities:list:org:{orgID}:*
}

// Invalidate all user-specific caches
func (r *repo) InvalidateUserEntityCaches(userID uuid.UUID) {
    // Invalidate entities:user:{userID}:*
}
```

## Background Goroutines

**Always run invalidation in background goroutines:**

```go
// ✅ CORRECT - Background invalidation with copied parameters
go func(key string) {
    r.InvalidateCache(context.Background(), key)
}(cacheKey)

// ❌ WRONG - Blocks the caller
r.InvalidateCache(ctx, cacheKey)
```

**Why goroutines?**
- Prevents write operations from blocking on cache cleanup
- Better performance for the caller
- Use `context.Background()` (request context may be cancelled)

**Copy parameters into closure:**

```go
// ✅ CORRECT - Copy parameter
idStr := entityID.String()
go func(key string) {
    r.InvalidateCache(context.Background(), key)
}(idStr)

// ❌ WRONG - Capture variable (race condition)
go func() {
    r.InvalidateCache(context.Background(), entityID.String())
}()
```

## When to Invalidate: Operation Checklist

| Operation | Entity Cache | List Cache | Notes |
|-----------|-------------|------------|-------|
| **Create** | ❌ No | ✅ Yes | New entity only affects lists |
| **Update** | ✅ Yes | ✅ Yes | Both caches need update |
| **Delete** | ✅ Yes | ✅ Yes | Both caches need update |
| **Deactivate** | Special | Special | Use hierarchical cascade |

### Complete Example

```go
// Service coordinates invalidation after operations
type service struct {
    entityRepo repository.EntityRepository
}

func (s *service) CreateEntity(ctx context.Context, entity *schemas.Entity) error {
    if err := s.entityRepo.CreateEntity(ctx, nil, entity); err != nil {
        return err
    }
    s.entityRepo.InvalidateEntitiesListCache()  // ✅ List only
    return nil
}

func (s *service) UpdateEntity(ctx context.Context, entityID uuid.UUID, updates map[string]any) error {
    if err := s.entityRepo.UpdateEntity(ctx, nil, entityID, updates); err != nil {
        return err
    }
    s.entityRepo.InvalidateEntityCache(entityID)  // ✅ Entity
    s.entityRepo.InvalidateEntitiesListCache()     // ✅ List
    return nil
}

func (s *service) DeleteEntity(ctx context.Context, entityID uuid.UUID, deletedBy uuid.UUID) error {
    ctx = base.ContextWithDeletedBy(ctx, deletedBy)
    if err := s.entityRepo.DeleteEntity(ctx, entityID, nil); err != nil {
        return err
    }
    s.entityRepo.InvalidateEntityCache(entityID)  // ✅ Entity
    s.entityRepo.InvalidateEntitiesListCache()     // ✅ List
    return nil
}
```

## Best Practices

### DO

- ✅ Invalidate in service layer after successful operations
- ✅ Use goroutines with copied parameters
- ✅ Use tightly scoped cache patterns
- ✅ Use cache builders from `pkg/common/caches`
- ✅ Document what is/isn't invalidated
- ✅ Check `cachingEnabled` and `cacheManager != nil`
- ✅ Use hierarchical cascade ONLY for deactivation

### DON'T

- ❌ No cache invalidation inside repository methods
- ❌ No broad patterns like `*entity*id*`
- ❌ No inline cache key construction with `fmt.Sprintf`
- ❌ No blocking invalidation (always use goroutines)
- ❌ No hierarchical cascade for regular CRUD
- ❌ No invalidation before operation succeeds

## Next Steps

- [How to Handle Transactions](./how-to-handle-transactions.md) - Write operations
- [How to Handle Errors](./how-to-handle-errors.md) - Error patterns
