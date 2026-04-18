---
paths:
  - "pkg/database/**"
  - "pkg/common/caches/**"
---

# Database Caching Rules

## Cache-Aside Pattern

All cached reads go through `GetFromCacheOrDB` from `base.Repository`:

```go
err = r.GetFromCacheOrDB(ctx, cacheKey, caches.DeviceHostCacheTTL, &result, func() error {
    return db.Where("id = ?", id).First(&result).Error
})
```

This checks cache first, falls back to DB, then populates cache. Never implement this logic manually.

## Cache Keys

Build keys with the dedicated builder functions in `pkg/common/caches/`:

```go
cacheKey, err := caches.BuildDeviceCacheKey(deviceID, orgID)
```

Never construct cache key strings manually.

## TTLs

Use constants from `pkg/common/caches/`:

```go
caches.DeviceHostCacheTTL
caches.AccessCardListCacheTTL
```

Never hardcode TTL durations.

## Invalidation on Writes

Every Create, Update, and Delete operation MUST invalidate the relevant cache:

```go
func (r *repo) UpdateDevice(ctx context.Context, deviceID, orgID uuid.UUID, updates map[string]any) error {
    // ... update logic ...
    if !transaction.InTransaction(ctx) {
        r.InvalidateDeviceCache(deviceID, orgID)
    }
    return nil
}
```

**Inside a transaction**: Skip invalidation — the caller invalidates after commit.
**Outside a transaction**: Invalidate immediately.

## Hierarchical Invalidation

Deactivation or deletion of a parent entity cascades invalidation to children:

```go
func (r *repo) InvalidateOnOrganizationDeactivation(orgID uuid.UUID) {
    invalidator := caches.NewHierarchicalCacheInvalidator(r.GetCacheManager(), logger)
    // Cascades to: organization_users, access_cards, sites, devices, FTM caches
}
```

Use `HierarchicalCacheInvalidator` for cascading — never manually invalidate each child cache.
