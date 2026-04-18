---
paths:
  - "pkg/services/**"
  - "pkg/common/**"
  - "pkg/eventbus/**"
  - "workers/jobs/consumers/**"
---

# Business Logic Layer Naming

## Service Interface

Named `Service` (singular). Public, exported. Methods grouped by feature domain with section comments:

```go
type Service interface {
    // ==========================================================================
    // Fuel Tank Monitoring Quotas
    // ==========================================================================
    GetFuelTankMonitoringQuotas(ctx context.Context, ...) (*types.FuelTankMonitoringQuotaList, error)
    CreateFuelTankMonitoringQuota(ctx context.Context, ...) (*types.FuelTankMonitoringQuota, error)

    // ==========================================================================
    // Fuel Tank Monitoring Devices
    // ==========================================================================
    GetFuelTankMonitoringDevicesWithStats(ctx context.Context, ...) (*types.FuelTankMonitoringDevicesResponse, error)
}
```

## Implementation

Lowercase struct name `service`. Dependencies injected via struct fields:

```go
type service struct {
    txManager    transaction.Manager
    db           *gorm.DB
    deviceRepo   devices.Repository
    userRepo     users.Repository
}
```

## Method Naming

| Prefix | Purpose | Example |
|--------|---------|---------|
| `Get*` | Retrieve single/detail | `GetFuelTankMonitoringQuota` |
| `List*` | Paginated collection | `ListOrganizationSubscriptions` |
| `Create*` | Create with business rules | `CreateFuelTankMonitoringQuota` |
| `Update*` | Update with validation | `UpdateOrganizationSubscription` |
| `Delete*` | Delete with cascading logic | `DeleteFuelTankMonitoringQuota` |
| `Bulk*` | Multi-entity operations | `BulkGetDeviceConfigs` |
| `Trigger*` | Side-effect operations | `TriggerDeviceUserDataRefresh` |

## File Organization

One file per operation or tightly related group:

```
services/ftm/
├── SERVICE.go              # Interface definition
├── MODULE.go               # FX module
├── quotas.go               # Quota operations
├── devices.go              # Device operations
├── dashboard.go            # Dashboard operations
├── helpers.go              # Shared private helpers
└── quotas_test.go          # Tests for quota operations
```

## Domain Types (`pkg/services/types/`)

- Entities: full domain prefix — `FuelTankMonitoringQuota`, `FTMDevice`
- Request types: `Create{Entity}Request`, `Update{Entity}Request`
- Response types: `{Entity}Detail`, `{Entity}List`
- List aliases: `type {Entity}List = PaginatedList[{Entity}ListItem]`
- Enums: `type QuotaApplyFor string` with constants `QuotaApplyForToday`, `QuotaApplyForFuture`
- Optional fields use pointers: `*string`, `*float64`, `*bool`

## FX Module (`MODULE.go`)

```go
type ServiceParams struct {
    fx.In
    TxManager    transaction.Manager
    DB           *gorm.DB
    DeviceRepo   devices.Repository
    // field names match the interface type
}

type ServiceResult struct {
    fx.Out
    Service Service
}

func Provide(params ServiceParams) (ServiceResult, error)

var Module = fx.Module("ftm-service", fx.Provide(Provide))
```

## Consumer Handler Logic (`workers/jobs/consumers/`)

When biz-dev writes handler logic in the workers directory:
- Package name matches the domain: `package subscription`, `package notification`
- Handler methods: `handle{EventName}` (unexported)
- Payload structs: `{eventName}Payload` (unexported)
- Always validate and parse UUIDs from event payloads before calling services
