# How to Implement Caching

This guide explains how to add Redis caching to your service, including cache keys, invalidation patterns, and best practices.

## Table of Contents

- [Quick Start](#quick-start)
- [Understanding Cache Keys vs Patterns](#understanding-cache-keys-vs-patterns)
- [Step-by-Step Implementation](#step-by-step-implementation)
- [TTL Guidelines](#ttl-guidelines)
- [Common Patterns](#common-patterns)
- [Troubleshooting](#troubleshooting)

---

## Quick Start

### The 3 Things You Need

1. **Cache key builder** - Creates exact keys for storing/retrieving data
2. **Pattern builder** - Creates wildcard patterns for invalidation
3. **Invalidation method** - Called after successful write operations

```go
// 1. Build cache key (exact) - for GET/SET
cacheKey, _ := caches.BuildSiteByIDCacheKey(siteID, locale)
// Result: "sites:details:abc-123:en"

// 2. Build pattern (wildcard) - for invalidation
pattern, _ := caches.BuildSiteDetailsPattern(siteID)
// Result: "sites:details:abc-123:*"

// 3. Invalidate after writes
r.InvalidateCachePattern(ctx, pattern)
```

---

## Understanding Cache Keys vs Patterns

### Cache Keys: Exact Match

Used when **reading or writing** cache data:

```go
// Same site, different locales = different cache keys
"sites:details:abc-123:en"  →  {"name": "Factory", "category": "Industrial"}
"sites:details:abc-123:id"  →  {"name": "Factory", "category": "Industri"}
```

### Patterns: Wildcard Match

Used when **invalidating** (deleting) cache data:

```go
// One pattern deletes all locale variants
"sites:details:abc-123:*"  →  Deletes both :en and :id keys
```

### Why Use Wildcards?

Without wildcards, you'd need to know and delete every possible cache key:

```go
// BAD: Must know all locales
r.InvalidateCache(ctx, "sites:details:abc-123:en")
r.InvalidateCache(ctx, "sites:details:abc-123:id")
r.InvalidateCache(ctx, "sites:details:abc-123:zh")
// ... what if new locale is added?

// GOOD: Wildcard catches all
r.InvalidateCachePattern(ctx, "sites:details:abc-123:*")
```

---

## Step-by-Step Implementation

### Step 1: Create Cache Definitions

Create or update `pkg/common/caches/<entity>.go`:

```go
package caches

import (
    "bytes"
    "text/template"
    "time"
)

// ===========================================================================
// TEMPLATES (for exact cache keys)
// ===========================================================================
var (
    // For single entity lookup
    widgetByIDTmpl = template.Must(template.New("widgetByID").Parse(
        `{{.prefix}}:{{.widgetID}}:{{.locale}}`))

    // For list queries
    widgetListTmpl = template.Must(template.New("widgetList").Parse(
        `{{.prefix}}:{{.orgID}}:page:{{.page}}:limit:{{.limit}}:search:{{.search}}`))
)

// ===========================================================================
// PREFIXES (shared between keys and patterns)
// ===========================================================================
const (
    WidgetDetailsPrefix = "widgets:details:"
    WidgetsPrefix       = "widgets:"
)

// ===========================================================================
// PATTERN TEMPLATES (for wildcard invalidation)
// ===========================================================================
var (
    // Invalidate all locale variants of a widget
    widgetDetailsPatternTmpl = template.Must(template.New("widgetDetailsPattern").Parse(
        `{{.prefix}}{{.widgetID}}:*`))

    // Invalidate all list caches for an organization
    widgetListByOrgPatternTmpl = template.Must(template.New("widgetListByOrgPattern").Parse(
        `{{.prefix}}{{.orgID}}:*`))
)

// ===========================================================================
// TTL CONSTANTS
// ===========================================================================
const (
    WidgetDetailsCacheTTL = 30 * time.Minute  // Details: longer TTL
    WidgetListCacheTTL    = 2 * time.Minute   // Lists: shorter TTL
)

// ===========================================================================
// CACHE KEY BUILDERS (return exact keys)
// ===========================================================================

func BuildWidgetByIDCacheKey(widgetID, locale string) (string, error) {
    var buf bytes.Buffer
    if err := widgetByIDTmpl.Execute(&buf, map[string]any{
        "prefix":   WidgetDetailsPrefix,
        "widgetID": widgetID,
        "locale":   locale,
    }); err != nil {
        return "", err
    }
    return buf.String(), nil
}

func BuildWidgetListCacheKey(orgID string, page, limit int, search string) (string, error) {
    var buf bytes.Buffer
    if err := widgetListTmpl.Execute(&buf, map[string]any{
        "prefix": WidgetsPrefix,
        "orgID":  orgID,
        "page":   page,
        "limit":  limit,
        "search": search,
    }); err != nil {
        return "", err
    }
    return buf.String(), nil
}

// ===========================================================================
// PATTERN BUILDERS (return wildcard patterns)
// ===========================================================================

// BuildWidgetDetailsPattern creates a pattern to invalidate all locale variants.
// Example: BuildWidgetDetailsPattern("abc-123") → "widgets:details:abc-123:*"
func BuildWidgetDetailsPattern(widgetID string) (string, error) {
    var buf bytes.Buffer
    if err := widgetDetailsPatternTmpl.Execute(&buf, map[string]any{
        "prefix":   WidgetDetailsPrefix,
        "widgetID": widgetID,
    }); err != nil {
        return "", err
    }
    return buf.String(), nil
}

// BuildWidgetListByOrgPattern creates a pattern to invalidate all list caches for an org.
// Example: BuildWidgetListByOrgPattern("org-456") → "widgets:org-456:*"
func BuildWidgetListByOrgPattern(orgID string) (string, error) {
    var buf bytes.Buffer
    if err := widgetListByOrgPatternTmpl.Execute(&buf, map[string]any{
        "prefix": WidgetsPrefix,
        "orgID":  orgID,
    }); err != nil {
        return "", err
    }
    return buf.String(), nil
}
```

### Step 2: Add Repository Interface Method

```go
type WidgetRepository interface {
    // ... existing CRUD methods ...

    // InvalidateWidgetCache invalidates cache for a widget and its org's list.
    // Call this after any create/update/delete operation.
    InvalidateWidgetCache(ctx context.Context, widgetID, orgID uuid.UUID)
}
```

### Step 3: Implement Cache Invalidation

```go
// InvalidateWidgetCache invalidates cache for a specific widget and its organization's list.
// Uses tightly scoped patterns to avoid over-invalidation.
//
// Runs async (goroutine) so the user doesn't wait for Redis calls after DB commit.
// Uses context.Background() to prevent cancellation if request context is canceled.
func (r *widgetRepository) InvalidateWidgetCache(ctx context.Context, widgetID, orgID uuid.UUID) {
    // Invalidate widget details cache (all locale variants)
    if pattern, err := caches.BuildWidgetDetailsPattern(widgetID.String()); err == nil {
        go func(p string) {
            r.InvalidateCachePattern(context.Background(), p)
        }(pattern)
    }

    // Invalidate organization's widget list cache (all paginated variants)
    if pattern, err := caches.BuildWidgetListByOrgPattern(orgID.String()); err == nil {
        go func(p string) {
            r.InvalidateCachePattern(context.Background(), p)
        }(pattern)
    }
}
```

### Step 4: Call Invalidation in Service Layer

```go
func (s *service) CreateWidget(ctx context.Context, orgID uuid.UUID, req stub.WidgetCreate) (*stub.Widget, error) {
    // 1. Create in database (repository uses transaction.GetDB(ctx, r.db) internally)
    widget := &schemas.Widget{Name: req.Name}
    if err := s.widgetRepository.CreateWidget(ctx, widget); err != nil {
        return nil, err
    }

    // 2. Invalidate cache AFTER successful operation
    s.widgetRepository.InvalidateWidgetCache(ctx, widget.ID, orgID)

    return toWidgetResponse(widget), nil
}

func (s *service) UpdateWidget(ctx context.Context, orgID, widgetID uuid.UUID, req stub.WidgetUpdate) (*stub.Widget, error) {
    // 1. Update in database (repository uses transaction.GetDB(ctx, r.db) internally)
    if err := s.widgetRepository.UpdateWidget(ctx, widgetID, updates); err != nil {
        return nil, err
    }

    // 2. Invalidate cache AFTER successful operation
    s.widgetRepository.InvalidateWidgetCache(ctx, widgetID, orgID)

    return s.widgetRepository.GetWidgetByID(ctx, widgetID, locale)
}

func (s *service) DeleteWidget(ctx context.Context, orgID, widgetID uuid.UUID) error {
    // 1. Delete from database (repository uses transaction.GetDB(ctx, r.db) internally)
    if err := s.widgetRepository.DeleteWidget(ctx, widgetID); err != nil {
        return err
    }

    // 2. Invalidate cache AFTER successful operation
    s.widgetRepository.InvalidateWidgetCache(ctx, widgetID, orgID)

    return nil
}
```

---

## TTL Guidelines

| Cache Type | TTL | Reason |
|------------|-----|--------|
| **Entity details** | 15-60 min | Rarely changes, explicit invalidation on update |
| **Entity lists** | 1-2 min | Many factors affect list content, short TTL safer |
| **Lists with embedded data** | 1-2 min | Hard to know which lists contain updated entity |
| **Configuration/static data** | 1-24 hours | Very rarely changes |

### Why Short TTL for Lists?

When you update a user's name, which list caches are stale?

```
User "John" → "Johnny"

Stale caches?
├── users:org-A:page:1:...  ← Contains John?
├── users:org-A:page:2:...  ← Contains John?
├── quota:org-A:page:1:...  ← Contains John?
└── ... hundreds of keys
```

You don't know which lists contain this user. Options:
1. Clear ALL lists (defeats caching purpose)
2. Short TTL (1-2 min staleness is acceptable)
3. Don't cache lists (performance hit)

**Recommendation:** Use short TTL for lists.

---

## Common Patterns

### Pattern 1: Transactional Operations

For operations with transactions, invalidate AFTER the transaction commits:

```go
func (s *service) DeleteWidget(ctx context.Context, orgID, widgetID uuid.UUID) error {
    // 1. Transaction - use txManager.RunInTx with context-based transaction propagation
    err := s.txManager.RunInTx(ctx, func(txCtx context.Context) error {
        if err := s.widgetRepository.UnbindWidget(txCtx, widgetID); err != nil {
            return err
        }
        return s.widgetRepository.DeleteWidget(txCtx, widgetID)
    })
    if err != nil {
        return err  // Don't invalidate on failure!
    }

    // 2. Invalidate AFTER successful commit
    s.widgetRepository.InvalidateWidgetCache(ctx, widgetID, orgID)

    return nil
}
```

### Pattern 2: Deactivation Cascade Invalidation

For deactivation operations (is_active=false), use `HierarchicalCacheInvalidator`:

```go
func (s *service) DeactivateSite(ctx context.Context, orgID, siteID uuid.UUID) error {
    // Deactivate site - repository uses transaction.GetDB(ctx, r.db) internally
    if err := s.siteRepository.UpdateSite(ctx, siteID, map[string]any{"is_active": false}); err != nil {
        return err
    }

    // Deactivation cascade - affects site AND all devices in site
    invalidator := caches.NewHierarchicalCacheInvalidator(s.cacheManager, zap.S())
    invalidator.InvalidateOnSiteDeactivation(ctx, orgID.String(), siteID.String())

    return nil
}
```

### Pattern 3: Bulk Operations

For bulk operations, consider pattern-based invalidation:

```go
func (s *service) BulkUpdateWidgets(ctx context.Context, orgID uuid.UUID, widgetIDs []uuid.UUID) error {
    // Update all widgets (repository uses transaction.GetDB(ctx, r.db) internally)
    for _, id := range widgetIDs {
        s.widgetRepository.UpdateWidget(ctx, id, updates)
    }

    // Instead of invalidating each widget, invalidate all org's widget caches
    if pattern, _ := caches.BuildWidgetListByOrgPattern(orgID.String()); pattern != "" {
        s.cacheManager.InvalidateCachePattern(ctx, pattern)
    }

    return nil
}
```

---

## Troubleshooting

### Cache Not Invalidating

1. **Check pattern matches key format:**
   ```
   Key:     "widgets:details:abc-123:en"
   Pattern: "widgets:details:abc-123:*"  ✓ Matches

   Key:     "widgets:details:abc-123:en"
   Pattern: "widget:details:abc-123:*"   ✗ Missing 's'
   ```

2. **Check goroutine is running:**
   ```go
   // Add logging to verify
   go func(p string) {
       zap.S().Debugf("Invalidating pattern: %s", p)
       r.InvalidateCachePattern(context.Background(), p)
   }(pattern)
   ```

### Stale Data After Update

1. **Verify invalidation is called AFTER successful operation**
2. **Check all related patterns are invalidated** (details + list)
3. **Check TTL isn't too long** for lists

### Cache Miss Rate Too High

1. **TTL too short?** Increase for stable data
2. **Too many pattern variations?** Simplify cache key structure
3. **Over-invalidating?** Use more specific patterns

---

## Related Documentation

- [pkg/common/caches/README.md](../../../pkg/common/caches/README.md) - Detailed caching system documentation
- [Cache Invalidation Strategy](../../plans/automatic-cache-invalidation.md) - Architecture decisions
- [How to Write Repositories](how-to-write-repositories.md) - Repository patterns including caching
