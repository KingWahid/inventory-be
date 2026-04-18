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

func rowToProduct(m productRow) Product {
	meta := json.RawMessage(m.Metadata)
	if len(meta) == 0 {
		meta = json.RawMessage("{}")
	}
	return Product{
		ID:           m.ID,
		TenantID:     m.TenantID,
		CategoryID:   m.CategoryID,
		SKU:          m.SKU,
		Name:         m.Name,
		Description:  m.Description,
		Unit:         m.Unit,
		Price:        m.Price,
		ReorderLevel: m.ReorderLevel,
		Metadata:     meta,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
		DeletedAt:    m.DeletedAt,
	}
}

func isDuplicateErr(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "duplicate") || strings.Contains(s, "unique")
}

func (r *repository) ListProducts(ctx context.Context, tenantID string, f ListProductsFilter) ([]Product, int64, error) {
	base := r.db.WithContext(ctx).Model(&productRow{}).
		Where("tenant_id = ? AND deleted_at IS NULL", tenantID)

	if s := strings.TrimSpace(f.Search); s != "" {
		like := "%" + s + "%"
		base = base.Where("(name ILIKE ? OR sku ILIKE ? OR description ILIKE ?)", like, like, like)
	}
	if f.CategoryID != nil && strings.TrimSpace(*f.CategoryID) != "" {
		base = base.Where("category_id = ?", strings.TrimSpace(*f.CategoryID))
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	col := "name"
	switch strings.ToLower(strings.TrimSpace(f.Sort)) {
	case "sku":
		col = "sku"
	case "created_at":
		col = "created_at"
	case "updated_at":
		col = "updated_at"
	case "price":
		col = "price"
	case "reorder_level":
		col = "reorder_level"
	case "name":
		col = "name"
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

	q := r.db.WithContext(ctx).Model(&productRow{}).
		Where("tenant_id = ? AND deleted_at IS NULL", tenantID)
	if s := strings.TrimSpace(f.Search); s != "" {
		like := "%" + s + "%"
		q = q.Where("(name ILIKE ? OR sku ILIKE ? OR description ILIKE ?)", like, like, like)
	}
	if f.CategoryID != nil && strings.TrimSpace(*f.CategoryID) != "" {
		q = q.Where("category_id = ?", strings.TrimSpace(*f.CategoryID))
	}

	var rows []productRow
	err := q.Order(col + " " + ord + ", id ASC").
		Offset((page - 1) * per).
		Limit(per).
		Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	out := make([]Product, 0, len(rows))
	for i := range rows {
		out = append(out, rowToProduct(rows[i]))
	}
	return out, total, nil
}

func (r *repository) GetProductByID(ctx context.Context, tenantID, id string) (Product, error) {
	var m productRow
	err := r.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", strings.TrimSpace(id), tenantID).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Product{}, errorcodes.ErrNotFound
		}
		return Product{}, err
	}
	return rowToProduct(m), nil
}

func (r *repository) CreateProduct(ctx context.Context, tenantID string, in CreateProductInput) (Product, error) {
	now := time.Now().UTC()
	unit := "pcs"
	if in.Unit != nil && strings.TrimSpace(*in.Unit) != "" {
		unit = strings.TrimSpace(*in.Unit)
	}
	price := 0.0
	if in.Price != nil {
		price = *in.Price
	}
	rl := int32(0)
	if in.ReorderLevel != nil {
		rl = *in.ReorderLevel
	}
	meta := in.Metadata
	if len(meta) == 0 {
		meta = json.RawMessage("{}")
	}

	m := productRow{
		ID:           newUUID(),
		TenantID:     tenantID,
		CategoryID:   in.CategoryID,
		SKU:          strings.TrimSpace(in.SKU),
		Name:         strings.TrimSpace(in.Name),
		Description:  in.Description,
		Unit:         unit,
		Price:        price,
		ReorderLevel: rl,
		Metadata:     []byte(meta),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := r.db.WithContext(ctx).Create(&m).Error; err != nil {
		if isDuplicateErr(err) {
			return Product{}, errorcodes.ErrConflict.WithDetails(map[string]any{"message": "sku already exists"})
		}
		return Product{}, err
	}
	return rowToProduct(m), nil
}

func (r *repository) UpdateProduct(ctx context.Context, tenantID, id string, in UpdateProductInput) (Product, error) {
	id = strings.TrimSpace(id)
	var cur productRow
	err := r.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", id, tenantID).
		First(&cur).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Product{}, errorcodes.ErrNotFound
		}
		return Product{}, err
	}

	updates := map[string]any{"updated_at": time.Now().UTC()}
	if in.SKU != nil {
		updates["sku"] = strings.TrimSpace(*in.SKU)
	}
	if in.Name != nil {
		updates["name"] = strings.TrimSpace(*in.Name)
	}
	if in.Description != nil {
		updates["description"] = in.Description
	}
	if in.Unit != nil {
		updates["unit"] = strings.TrimSpace(*in.Unit)
	}
	if in.Price != nil {
		updates["price"] = *in.Price
	}
	if in.ReorderLevel != nil {
		updates["reorder_level"] = *in.ReorderLevel
	}
	if in.Metadata != nil {
		meta := *in.Metadata
		if len(meta) == 0 {
			meta = json.RawMessage("{}")
		}
		updates["metadata"] = []byte(meta)
	}
	if in.CategoryID != nil {
		if *in.CategoryID == "" {
			updates["category_id"] = nil
		} else {
			updates["category_id"] = *in.CategoryID
		}
	}

	res := r.db.WithContext(ctx).Model(&productRow{}).
		Where("id = ? AND tenant_id = ? AND deleted_at IS NULL", id, tenantID).
		Updates(updates)
	if res.Error != nil {
		if isDuplicateErr(res.Error) {
			return Product{}, errorcodes.ErrConflict.WithDetails(map[string]any{"message": "sku already exists"})
		}
		return Product{}, res.Error
	}

	var m productRow
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&m).Error; err != nil {
		return Product{}, err
	}
	return rowToProduct(m), nil
}

func (r *repository) SoftDeleteProduct(ctx context.Context, tenantID, id string) error {
	id = strings.TrimSpace(id)
	now := time.Now().UTC()
	res := r.db.WithContext(ctx).Model(&productRow{}).
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

func (r *repository) RestoreProduct(ctx context.Context, tenantID, id string) (Product, error) {
	id = strings.TrimSpace(id)
	now := time.Now().UTC()
	res := r.db.WithContext(ctx).Model(&productRow{}).
		Where("id = ? AND tenant_id = ? AND deleted_at IS NOT NULL", id, tenantID).
		Updates(map[string]any{
			"deleted_at": nil,
			"updated_at": now,
		})
	if res.Error != nil {
		if isDuplicateErr(res.Error) {
			return Product{}, errorcodes.ErrConflict.WithDetails(map[string]any{"message": "sku already exists"})
		}
		return Product{}, res.Error
	}
	if res.RowsAffected == 0 {
		return Product{}, errorcodes.ErrNotFound
	}
	return r.GetProductByID(ctx, tenantID, id)
}

func (r *repository) HasPositiveStock(ctx context.Context, tenantID, productID string) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Raw(
		`SELECT COUNT(*) FROM stock_balances WHERE tenant_id = ?::uuid AND product_id = ?::uuid AND quantity > 0`,
		tenantID, strings.TrimSpace(productID),
	).Scan(&n).Error
	if err != nil {
		return false, err
	}
	return n > 0, nil
}
