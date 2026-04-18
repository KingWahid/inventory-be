package repository

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/KingWahid/inventory/backend/pkg/common/errorcodes"
	"gorm.io/gorm"
)

// Category is a tenant-scoped category returned from the repository.
type Category struct {
	ID          string
	TenantID    string
	ParentID    *string
	Name        string
	Description *string
	SortOrder   int32
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time
}

// ListFilter holds list query options (pagination + search + sort).
type ListFilter struct {
	Page    int
	PerPage int
	Search  string
	Sort    string
	Order   string
}

// Repository defines data-access for catalog (categories + products).
type Repository interface {
	Ping() error
	List(ctx context.Context, tenantID string, f ListFilter) ([]Category, int64, error)
	GetByID(ctx context.Context, tenantID, id string) (Category, error)
	Create(ctx context.Context, tenantID string, in CreateInput) (Category, error)
	Update(ctx context.Context, tenantID, id string, in UpdateInput) (Category, error)
	SoftDelete(ctx context.Context, tenantID, id string) error
	CountActiveProductsByCategoryID(ctx context.Context, tenantID, categoryID string) (int64, error)

	ListProducts(ctx context.Context, tenantID string, f ListProductsFilter) ([]Product, int64, error)
	GetProductByID(ctx context.Context, tenantID, id string) (Product, error)
	CreateProduct(ctx context.Context, tenantID string, in CreateProductInput) (Product, error)
	UpdateProduct(ctx context.Context, tenantID, id string, in UpdateProductInput) (Product, error)
	SoftDeleteProduct(ctx context.Context, tenantID, id string) error
	RestoreProduct(ctx context.Context, tenantID, id string) (Product, error)
	HasPositiveStock(ctx context.Context, tenantID, productID string) (bool, error)
}

// CreateInput is repository payload for insert.
type CreateInput struct {
	Name        string
	Description *string
	ParentID    *string
	SortOrder   *int32
}

// UpdateInput is repository payload for update (nil pointer = omit field).
type UpdateInput struct {
	Name        *string
	Description *string
	ParentID    *string
	SortOrder   *int32
}

// Product is a tenant-scoped product row for application layers.
type Product struct {
	ID           string
	TenantID     string
	CategoryID   *string
	SKU          string
	Name         string
	Description  *string
	Unit         string
	Price        float64
	ReorderLevel int32
	Metadata     json.RawMessage
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
}

// ListProductsFilter lists active products with optional category filter.
type ListProductsFilter struct {
	Page       int
	PerPage    int
	Search     string
	Sort       string
	Order      string
	CategoryID *string
}

// CreateProductInput is insert payload for products.
type CreateProductInput struct {
	CategoryID   *string
	SKU          string
	Name         string
	Description  *string
	Unit         *string
	Price        *float64
	ReorderLevel *int32
	Metadata     json.RawMessage
}

// UpdateProductInput is partial update for products (nil = omit).
type UpdateProductInput struct {
	CategoryID   *string
	SKU          *string
	Name         *string
	Description  *string
	Unit         *string
	Price        *float64
	ReorderLevel *int32
	Metadata     *json.RawMessage
}

type repository struct {
	db *gorm.DB
}

// New creates catalog repository implementation.
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

func rowToCategory(m categoryRow) Category {
	return Category{
		ID:          m.ID,
		TenantID:    m.TenantID,
		ParentID:    m.ParentID,
		Name:        m.Name,
		Description: m.Description,
		SortOrder:   m.SortOrder,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
		DeletedAt:   m.DeletedAt,
	}
}

