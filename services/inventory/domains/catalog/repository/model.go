package repository

import (
	"time"

	"github.com/google/uuid"
)

// categoryRow maps to table categories (explicit soft delete column).
type categoryRow struct {
	ID          string     `gorm:"column:id;type:uuid;primaryKey"`
	TenantID    string     `gorm:"column:tenant_id;type:uuid"`
	ParentID    *string    `gorm:"column:parent_id;type:uuid"`
	Name        string     `gorm:"column:name"`
	Description *string    `gorm:"column:description"`
	SortOrder   int32      `gorm:"column:sort_order"`
	CreatedAt   time.Time  `gorm:"column:created_at"`
	UpdatedAt   time.Time  `gorm:"column:updated_at"`
	DeletedAt   *time.Time `gorm:"column:deleted_at"`
}

func (categoryRow) TableName() string { return "categories" }

func newUUID() string {
	return uuid.New().String()
}

// productRow maps to table products.
type productRow struct {
	ID           string     `gorm:"column:id;type:uuid;primaryKey"`
	TenantID     string     `gorm:"column:tenant_id;type:uuid"`
	CategoryID   *string    `gorm:"column:category_id;type:uuid"`
	SKU          string     `gorm:"column:sku"`
	Name         string     `gorm:"column:name"`
	Description  *string    `gorm:"column:description"`
	Unit         string     `gorm:"column:unit"`
	Price        float64    `gorm:"column:price"`
	ReorderLevel int32      `gorm:"column:reorder_level"`
	Metadata     []byte     `gorm:"column:metadata;type:jsonb"`
	CreatedAt    time.Time  `gorm:"column:created_at"`
	UpdatedAt    time.Time  `gorm:"column:updated_at"`
	DeletedAt    *time.Time `gorm:"column:deleted_at"`
}

func (productRow) TableName() string { return "products" }
