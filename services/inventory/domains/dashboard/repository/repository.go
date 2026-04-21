// Package repository runs aggregate queries for the tenant dashboard (ARCHITECTURE §9).
// Heavy aggregates may later move to materialized views; v1 uses indexed SQL + application cache (plan 7.2).
package repository

import (
	"context"

	"gorm.io/gorm"
)

// Summary is non-sensitive aggregate counts for one tenant.
type Summary struct {
	TotalProducts   int64 `gorm:"column:total_products"`
	TotalWarehouses int64 `gorm:"column:total_warehouses"`
	MovementsToday  int64 `gorm:"column:movements_today"`
	LowStockCount   int64 `gorm:"column:low_stock_count"`
}

// StorageUtilizationRow is one warehouse utilization snapshot row.
type StorageUtilizationRow struct {
	WarehouseID   string `gorm:"column:warehouse_id"`
	WarehouseCode string `gorm:"column:warehouse_code"`
	WarehouseName string `gorm:"column:warehouse_name"`
	OnHandQty     int64  `gorm:"column:on_hand_qty"`
	Percent       int32  `gorm:"column:percent"`
}

// Repository loads dashboard aggregates.
type Repository interface {
	GetDashboardSummary(ctx context.Context, tenantID string) (Summary, error)
	GetMovementChart(ctx context.Context, tenantID string, period MovementChartPeriod) ([]MovementChartPoint, error)
	GetStorageUtilization(ctx context.Context, tenantID string, limit int) ([]StorageUtilizationRow, error)
}

type repo struct {
	db *gorm.DB
}

// New wires dashboard aggregates on the shared DB handle.
func New(db *gorm.DB) Repository {
	return &repo{db: db}
}

func (r *repo) GetDashboardSummary(ctx context.Context, tenantID string) (Summary, error) {
	var out Summary
	// UTC day boundary for "movements today"; low stock = total on-hand across warehouses below product reorder_level.
	err := r.db.WithContext(ctx).Raw(`
SELECT
  (SELECT COUNT(*)::bigint FROM products WHERE tenant_id = ?::uuid AND deleted_at IS NULL) AS total_products,
  (SELECT COUNT(*)::bigint FROM warehouses WHERE tenant_id = ?::uuid AND deleted_at IS NULL AND is_active) AS total_warehouses,
  (SELECT COUNT(*)::bigint FROM movements
     WHERE tenant_id = ?::uuid AND status = 'confirmed'
       AND (updated_at AT TIME ZONE 'UTC')::date = (timezone('UTC', now()))::date) AS movements_today,
  (SELECT COUNT(*)::bigint FROM products p
     LEFT JOIN (
       SELECT product_id, SUM(quantity)::bigint AS qty FROM stock_balances WHERE tenant_id = ?::uuid GROUP BY product_id
     ) s ON s.product_id = p.id
     WHERE p.tenant_id = ?::uuid AND p.deleted_at IS NULL AND COALESCE(s.qty, 0) < p.reorder_level) AS low_stock_count
`, tenantID, tenantID, tenantID, tenantID, tenantID).Scan(&out).Error
	return out, err
}

func (r *repo) GetStorageUtilization(ctx context.Context, tenantID string, limit int) ([]StorageUtilizationRow, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	rows := make([]StorageUtilizationRow, 0)
	err := r.db.WithContext(ctx).Raw(`
WITH wh_qty AS (
  SELECT
    w.id AS warehouse_id,
    w.code AS warehouse_code,
    w.name AS warehouse_name,
    COALESCE(SUM(sb.quantity), 0)::bigint AS on_hand_qty
  FROM warehouses w
  LEFT JOIN stock_balances sb
    ON sb.tenant_id = w.tenant_id
   AND sb.warehouse_id = w.id
  WHERE w.tenant_id = ?::uuid
    AND w.deleted_at IS NULL
    AND w.is_active
  GROUP BY w.id, w.code, w.name
)
SELECT
  warehouse_id,
  warehouse_code,
  warehouse_name,
  on_hand_qty,
  CASE
    WHEN MAX(on_hand_qty) OVER () <= 0 THEN 0
    ELSE ROUND((on_hand_qty::numeric / NULLIF(MAX(on_hand_qty) OVER (), 0)::numeric) * 100)::int
  END AS percent
FROM wh_qty
ORDER BY on_hand_qty DESC, warehouse_name ASC
LIMIT ?;
`, tenantID, limit).Scan(&rows).Error
	return rows, err
}
