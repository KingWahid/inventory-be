# Context and Transaction Rules

## Context as First Parameter

Every repository and service method takes `context.Context` as the first parameter. No exceptions.

```go
func (r *repo) GetDeviceByID(ctx context.Context, deviceID uuid.UUID) (*schemas.Device, error)
func (s *svc) CreateOrganization(ctx context.Context, org *schemas.Organization) error
```

## Transaction Propagation

Transactions propagate through context — never pass `*gorm.DB` or `*sql.Tx` as a method parameter.

**In services** — attach transaction to context:
```go
ctx = transaction.WithTx(ctx, tx)
err := s.repo.CreateDevice(ctx, device)  // repo detects tx from context
```

**In repositories** — retrieve the DB/tx from context:
```go
db := transaction.GetDB(ctx, r.db)  // returns tx if in transaction, else r.db (context already applied)
err := db.Create(&record).Error
```

> **Note:** `transaction.GetDB` already applies context to the returned DB instance, so `.WithContext(ctx)` is not needed after calling it.

## GORM WithContext

Always call `.WithContext(ctx)` on GORM queries:

```go
// Correct
query := r.db.WithContext(ctx).Model(&schemas.Device{})

// Wrong — loses context (deadline, cancellation, transaction)
query := r.db.Model(&schemas.Device{})
```

## Cache Invalidation in Transactions

Inside a transaction, the caller handles cache invalidation after commit — not the repository:

```go
// Repository checks transaction context
if !transaction.InTransaction(ctx) {
    r.InvalidateDeviceCache(device.ID, device.OrganizationID)
}
```

## No Global State

Never store context, DB connections, or transactions in package-level variables. All state flows through function parameters and context.
