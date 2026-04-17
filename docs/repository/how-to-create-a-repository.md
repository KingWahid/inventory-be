# How to Create a Repository

This guide provides a step-by-step checklist for creating a new repository from scratch.

## Prerequisites

Before creating a repository, ensure:
- [ ] The database table exists (check `infra/database/migrations/*`)
- [ ] The GORM schema exists in `pkg/database/schemas/`
- [ ] Schema field types match database nullability (see [schema field types](./how-to-understand-repositories.md#schema-field-types))

## Step 1: Verify Table and Schema

### Check the Database Schema

Look at migration files to understand the table structure:

```bash
# View recent migrations
ls -la infra/database/migrations/

# Or check db.dbml for visual reference
# To regenerate (if needed):
dbdocs db2dbml postgres postgresql://industrix_user:industrix_password@localhost:5432/industrix_db -o db.dbml
```

**Warning**: Connection URL is environment-specific - adapt username, password, host, port, and database name to your local setup.

### Verify Schema Struct

Ensure `pkg/database/schemas/<entity>.go` exists and matches the table:

```go
type Invoice struct {
    ID             uuid.UUID      `gorm:"column:id;type:uuid;primaryKey;default:gen_random_uuid()"`
    OrganizationID uuid.UUID      `gorm:"column:organization_id;type:uuid;not null"`
    Number         string         `gorm:"column:number;size:50;not null"`
    Status         string         `gorm:"column:status;size:20;not null"`
    Description    *string        `gorm:"column:description;size:500"`  // Nullable
    CreatedAt      time.Time      `gorm:"column:created_at;not null;default:CURRENT_TIMESTAMP"`
    UpdatedAt      time.Time      `gorm:"column:updated_at;not null;default:CURRENT_TIMESTAMP"`
    DeletedAt      gorm.DeletedAt `gorm:"column:deleted_at;index"`
    DeletedBy      *uuid.UUID     `gorm:"column:deleted_by"`  // Nullable
}

func (Invoice) TableName() string {
    return "common.invoices"
}
```

**Field type rules:**
- Pointer types (`*string`, `*uuid.UUID`) = Nullable (no `not null` in tag)
- Non-pointer types (`string`, `uuid.UUID`) = NOT NULL (has `not null` in tag)

## Step 2: Create the Repository Package

Create a new directory `pkg/database/repositories/invoices/` with `repo.go`:

```go
// Package invoices provides data access layer for invoice records.
package invoices

import (
    "context"   // Request-scoped context for deadlines, cancellation
    "errors"    // Error helpers (gorm.ErrRecordNotFound, context.DeadlineExceeded)
    "net/http"  // HTTP status codes (metadata in CustomError)

    "github.com/google/uuid" // UUID type for DB IDs (project convention)
    "go.uber.org/zap"        // Structured logger
    "gorm.io/gorm"           // GORM ORM for queries

    "github.com/industrix-id/backend/pkg/common"            // CustomError, PaginationInfo
    "github.com/industrix-id/backend/pkg/common/caches"     // Cache key builders
    "github.com/industrix-id/backend/pkg/common/errorcodes" // Error code definitions
    "github.com/industrix-id/backend/pkg/common/utils"      // Shared utilities
    "github.com/industrix-id/backend/pkg/database/base"     // Base repository with cache helpers
    "github.com/industrix-id/backend/pkg/database/db_utils" // Database error helpers
    "github.com/industrix-id/backend/pkg/database/schemas"  // GORM models
    "github.com/industrix-id/backend/pkg/database/transaction" // Transaction context
)
```

### Define Package-Level Variables

```go
// Package-level named logger for invoice repository.
var logger = zap.S().Named("repositories.invoices")

// Package-level table name for reuse across methods.
var invoiceTableName = (&schemas.Invoice{}).TableName()
```

## Step 3: Define the Interface

Design the interface around how services will consume this repository.

**Note**: Interface is named `Repository` (not `InvoiceRepository`) since the package name (`invoices`) provides context.

```go
// Repository interface for invoice data access.
type Repository interface {
    // READ operations
    GetInvoiceByID(
        ctx context.Context,      // Request-scoped context
        invoiceID uuid.UUID,      // DB identifier
    ) (*schemas.Invoice, error)   // Returns GORM model + CustomError

    ListInvoices(
        ctx context.Context,
        orgID uuid.UUID,                    // Organization scoping
        page, limit *int,                   // Pagination (nil = defaults)
        status *string,                     // Optional filter
    ) ([]schemas.Invoice, *common.PaginationInfo, error)

    // WRITE operations (use transaction.GetDB(ctx, r.db) internally)
    CreateInvoice(
        ctx context.Context,
        invoice *schemas.Invoice, // Pointer for GORM to populate IDs
    ) error

    UpdateInvoice(
        ctx context.Context,
        invoiceID uuid.UUID,
        updates map[string]any,
    ) error

    DeleteInvoice(
        ctx context.Context,
        invoiceID uuid.UUID,
        deletedBy *uuid.UUID,
    ) error

    // CACHE invalidation (called by service layer)
    InvalidateInvoiceCache(invoiceID uuid.UUID)
    InvalidateInvoicesListCache(orgID uuid.UUID)
}
```

**Pattern rules:**
- Interface is named `Repository` (package name provides entity context)
- `ctx context.Context` is always the first parameter
- Use `uuid.UUID` for IDs (except where tables use strings)
- Pointers vs values:
  - `*schemas.Invoice` when GORM mutates the struct
  - `[]schemas.Invoice` for read-only slices
- Write methods use `transaction.GetDB(ctx, r.db)` internally (no `tx *gorm.DB` parameter)

## Step 4: Implement the Struct and Constructor

```go
type invoiceRepository struct {
    *base.Repository
    db *gorm.DB
}

// NewRepository creates a new invoice repository instance.
func NewRepository(db *gorm.DB, cacheManager *caches.CacheManager, cachingEnabled bool) (Repository, error) {
    if db == nil {
        return nil, common.NewCustomError("Database connection is nil").
            WithMessageID("error_db_connection_nil").
            WithErrorCode(errorcodes.InitializationError).
            WithHTTPCode(http.StatusInternalServerError)
    }

    return &invoiceRepository{
        Repository: base.NewRepository(cacheManager, cachingEnabled),
        db:         db,
    }, nil
}
```

**What this gives you:**
- Embeds `*base.Repository` for cache helpers (from `pkg/database/base`)
- Holds `*gorm.DB` for queries
- Validates `db != nil` with standard error pattern
- Constructor is named `NewRepository` (package provides context)

## Step 5: Implement READ Methods

### Single Record with Caching

```go
func (r *invoiceRepository) GetInvoiceByID(ctx context.Context, invoiceID uuid.UUID) (*schemas.Invoice, error) {
    var invoice schemas.Invoice

    // Build cache key
    cacheKey, err := caches.BuildInvoiceByIDCacheKey(invoiceID.String())
    if err != nil {
        return nil, NewDatabaseError("Failed to build cache key", "error_build_cache_key_failed")
    }

    // Cache-aside pattern
    err = r.GetFromCacheOrDB(
        ctx,
        cacheKey,
        caches.InvoiceCacheTTL,
        &invoice,
        func() error {
            err := r.db.WithContext(ctx).
                Where("id = ? AND deleted_at IS NULL", invoiceID).
                First(&invoice).Error

            if err != nil {
                return HandleFindError(err, invoiceTableName,
                    "Invoice not found",
                    "error_invoice_not_found",
                    "error_retrieve_invoice_failed")
            }
            return nil
        },
    )

    if err != nil {
        return nil, err
    }

    return &invoice, nil
}
```

### List with Pagination and Caching

```go
func (r *invoiceRepository) ListInvoices(
    ctx context.Context,
    orgID uuid.UUID,
    page, limit *int,
    status *string,
) ([]schemas.Invoice, *common.PaginationInfo, error) {
    // Get pagination values with offset
    p, l, offset := utils.GetPaginationValues(page, limit)

    // Handle optional filter
    statusFilter := utils.ToValue(status)

    // Build cache key
    cacheKey, err := caches.BuildInvoicesListCacheKey(orgID.String(), p, l, statusFilter)
    if err != nil {
        return nil, nil, NewDatabaseError("Failed to build cache key", "error_build_cache_key_failed")
    }

    // Method-level struct for caching composite results
    type cachedResult struct {
        Invoices []schemas.Invoice
        Total    int64
    }
    var result cachedResult

    err = r.GetFromCacheOrDB(ctx, cacheKey, caches.InvoicesListCacheTTL, &result, func() error {
        // Base query
        query := r.db.WithContext(ctx).
            Model(&schemas.Invoice{}).
            Where("organization_id = ? AND deleted_at IS NULL", orgID)

        // Apply optional filter
        if statusFilter != "" {
            query = query.Where("status = ?", statusFilter)
        }

        // Count total
        if err := ExecuteCountQuery(query, &result.Total, invoiceTableName); err != nil {
            return err
        }

        // Fetch paginated data
        return ExecutePaginatedQuery(
            query,
            "created_at DESC",
            offset,
            l,
            &result.Invoices,
            invoiceTableName,
        )
    })

    if err != nil {
        return nil, nil, err
    }

    return result.Invoices, common.NewPaginationInfo(p, l, result.Total), nil
}
```

**Key patterns:**
- `utils.GetPaginationValues` returns page, limit, and offset
- `utils.ToValue` safely dereferences optional parameters
- Method-level `cachedResult` struct (not package-level)
- `ExecuteCountQuery` and `ExecutePaginatedQuery` from query helpers
- `common.NewPaginationInfo` handles int64 → int conversion

## Step 6: Implement WRITE Methods

### Create Operation

```go
import "github.com/industrix-id/backend/pkg/database/transaction"

func (r *invoiceRepository) CreateInvoice(ctx context.Context, invoice *schemas.Invoice) error {
    db := transaction.GetDB(ctx, r.db)

    if err := db.Create(invoice).Error; err != nil {
        return r.handleCreateInvoiceError(err, invoice)
    }
    return nil
}

// handleCreateInvoiceError handles Create errors with context-specific messages.
func (r *invoiceRepository) handleCreateInvoiceError(err error, invoice *schemas.Invoice) error {
    if errors.Is(err, context.DeadlineExceeded) {
        return NewDeadlineExceededError()
    }

    if dbErr := ClassifyDBError(err); dbErr != nil {
        switch dbErr.Type {
        case DBErrorDuplicate:
            zap.S().With(zap.Error(err), zap.String("invoice_number", invoice.Number)).
                Warn("Duplicate invoice number")
            return NewDuplicateError(
                "An invoice with this number already exists",
                "error_invoice_duplicate")

        case DBErrorForeignKey:
            zap.S().With(zap.Error(err), zap.String("organization_id", invoice.OrganizationID.String())).
                Warn("Organization not found for invoice")
            return NewForeignKeyError(
                "The selected organization does not exist",
                "error_invoice_organization_not_found")

        case DBErrorNotNull:
            zap.S().With(zap.Error(err), zap.String("column", dbErr.Column)).
                Warn("Required field missing for invoice")
            return NewInvalidRequestError(
                "A required field is missing",
                "error_invoice_required_field_missing")

        case DBErrorSerializationFailure, DBErrorDeadlock:
            zap.S().With(zap.Error(err)).Warn("Transaction conflict")
            return NewDatabaseError("Please try again", "error_transaction_conflict")
        }
    }

    return NewDatabaseError("Failed to create invoice", "error_create_invoice_failed")
}
```

### Update Operation

```go
func (r *invoiceRepository) UpdateInvoice(
    ctx context.Context,
    invoiceID uuid.UUID,
    updates map[string]any,
) error {
    db := transaction.GetDB(ctx, r.db)

    result := db.Model(&schemas.Invoice{}).
        Where("id = ? AND deleted_at IS NULL", invoiceID).
        Updates(updates)

    if result.Error != nil {
        return r.handleUpdateInvoiceError(result.Error)
    }

    if result.RowsAffected == 0 {
        return NewNotFoundError("Invoice not found", "error_invoice_not_found")
    }

    return nil
}

// handleUpdateInvoiceError handles Update errors.
func (r *invoiceRepository) handleUpdateInvoiceError(err error) error {
    if errors.Is(err, context.DeadlineExceeded) {
        return NewDeadlineExceededError()
    }

    if dbErr := ClassifyDBError(err); dbErr != nil {
        switch dbErr.Type {
        case DBErrorDataTooLong:
            zap.S().With(zap.Error(err), zap.String("column", dbErr.Column)).
                Warn("Data too long when updating invoice")
            return NewInvalidRequestError(
                "The provided value is too long",
                "error_invoice_data_too_long")

        case DBErrorSerializationFailure, DBErrorDeadlock:
            zap.S().With(zap.Error(err)).Warn("Transaction conflict")
            return NewDatabaseError("Please try again", "error_transaction_conflict")
        }
    }

    return NewDatabaseError("Failed to update invoice", "error_update_invoice_failed")
}
```

### Delete Operation

```go
func (r *invoiceRepository) DeleteInvoice(
    ctx context.Context,
    invoiceID uuid.UUID,
    deletedBy *uuid.UUID,
) error {
    err := r.Delete(
        ctx,
        r.db,
        &schemas.Invoice{},
        deletedBy,
        "id = ? AND deleted_at IS NULL",
        invoiceID,
    )

    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return NewNotFoundError("Invoice not found", "error_invoice_not_found")
        }
        zap.S().With(zap.Error(err)).Error("Failed to delete invoice")
        return NewDatabaseError("Failed to delete invoice", "error_delete_invoice_failed")
    }

    return nil
}
```

**Note**: If the entity has `is_active`, service layer must check deactivation before calling delete. See [how-to-handle-transactions.md](./how-to-handle-transactions.md#deactivation-before-deletion).

## Step 7: Implement Cache Invalidation

```go
// InvalidateInvoiceCache invalidates caches affected by invoice changes.
//
// Invalidates:
//   - Individual invoice cache (by invoiceID)
//
// Cache invalidation runs in goroutines with copied parameters to avoid blocking.
func (r *invoiceRepository) InvalidateInvoiceCache(invoiceID uuid.UUID) {
    if !r.cachingEnabled || r.cacheManager == nil {
        return
    }

    idStr := invoiceID.String()

    if cacheKey, err := caches.BuildInvoiceByIDCacheKey(idStr); err == nil {
        go func(key string) {
            r.InvalidateCache(context.Background(), key)
        }(cacheKey)
    }
}

// InvalidateInvoicesListCache invalidates the invoices list cache pattern.
//
// Invalidates:
//   - All paginated list caches for the organization
//
// Cache invalidation runs in goroutines to avoid blocking.
func (r *invoiceRepository) InvalidateInvoicesListCache(orgID uuid.UUID) {
    if !r.cachingEnabled || r.cacheManager == nil {
        return
    }

    orgIDStr := orgID.String()

    if pattern, err := caches.BuildInvoicesListCachePattern(orgIDStr); err == nil {
        go func(p string) {
            r.InvalidateCachePattern(context.Background(), p)
        }(pattern)
    }
}
```

## Step 8: Add Cache Key Builders

Create `pkg/common/caches/invoices.go`:

```go
package caches

import (
    "bytes"
    "fmt"
    "text/template"
    "time"
)

const InvoiceCacheTTL = 15 * time.Minute
const InvoicesListCacheTTL = 5 * time.Minute

var invoiceByIDTmpl = template.Must(template.New("invoiceByID").Parse(
    "invoice:id:{{.invoiceID}}",
))

var invoicesListTmpl = template.Must(template.New("invoicesList").Parse(
    "invoices:list:org:{{.orgID}}:p:{{.page}}:l:{{.limit}}:status:{{.status}}",
))

// BuildInvoiceByIDCacheKey creates a cache key for invoice lookup by ID.
func BuildInvoiceByIDCacheKey(invoiceID string) (string, error) {
    var buf bytes.Buffer
    if err := invoiceByIDTmpl.Execute(&buf, map[string]any{"invoiceID": invoiceID}); err != nil {
        return "", err
    }
    return buf.String(), nil
}

// BuildInvoicesListCacheKey creates a cache key for invoices list with pagination.
func BuildInvoicesListCacheKey(orgID string, page, limit int, status string) (string, error) {
    var buf bytes.Buffer
    if err := invoicesListTmpl.Execute(&buf, map[string]any{
        "orgID":  orgID,
        "page":   page,
        "limit":  limit,
        "status": status,
    }); err != nil {
        return "", err
    }
    return buf.String(), nil
}

// BuildInvoicesListCachePattern creates a pattern to invalidate all invoice list caches.
func BuildInvoicesListCachePattern(orgID string) (string, error) {
    return fmt.Sprintf("invoices:list:org:%s:*", orgID), nil
}
```

**Cache key conventions:**
- Use underscores (`_`) in prefixes (e.g., `device_config`, not `device-config`)
- Use colons (`:`) to separate segments
- Tightly scoped patterns (not `*invoice*`)

## Step 9: Create FX Module

Create `pkg/database/repositories/invoices/MODULE.go` for dependency injection:

```go
// Package invoices provides FX module for invoice repository.
package invoices

import (
    "go.uber.org/fx"
    "gorm.io/gorm"

    "github.com/industrix-id/backend/pkg/common/caches"
)

// Params holds the parameters needed for Repository creation.
type Params struct {
    fx.In

    DB             *gorm.DB
    CacheManager   *caches.CacheManager
    CachingEnabled bool `name:"cachingEnabled"`
}

// Result holds the Repository provided by this module.
type Result struct {
    fx.Out

    Repository Repository
}

// Provide creates a Repository instance for FX dependency injection.
func Provide(params Params) (Result, error) {
    logger.Debug("Creating Repository")

    repo, err := NewRepository(params.DB, params.CacheManager, params.CachingEnabled)
    if err != nil {
        return Result{}, err
    }

    logger.Info("Repository created successfully")
    return Result{Repository: repo}, nil
}

// Module provides Repository as an FX module.
var Module = fx.Module("invoice-repository",
    fx.Provide(Provide),
)
```

## Step 10: Create Tests

### Unit Test (`repo_test.go`)

```go
//go:build !integration
// +build !integration

package invoices

import (
    "context"
    "testing"

    "github.com/stretchr/testify/suite"

    "github.com/industrix-id/backend/pkg/database/schemas"
)

type InvoiceRepositoryTestSuite struct {
    suite.Suite
    // ... test fields
}

func TestInvoiceRepositoryTestSuite(t *testing.T) {
    suite.Run(t, new(InvoiceRepositoryTestSuite))
}
```

### Integration Test (`repo_integration_test.go`)

```go
//go:build integration || integration_all
// +build integration integration_all

package invoices

import (
    "context"
    "testing"

    "github.com/stretchr/testify/suite"

    "github.com/industrix-id/backend/pkg/database/schemas"
)

type InvoiceRepositoryIntegrationTestSuite struct {
    suite.Suite
    // ... test fields
}

func TestInvoiceRepositoryIntegrationTestSuite(t *testing.T) {
    suite.Run(t, new(InvoiceRepositoryIntegrationTestSuite))
}
```

## Step 11: Generate Mock

```bash
# Generate all repository mocks (recommended)
make mocks-generate
```

This generates `pkg/database/repositories/invoices/mocks/Repository.go`.

## Checklist

- [ ] Verified table exists in migrations
- [ ] Verified schema exists with correct field types
- [ ] Created repository package: `pkg/database/repositories/invoices/`
- [ ] Created `repo.go` with standard imports and named logger
- [ ] Defined package-level table name
- [ ] Defined `Repository` interface with all methods
- [ ] Implemented struct and constructor (`NewRepository`)
- [ ] Implemented read methods with caching
- [ ] Implemented write methods with error handlers
- [ ] Implemented cache invalidation methods
- [ ] Created cache key builders in `pkg/common/caches`
- [ ] Created FX module (`MODULE.go`)
- [ ] Created unit tests (`repo_test.go`)
- [ ] Created integration tests (`repo_integration_test.go`)
- [ ] Generated mock: `make mocks-generate`
- [ ] Verified build passes: `go build ./pkg/database/repositories/invoices/...`

## Next Steps

- [How to Implement Queries](./how-to-implement-queries.md) - Advanced query patterns
- [How to Handle Errors](./how-to-handle-errors.md) - Error handling deep dive
- [How to Invalidate Caches](./how-to-invalidate-caches.md) - Cache patterns
