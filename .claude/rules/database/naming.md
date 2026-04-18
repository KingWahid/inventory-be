---
paths:
  - "pkg/database/**"
  - "infra/database/**"
---

# Database Layer Naming

## Interface

Named `Repository` (singular). Public, exported.

```go
type Repository interface {
    GetDeviceByID(ctx context.Context, orgID, deviceID uuid.UUID) (*schemas.Device, error)
    // ...
}
```

## Implementation

Lowercase struct name matching the entity. Embeds `*base.Repository`.

```go
type deviceRepository struct {
    *base.Repository
    db *gorm.DB
}
```

## Constructor

Always `NewRepository`. Returns `(Repository, error)`.

```go
func NewRepository(db *gorm.DB, cacheManager *caches.CacheManager, cachingEnabled bool) (Repository, error)
```

## Method Naming

| Prefix | Purpose | Example |
|--------|---------|---------|
| `Get*` | Single entity retrieval | `GetDeviceByID`, `GetUserByEmail` |
| `List*` | Paginated collection | `ListOrganizations` |
| `Count*` | Count aggregation | `CountDevices` |
| `Create*` | Insert | `CreateDevice` |
| `Update*` | Modify | `UpdateDevice`, `UpdateDeviceLastSeen` |
| `Delete*` | Soft delete | `DeleteDevice` |
| `HardDelete*` | Permanent (hard) delete | `HardDeleteLicenseRole`, `HardDeleteRolePermissions` |
| `Is*` | Boolean check | `IsOrganizationActive` |
| `Invalidate*` | Cache invalidation | `InvalidateDeviceCache` |

## Organization Scoping

Default methods include `organizationID` parameter. Unscoped variants use the `Unscoped` suffix:

```go
GetAccessCardByID(ctx context.Context, organizationID, cardID uuid.UUID)        // normal
GetAccessCardByIDUnscoped(ctx context.Context, cardID uuid.UUID)                // admin only
```

## Filter and Query Structs

For repositories with complex filtering, use `Filter` and `Query` structs:

```go
type Filter struct {
    ID         *uuid.UUID
    OrgID      *uuid.UUID
    Search     *string
    IsActive   *bool
}

type Query struct {
    Filter   Filter
    Preloads []string
    Page     *int
    Limit    *int
    OrderBy  string
}
```

## FX Module (`module.go`)

```go
type Params struct {
    fx.In
    DB             *gorm.DB
    CacheManager   *caches.CacheManager
    CachingEnabled bool `name:"cachingEnabled"`
}

type Result struct {
    fx.Out
    Repository Repository
}

func Provide(params Params) (Result, error)

var Module = fx.Module("entity-repository", fx.Provide(Provide))
```
