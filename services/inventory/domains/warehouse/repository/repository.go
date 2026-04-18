package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/KingWahid/inventory/backend/pkg/common/errorcodes"
	"gorm.io/gorm"
)

// Warehouse is a tenant-scoped warehouse.
type Warehouse struct {
	ID        string
	TenantID  string
	Code      string
	Name      string
	Address   *string
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

// ListFilter is list query options.
type ListFilter struct {
	Page    int
	PerPage int
	Search  string
	Sort    string
	Order   string
}

// Repository defines data-access for warehouses.
type Repository interface {
	Ping() error
	List(ctx context.Context, tenantID string, f ListFilter) ([]Warehouse, int64, error)
	GetByID(ctx context.Context, tenantID, id string) (Warehouse, error)
	Create(ctx context.Context, tenantID string, in CreateInput) (Warehouse, error)
	Update(ctx context.Context, tenantID, id string, in UpdateInput) (Warehouse, error)
	SoftDelete(ctx context.Context, tenantID, id string) error
	HasPositiveStock(ctx context.Context, tenantID, warehouseID string) (bool, error)
}

// CreateInput is insert payload.
type CreateInput struct {
	Code     string
	Name     string
	Address  *string
	IsActive *bool
}

// UpdateInput is partial update.
type UpdateInput struct {
	Code     *string
	Name     *string
	Address  *string
	IsActive *bool
}

type repository struct {
	db *gorm.DB
}

// New creates warehouse repository implementation.
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

func isDuplicateErr(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "duplicate") || strings.Contains(s, "unique")
}

func rowToWarehouse(m warehouseRow) Warehouse {
	return Warehouse{
		ID:        m.ID,
		TenantID:  m.TenantID,
		Code:      m.Code,
		Name:      m.Name,
		Address:   m.Address,
		IsActive:  m.IsActive,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
		DeletedAt: m.DeletedAt,
	}
}

func (r *repository) List(ctx context.Context, tenantID string, f ListFilter) ([]Warehouse, int64, error) {
	base := r.db.WithContext(ctx).Model(&warehouseRow{}).
		Where("tenant_id = ? AND deleted_at IS NULL", tenantID)

	if s := strings.TrimSpace(f.Search); s != "" {
		like := "%" + s + "%"
		base = base.Where("(name ILIKE ? OR code ILIKE ? OR address ILIKE ?)", like, like, like)
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	col := "name"
	switch strings.ToLower(strings.TrimSpace(f.Sort)) {
	case "code":
		col = "code"
	case "created_at":
		col = "created_at"
	case "updated_at":
		col = "updated_at"
	case "name":
		col = "name"
	case "is_active":
		col = "is_active"
	}
	ord := "ASC"
	if strings.EqualFold(strings.TrimSpace(f.Order), "desc") {
		ord = "DESC"
	}

	page := f.Page
	per := f.PerPage
	if page < 1 {
		page = 1
	}
	if per < 1 {
		per = 20
	}

	q := r.db.WithContext(ctx).Model(&warehouseRow{}).
		Where("tenant_id = ? AND deleted_at IS NULL", tenantID)
	if s := strings.TrimSpace(f.Search); s != "" {
		like := "%" + s + "%"
		q = q.Where("(name ILIKE ? OR code ILIKE ? OR address ILIKE ?)", like, like, like)
	}

	var rows []warehouseRow
	err := q.Order(col + " " + ord + ", id ASC").
		Offset((page - 1) * per).
		Limit(per).
		Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	out := make([]Warehouse, 0, len(rows))
	for i := range rows {
		out = append(out, rowToWarehouse(rows[i]))
	}
	return out, total, nil
}

func (r *repository) GetByID(ctx context.Context, tenantID, id string) (Warehouse, error) {
	var m warehouseRow
	err := r.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", strings.TrimSpace(id), tenantID).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Warehouse{}, errorcodes.ErrNotFound
		}
		return Warehouse{}, err
	}
	return rowToWarehouse(m), nil
}

func (r *repository) Create(ctx context.Context, tenantID string, in CreateInput) (Warehouse, error) {
	now := time.Now().UTC()
	active := true
	if in.IsActive != nil {
		active = *in.IsActive
	}
	m := warehouseRow{
		ID:        newUUID(),
		TenantID:  tenantID,
		Code:      strings.TrimSpace(in.Code),
		Name:      strings.TrimSpace(in.Name),
		Address:   in.Address,
		IsActive:  active,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		if isDuplicateErr(err) {
			return Warehouse{}, errorcodes.ErrConflict.WithDetails(map[string]any{"message": "warehouse code already exists"})
		}
		return Warehouse{}, err
	}
	return rowToWarehouse(m), nil
}

func (r *repository) Update(ctx context.Context, tenantID, id string, in UpdateInput) (Warehouse, error) {
	id = strings.TrimSpace(id)
	var cur warehouseRow
	err := r.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", id, tenantID).
		First(&cur).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Warehouse{}, errorcodes.ErrNotFound
		}
		return Warehouse{}, err
	}

	updates := map[string]any{"updated_at": time.Now().UTC()}
	if in.Code != nil {
		updates["code"] = strings.TrimSpace(*in.Code)
	}
	if in.Name != nil {
		updates["name"] = strings.TrimSpace(*in.Name)
	}
	if in.Address != nil {
		updates["address"] = in.Address
	}
	if in.IsActive != nil {
		updates["is_active"] = *in.IsActive
	}

	res := r.db.WithContext(ctx).Model(&warehouseRow{}).
		Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", id, tenantID).
		Updates(updates)
	if res.Error != nil {
		if isDuplicateErr(res.Error) {
			return Warehouse{}, errorcodes.ErrConflict.WithDetails(map[string]any{"message": "warehouse code already exists"})
		}
		return Warehouse{}, res.Error
	}

	var m warehouseRow
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		return Warehouse{}, err
	}
	return rowToWarehouse(m), nil
}

func (r *repository) SoftDelete(ctx context.Context, tenantID, id string) error {
	id = strings.TrimSpace(id)
	now := time.Now().UTC()
	res := r.db.WithContext(ctx).Model(&warehouseRow{}).
		Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", id, tenantID).
		Updates(map[string]any{
			"is_active":  false,
			"deleted_at": now,
			"updated_at": now,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errorcodes.ErrNotFound
	}
	return nil
}

func (r *repository) HasPositiveStock(ctx context.Context, tenantID, warehouseID string) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Raw(
		`SELECT COUNT(*) FROM stock_balances WHERE tenant_id = ?::uuid AND warehouse_id = ?::uuid AND quantity > 0`,
		tenantID, strings.TrimSpace(warehouseID),
	).Scan(&n).Error
	if err != nil {
		return false, err
	}
	return n > 0, nil
}
