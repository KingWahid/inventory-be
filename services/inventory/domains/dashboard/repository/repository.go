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

// Repository loads dashboard aggregates.
type Repository interface {
	GetDashboardSummary(ctx context.Context, tenantID string) (Summary, error)
	GetMovementChart(ctx context.Context, tenantID string, period MovementChartPeriod) ([]MovementChartPoint, error)
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
