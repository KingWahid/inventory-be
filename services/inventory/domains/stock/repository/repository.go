// Package repository implements stock_balances access. All mutating reads use SELECT … FOR UPDATE;
// callers must wrap operations in transaction.RunInTx and pass ctx so transaction.GetDB propagates the *gorm.DB transaction.
package repository

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"

	"github.com/KingWahid/inventory/backend/pkg/common/errorcodes"
	"github.com/KingWahid/inventory/backend/pkg/database/transaction"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Repository defines locked reads and balance updates for stock_balances.
type Repository interface {
	Ping() error
	GetForUpdate(ctx context.Context, tenantID, warehouseID, productID string) (StockBalance, error)
	// EnsureBalanceRow inserts a zero row if missing (ON CONFLICT DO NOTHING), then returns the row locked — for first inbound / tests.
	EnsureBalanceRow(ctx context.Context, tenantID, warehouseID, productID string) (StockBalance, error)
	ApplyDelta(ctx context.Context, tenantID, warehouseID, productID string, delta int64) error
	// TransferDelta moves qty from src warehouse to dst for one product; locks the two rows in deterministic order.
	TransferDelta(ctx context.Context, tenantID, srcWarehouseID, dstWarehouseID, productID string, qty int64) error
}

type repository struct {
	db *gorm.DB
}

// New creates a stock balances repository.
func New(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Ping() error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

// GetForUpdate returns the balance row locked for update. Must be called inside transaction.RunInTx.
func (r *repository) GetForUpdate(ctx context.Context, tenantID, warehouseID, productID string) (StockBalance, error) {
	row, err := r.getBalanceRowForUpdate(ctx, tenantID, warehouseID, productID)
	if err != nil {
		return StockBalance{}, err
	}
	return rowToStockBalance(row), nil
}

func (r *repository) EnsureBalanceRow(ctx context.Context, tenantID, warehouseID, productID string) (StockBalance, error) {
	tx := transaction.GetDB(ctx, r.db).WithContext(ctx)
	err := tx.Exec(`
		INSERT INTO stock_balances (tenant_id, warehouse_id, product_id)
		VALUES (?::uuid, ?::uuid, ?::uuid)
		ON CONFLICT ON CONSTRAINT uq_stock_tenant_warehouse_product DO NOTHING`,
		strings.TrimSpace(tenantID), strings.TrimSpace(warehouseID), strings.TrimSpace(productID),
	).Error
	if err != nil {
		return StockBalance{}, err
	}
	row, err := r.getBalanceRowForUpdate(ctx, tenantID, warehouseID, productID)
	if err != nil {
		return StockBalance{}, err
	}
	return rowToStockBalance(row), nil
}

// ApplyDelta loads the row with FOR UPDATE then sets quantity += delta. Fails with ErrInsufficient if result would be negative.
func (r *repository) ApplyDelta(ctx context.Context, tenantID, warehouseID, productID string, delta int64) error {
	row, err := r.getBalanceRowForUpdate(ctx, tenantID, warehouseID, productID)
	if err != nil {
		return err
	}
	newQty := int64(row.Quantity) + delta
	return r.applyQuantity(ctx, row, newQty)
}

func (r *repository) TransferDelta(ctx context.Context, tenantID, srcWarehouseID, dstWarehouseID, productID string, qty int64) error {
	if qty <= 0 {
		return errorcodes.ErrValidationError.WithDetails(map[string]any{"message": "transfer quantity must be positive"})
	}
	srcWH := strings.TrimSpace(srcWarehouseID)
	dstWH := strings.TrimSpace(dstWarehouseID)
	if srcWH == dstWH {
		return nil
	}

	kSrc := WarehouseProductKey{WarehouseID: srcWH, ProductID: strings.TrimSpace(productID)}
	kDst := WarehouseProductKey{WarehouseID: dstWH, ProductID: kSrc.ProductID}
	first, second := OrderedWarehouseProduct(kSrc, kDst)

	r1, err := r.getBalanceRowForUpdate(ctx, tenantID, first.WarehouseID, first.ProductID)
	if err != nil {
		return err
	}
	r2, err := r.getBalanceRowForUpdate(ctx, tenantID, second.WarehouseID, second.ProductID)
	if err != nil {
		return err
	}

	var srcRow, dstRow stockBalanceRow
	switch {
	case r1.WarehouseID == srcWH && r1.ProductID == kSrc.ProductID:
		srcRow = r1
		dstRow = r2
	case r2.WarehouseID == srcWH && r2.ProductID == kSrc.ProductID:
		srcRow = r2
		dstRow = r1
	default:
		return errorcodes.ErrInternal.WithDetails(map[string]any{"message": "stock: transfer lock mismatch"})
	}

	newSrc := int64(srcRow.Quantity) - qty
	newDst := int64(dstRow.Quantity) + qty
	if err := r.applyQuantity(ctx, srcRow, newSrc); err != nil {
		return err
	}
	return r.applyQuantity(ctx, dstRow, newDst)
}

func (r *repository) getBalanceRowForUpdate(ctx context.Context, tenantID, warehouseID, productID string) (stockBalanceRow, error) {
	tid := strings.TrimSpace(tenantID)
	wid := strings.TrimSpace(warehouseID)
	pid := strings.TrimSpace(productID)

	db := transaction.GetDB(ctx, r.db).WithContext(ctx)
	var row stockBalanceRow
	err := db.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("tenant_id = ? AND warehouse_id = ? AND product_id = ?", tid, wid, pid).
		First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return stockBalanceRow{}, errorcodes.ErrNotFound
		}
		return stockBalanceRow{}, err
	}
	return row, nil
}

func (r *repository) applyQuantity(ctx context.Context, row stockBalanceRow, newQty int64) error {
	if newQty < 0 {
		return errorcodes.ErrInsufficient
	}
	if newQty > math.MaxInt32 {
		return errorcodes.ErrValidationError.WithDetails(map[string]any{"message": "quantity overflow"})
	}

	db := transaction.GetDB(ctx, r.db).WithContext(ctx)
	now := time.Now().UTC()
	qty32 := int32(newQty)
	res := db.Model(&stockBalanceRow{}).
		Where("id = ?", row.ID).
		Updates(map[string]any{
			"quantity":   qty32,
			"updated_at": now,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected != 1 {
		return errorcodes.ErrNotFound
	}
	return nil
}
