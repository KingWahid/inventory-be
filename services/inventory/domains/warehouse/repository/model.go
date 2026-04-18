package repository

import (
	"time"

	"github.com/google/uuid"
)

// warehouseRow maps to table warehouses.
type warehouseRow struct {
	ID        string     `gorm:"column:id;type:uuid;primaryKey"`
	TenantID  string     `gorm:"column:tenant_id;type:uuid"`
	Code      string     `gorm:"column:code"`
	Name      string     `gorm:"column:name"`
	Address   *string    `gorm:"column:address"`
	IsActive  bool       `gorm:"column:is_active"`
	CreatedAt time.Time  `gorm:"column:created_at"`
	UpdatedAt time.Time  `gorm:"column:updated_at"`
	DeletedAt *time.Time `gorm:"column:deleted_at"`
}

func (warehouseRow) TableName() string { return "warehouses" }

func newUUID() string {
	return uuid.New().String()
}
