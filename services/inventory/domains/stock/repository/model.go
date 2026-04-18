package repository

import "time"

// StockBalance is a tenant-scoped on-hand quantity for one product at one warehouse.
type StockBalance struct {
	ID               string
	TenantID         string
	WarehouseID      string
	ProductID        string
	Quantity         int32
	ReservedQuantity int32
	LastMovementAt   *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// stockBalanceRow maps to table stock_balances. Callers must scope work with transaction.RunInTx; this repo uses transaction.GetDB(ctx).
type stockBalanceRow struct {
	ID               string     `gorm:"column:id;type:uuid;primaryKey"`
	TenantID         string     `gorm:"column:tenant_id;type:uuid"`
	WarehouseID      string     `gorm:"column:warehouse_id;type:uuid"`
	ProductID        string     `gorm:"column:product_id;type:uuid"`
	Quantity         int32      `gorm:"column:quantity"`
	ReservedQuantity int32      `gorm:"column:reserved_quantity"`
	LastMovementAt   *time.Time `gorm:"column:last_movement_at"`
	CreatedAt        time.Time  `gorm:"column:created_at"`
	UpdatedAt        time.Time  `gorm:"column:updated_at"`
}

func (stockBalanceRow) TableName() string { return "stock_balances" }

func rowToStockBalance(m stockBalanceRow) StockBalance {
	return StockBalance{
		ID:               m.ID,
		TenantID:         m.TenantID,
		WarehouseID:      m.WarehouseID,
		ProductID:        m.ProductID,
		Quantity:         m.Quantity,
		ReservedQuantity: m.ReservedQuantity,
		LastMovementAt:   m.LastMovementAt,
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
	}
}