func (r *repository) List(ctx context.Context, tenantID string, f ListFilter) ([]Category, int64, error) {
	base := r.db.WithContext(ctx).Model(&categoryRow{}).
		Where("tenant_id = ? AND deleted_at IS NULL", tenantID)

	if s := strings.TrimSpace(f.Search); s != "" {
		like := "%" + s + "%"
		base = base.Where("(name ILIKE ? OR description ILIKE ?)", like, like)
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	col := "sort_order"
	switch strings.ToLower(strings.TrimSpace(f.Sort)) {
	case "name":
		col = "name"
	case "created_at":
		col = "created_at"
	case "updated_at":
		col = "updated_at"
	case "sort_order":
		col = "sort_order"
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

	var rows []categoryRow
	q := r.db.WithContext(ctx).Model(&categoryRow{}).
		Where("tenant_id = ? AND deleted_at IS NULL", tenantID)
	if s := strings.TrimSpace(f.Search); s != "" {
		like := "%" + s + "%"
		q = q.Where("(name ILIKE ? OR description ILIKE ?)", like, like)
	}
	err := q.Order(col + " " + ord + ", id ASC").
		Offset((page - 1) * per).
		Limit(per).
		Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	out := make([]Category, 0, len(rows))
	for i := range rows {
		out = append(out, rowToCategory(rows[i]))
	}
	return out, total, nil
}

func (r *repository) GetByID(ctx context.Context, tenantID, id string) (Category, error) {
	var m categoryRow
	err := r.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", strings.TrimSpace(id), tenantID).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Category{}, errorcodes.ErrNotFound
		}
		return Category{}, err
	}
	return rowToCategory(m), nil
}

func (r *repository) Create(ctx context.Context, tenantID string, in CreateInput) (Category, error) {
	now := time.Now().UTC()
	sort := int32(0)
	if in.SortOrder != nil {
		sort = *in.SortOrder
	}
	m := categoryRow{
		ID:          newUUID(),
		TenantID:    tenantID,
		ParentID:    in.ParentID,
		Name:        strings.TrimSpace(in.Name),
		Description: in.Description,
		SortOrder:   sort,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") || strings.Contains(strings.ToLower(err.Error()), "unique") {
			return Category{}, errorcodes.ErrConflict.WithDetails(map[string]any{"message": "category name already exists"})
		}
		return Category{}, err
	}
	return rowToCategory(m), nil
}

func (r *repository) Update(ctx context.Context, tenantID, id string, in UpdateInput) (Category, error) {
	id = strings.TrimSpace(id)
	var cur categoryRow
	err := r.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", id, tenantID).
		First(&cur).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Category{}, errorcodes.ErrNotFound
		}
		return Category{}, err
	}

	updates := map[string]any{"updated_at": time.Now().UTC()}
	if in.Name != nil {
		updates["name"] = strings.TrimSpace(*in.Name)
	}
	if in.Description != nil {
		updates["description"] = in.Description
	}
	if in.SortOrder != nil {
		updates["sort_order"] = *in.SortOrder
	}
	if in.ParentID != nil {
		updates["parent_id"] = in.ParentID
	}

	res := r.db.WithContext(ctx).Model(&categoryRow{}).
		Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", id, tenantID).
		Updates(updates)
	if res.Error != nil {
		if strings.Contains(strings.ToLower(res.Error.Error()), "duplicate") || strings.Contains(strings.ToLower(res.Error.Error()), "unique") {
			return Category{}, errorcodes.ErrConflict.WithDetails(map[string]any{"message": "category name already exists"})
		}
		return Category{}, res.Error
	}

	var m categoryRow
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		return Category{}, err
	}
	return rowToCategory(m), nil
}

func (r *repository) SoftDelete(ctx context.Context, tenantID, id string) error {
	id = strings.TrimSpace(id)
	now := time.Now().UTC()
	res := r.db.WithContext(ctx).Model(&categoryRow{}).
		Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", id, tenantID).
		Update("deleted_at", now)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errorcodes.ErrNotFound
	}
	return nil
}

func (r *repository) CountActiveProductsByCategoryID(ctx context.Context, tenantID, categoryID string) (int64, error) {
	var n int64
	err := r.db.WithContext(ctx).Raw(
		`SELECT COUNT(*) FROM products WHERE tenant_id = ?::uuid AND category_id = ?::uuid AND deleted_at IS NULL`,
		tenantID, strings.TrimSpace(categoryID),
	).Scan(&n).Error
	return n, err
}
