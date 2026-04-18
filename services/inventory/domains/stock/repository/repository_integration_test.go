//go:build integration || integration_all

// Set TEST_DATABASE_URL to a PostgreSQL DSN (e.g. postgres://user:pass@localhost:5432/dbname?sslmode=disable).

package repository_test

import (
	"context"
	"errors"
	"os"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/KingWahid/inventory/backend/pkg/common/errorcodes"
	"github.com/KingWahid/inventory/backend/pkg/database/transaction"
	stockrepo "github.com/KingWahid/inventory/backend/services/inventory/domains/stock/repository"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func openIntegrationDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TEST_DATABASE_URL for integration tests (PostgreSQL)")
	}
	gdb, err := gorm.Open(gormpostgres.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
	})
	if err != nil {
		t.Fatalf("gorm open: %v", err)
	}
	return gdb
}

type seedIDs struct {
	TenantID    string
	WarehouseA  string
	WarehouseB  string
	ProductID   string
}

func seedTenantWarehousesProduct(t *testing.T, gdb *gorm.DB) seedIDs {
	t.Helper()
	ctx := context.Background()
	tenantID := uuid.New().String()
	wa := uuid.New().String()
	wb := uuid.New().String()
	pid := uuid.New().String()

	if err := gdb.WithContext(ctx).Exec(
		`INSERT INTO tenants (id, name) VALUES (?::uuid, ?)`,
		tenantID, "stock-int-test-"+tenantID[:8],
	).Error; err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
	if err := gdb.WithContext(ctx).Exec(
		`INSERT INTO warehouses (id, tenant_id, code, name) VALUES (?::uuid, ?::uuid, ?, ?)`,
		wa, tenantID, "WA"+wa[:8], "Warehouse A",
	).Error; err != nil {
		t.Fatalf("seed wh A: %v", err)
	}
	if err := gdb.WithContext(ctx).Exec(
		`INSERT INTO warehouses (id, tenant_id, code, name) VALUES (?::uuid, ?::uuid, ?, ?)`,
		wb, tenantID, "WB"+wb[:8], "Warehouse B",
	).Error; err != nil {
		t.Fatalf("seed wh B: %v", err)
	}
	if err := gdb.WithContext(ctx).Exec(
		`INSERT INTO products (id, tenant_id, sku, name) VALUES (?::uuid, ?::uuid, ?, ?)`,
		pid, tenantID, "SKU-"+pid[:8], "Product",
	).Error; err != nil {
		t.Fatalf("seed product: %v", err)
	}

	t.Cleanup(func() {
		_ = gdb.WithContext(context.Background()).Exec(`DELETE FROM tenants WHERE id = ?::uuid`, tenantID).Error
	})

	return seedIDs{TenantID: tenantID, WarehouseA: wa, WarehouseB: wb, ProductID: pid}
}

func TestConcurrentOutbound_ApplyDelta(t *testing.T) {
	gdb := openIntegrationDB(t)
	ids := seedTenantWarehousesProduct(t, gdb)
	repo := stockrepo.New(gdb)

	const initial int32 = 25
	const goroutines = 40

	err := gdb.WithContext(context.Background()).Exec(
		`INSERT INTO stock_balances (tenant_id, warehouse_id, product_id, quantity)
		 VALUES (?::uuid, ?::uuid, ?::uuid, ?)`,
		ids.TenantID, ids.WarehouseA, ids.ProductID, initial,
	).Error
	if err != nil {
		t.Fatalf("seed balance: %v", err)
	}

	var successes int64
	var insufficient int64
	var otherErr atomic.Value

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			err := transaction.RunInTx(context.Background(), gdb, func(ctx context.Context) error {
				return repo.ApplyDelta(ctx, ids.TenantID, ids.WarehouseA, ids.ProductID, -1)
			})
			if err == nil {
				atomic.AddInt64(&successes, 1)
				return
			}
			if errors.Is(err, errorcodes.ErrInsufficient) {
				atomic.AddInt64(&insufficient, 1)
				return
			}
			otherErr.Store(err)
		}()
	}
	wg.Wait()

	if v := otherErr.Load(); v != nil {
		t.Fatalf("unexpected error: %v", v)
	}
	if successes != int64(initial) {
		t.Fatalf("successes want %d got %d", initial, successes)
	}
	if insufficient != int64(goroutines)-int64(initial) {
		t.Fatalf("insufficient want %d got %d", goroutines-int(initial), insufficient)
	}

	var qty int32
	row := gdb.WithContext(context.Background()).Raw(
		`SELECT quantity FROM stock_balances WHERE tenant_id = ?::uuid AND warehouse_id = ?::uuid AND product_id = ?::uuid`,
		ids.TenantID, ids.WarehouseA, ids.ProductID,
	).Scan(&qty)
	if row.Error != nil {
		t.Fatal(row.Error)
	}
	if qty != 0 {
		t.Fatalf("final quantity want 0 got %d", qty)
	}
}

func TestConcurrentTransfer_ConservesTotal(t *testing.T) {
	gdb := openIntegrationDB(t)
	ids := seedTenantWarehousesProduct(t, gdb)
	repo := stockrepo.New(gdb)

	const initialA int32 = 100
	const transfers = 80

	err := gdb.WithContext(context.Background()).Exec(
		`INSERT INTO stock_balances (tenant_id, warehouse_id, product_id, quantity)
		 VALUES (?::uuid, ?::uuid, ?::uuid, ?), (?::uuid, ?::uuid, ?::uuid, 0)`,
		ids.TenantID, ids.WarehouseA, ids.ProductID, initialA,
		ids.TenantID, ids.WarehouseB, ids.ProductID,
	).Error
	if err != nil {
		t.Fatalf("seed balances: %v", err)
	}

	var firstErr atomic.Value // error
	var wg sync.WaitGroup
	wg.Add(transfers)
	for i := 0; i < transfers; i++ {
		go func() {
			defer wg.Done()
			e := transaction.RunInTx(context.Background(), gdb, func(ctx context.Context) error {
				return repo.TransferDelta(ctx, ids.TenantID, ids.WarehouseA, ids.WarehouseB, ids.ProductID, 1)
			})
			if e != nil && firstErr.Load() == nil {
				firstErr.Store(e)
			}
		}()
	}
	wg.Wait()

	if e := firstErr.Load(); e != nil {
		t.Fatalf("transfer error: %v", e.(error))
	}

	var qa, qb int32
	if err := gdb.WithContext(context.Background()).Raw(
		`SELECT quantity FROM stock_balances WHERE tenant_id = ?::uuid AND warehouse_id = ?::uuid AND product_id = ?::uuid`,
		ids.TenantID, ids.WarehouseA, ids.ProductID,
	).Scan(&qa).Error; err != nil {
		t.Fatal(err)
	}
	if err := gdb.WithContext(context.Background()).Raw(
		`SELECT quantity FROM stock_balances WHERE tenant_id = ?::uuid AND warehouse_id = ?::uuid AND product_id = ?::uuid`,
		ids.TenantID, ids.WarehouseB, ids.ProductID,
	).Scan(&qb).Error; err != nil {
		t.Fatal(err)
	}
	if qa != initialA-transfers {
		t.Fatalf("warehouse A quantity want %d got %d", initialA-transfers, qa)
	}
	if qb != transfers {
		t.Fatalf("warehouse B quantity want %d got %d", transfers, qb)
	}
	if qa+qb != initialA {
		t.Fatalf("conservation failed: %d + %d != %d", qa, qb, initialA)
	}
}
