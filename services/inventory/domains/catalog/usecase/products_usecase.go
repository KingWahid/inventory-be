package usecase

import (
	"context"
	"encoding/json"
	"strings"

	cachepkg "github.com/KingWahid/inventory/backend/pkg/cache"
	"github.com/KingWahid/inventory/backend/pkg/common/errorcodes"
	"github.com/KingWahid/inventory/backend/pkg/common/pagination"
	"github.com/KingWahid/inventory/backend/services/inventory/domains/audit/logwriter"
	"github.com/KingWahid/inventory/backend/services/inventory/domains/catalog/repository"
)

const maxProductMetadataBytes = 65536

func validateJSONObjectMetadata(raw []byte) error {
	if len(raw) == 0 {
		return nil
	}
	if len(raw) > maxProductMetadataBytes {
		return errorcodes.ErrValidationError.WithDetails(map[string]any{"message": "metadata too large"})
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return errorcodes.ErrValidationError.WithDetails(map[string]any{"message": "metadata must be valid JSON"})
	}
	if _, ok := v.(map[string]any); !ok {
		return errorcodes.ErrValidationError.WithDetails(map[string]any{"message": "metadata must be a JSON object"})
	}
	return nil
}

func normalizeMetadataJSON(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage("{}")
	}
	return raw
}

func (u *usecase) ListProducts(ctx context.Context, in ListProductsInput) (ListProductsOutput, error) {
	tid, err := tenantFromCtx(ctx)
	if err != nil {
		return ListProductsOutput{}, err
	}

	page := 1
	perPage := 20
	if in.Page != nil {
		page = *in.Page
	}
	if in.PerPage != nil {
		perPage = *in.PerPage
	}
	pagination.Normalize(&page, &perPage)

	search := ""
	if in.Search != nil {
		search = strings.TrimSpace(*in.Search)
	}
	sort := ""
	if in.Sort != nil {
		sort = *in.Sort
	}
	order := ""
	if in.Order != nil {
		order = *in.Order
	}

	var catFilter *string
	catFP := ""
	if in.CategoryID != nil {
		s := strings.TrimSpace(*in.CategoryID)
		if s != "" {
			catFilter = &s
			catFP = s
		}
	}

	fp := cachepkg.ProductsFP(page, perPage, search, sort, order, catFP)
	listKey := cachepkg.KeyProductsList(tid, fp)
	if raw, hit, err := u.cache.Get(ctx, listKey); err == nil && hit {
		var out ListProductsOutput
		if err := json.Unmarshal(raw, &out); err == nil {
			return out, nil
		}
	}

	items, total, err := u.repo.ListProducts(ctx, tid, repository.ListProductsFilter{
		Page:       page,
		PerPage:    perPage,
		Search:     search,
		Sort:       sort,
		Order:      order,
		CategoryID: catFilter,
	})
	if err != nil {
		return ListProductsOutput{}, err
	}

	out := ListProductsOutput{
		Items:   items,
		Total:   total,
		Page:    int32(page),
		PerPage: int32(perPage),
	}
	if payload, err := json.Marshal(out); err == nil {
		_ = u.cache.Set(ctx, listKey, payload, cachepkg.TTLProductList)
	}
	return out, nil
}

func (u *usecase) GetProduct(ctx context.Context, productID string) (repository.Product, error) {
	tid, err := tenantFromCtx(ctx)
	if err != nil {
		return repository.Product{}, err
	}
	id := strings.TrimSpace(productID)
	ckey := cachepkg.KeyProduct(tid, id)
	if raw, hit, err := u.cache.Get(ctx, ckey); err == nil && hit {
		var p repository.Product
		if err := json.Unmarshal(raw, &p); err == nil {
			return p, nil
		}
	}
	p, err := u.repo.GetProductByID(ctx, tid, id)
	if err != nil {
		return repository.Product{}, err
	}
	if payload, err := json.Marshal(p); err == nil {
		_ = u.cache.Set(ctx, ckey, payload, cachepkg.TTLProductOne)
	}
	return p, nil
}

func (u *usecase) ensureCategory(ctx context.Context, tenantID string, categoryID *string) error {
	if categoryID == nil {
		return nil
	}
	cid := strings.TrimSpace(*categoryID)
	if cid == "" {
		return nil
	}
	if _, err := u.repo.GetByID(ctx, tenantID, cid); err != nil {
		return err
	}
	return nil
}

func (u *usecase) CreateProduct(ctx context.Context, in CreateProductInput) (repository.Product, error) {
	tid, err := tenantFromCtx(ctx)
	if err != nil {
		return repository.Product{}, err
	}

	sku := strings.TrimSpace(in.SKU)
	name := strings.TrimSpace(in.Name)
	if sku == "" {
		return repository.Product{}, errorcodes.ErrValidationError.WithDetails(map[string]any{"message": "sku is required"})
	}
	if name == "" {
		return repository.Product{}, errorcodes.ErrValidationError.WithDetails(map[string]any{"message": "name is required"})
	}

	meta := normalizeMetadataJSON(in.MetadataJSON)
	if err := validateJSONObjectMetadata(meta); err != nil {
		return repository.Product{}, err
	}

	if in.ReorderLevel != nil && *in.ReorderLevel < 0 {
		return repository.Product{}, errorcodes.ErrValidationError.WithDetails(map[string]any{"message": "reorder_level must be >= 0"})
	}
	if in.Price != nil && *in.Price < 0 {
		return repository.Product{}, errorcodes.ErrValidationError.WithDetails(map[string]any{"message": "price must be >= 0"})
	}

	catID := normalizeParentID(in.CategoryID)
	if err := u.ensureCategory(ctx, tid, catID); err != nil {
		return repository.Product{}, err
	}

	p, err := u.repo.CreateProduct(ctx, tid, repository.CreateProductInput{
		CategoryID:   catID,
		SKU:          sku,
		Name:         name,
		Description:  in.Description,
		Unit:         in.Unit,
		Price:        in.Price,
		ReorderLevel: in.ReorderLevel,
		Metadata:     meta,
	})
	if err != nil {
		return repository.Product{}, err
	}
	if u.auditLog != nil {
		_ = u.auditLog.Log(ctx, logwriter.Params{
			Action:   "product.create",
			Entity:   "product",
			EntityID: p.ID,
			Before:   nil,
			After:    toAuditMap(p),
		})
	}
	u.invalidateAfterProductWrite(ctx, tid, p.ID)
	return p, nil
}

func (u *usecase) UpdateProduct(ctx context.Context, productID string, in UpdateProductInput) (repository.Product, error) {
	tid, err := tenantFromCtx(ctx)
	if err != nil {
		return repository.Product{}, err
	}
	id := strings.TrimSpace(productID)
	if id == "" {
		return repository.Product{}, errorcodes.ErrValidationError.WithDetails(map[string]any{"message": "product id is required"})
	}

	if in.SKU != nil && strings.TrimSpace(*in.SKU) == "" {
		return repository.Product{}, errorcodes.ErrValidationError.WithDetails(map[string]any{"message": "sku cannot be empty"})
	}
	if in.Name != nil && strings.TrimSpace(*in.Name) == "" {
		return repository.Product{}, errorcodes.ErrValidationError.WithDetails(map[string]any{"message": "name cannot be empty"})
	}
	if in.ReorderLevel != nil && *in.ReorderLevel < 0 {
		return repository.Product{}, errorcodes.ErrValidationError.WithDetails(map[string]any{"message": "reorder_level must be >= 0"})
	}
	if in.Price != nil && *in.Price < 0 {
		return repository.Product{}, errorcodes.ErrValidationError.WithDetails(map[string]any{"message": "price must be >= 0"})
	}

	if in.MetadataJSON != nil {
		meta := normalizeMetadataJSON(*in.MetadataJSON)
		if err := validateJSONObjectMetadata(meta); err != nil {
			return repository.Product{}, err
		}
	}

	if in.CategoryID != nil {
		cid := strings.TrimSpace(*in.CategoryID)
		if cid != "" {
			if _, err := u.repo.GetByID(ctx, tid, cid); err != nil {
				return repository.Product{}, err
			}
		}
	}

	old, err := u.repo.GetProductByID(ctx, tid, id)
	if err != nil {
		return repository.Product{}, err
	}

	repoIn := repository.UpdateProductInput{
		SKU:          in.SKU,
		Name:         in.Name,
		Description:  in.Description,
		Unit:         in.Unit,
		Price:        in.Price,
		ReorderLevel: in.ReorderLevel,
	}
	if in.CategoryID != nil {
		c := normalizeParentID(in.CategoryID)
		repoIn.CategoryID = c
	}
	if in.MetadataJSON != nil {
		meta := normalizeMetadataJSON(*in.MetadataJSON)
		repoIn.Metadata = &meta
	}

	p, err := u.repo.UpdateProduct(ctx, tid, id, repoIn)
	if err != nil {
		return repository.Product{}, err
	}
	if u.auditLog != nil {
		_ = u.auditLog.Log(ctx, logwriter.Params{
			Action:   "product.update",
			Entity:   "product",
			EntityID: id,
			Before:   toAuditMap(old),
			After:    toAuditMap(p),
		})
	}
	u.invalidateAfterProductWrite(ctx, tid, id)
	return p, nil
}

func (u *usecase) DeleteProduct(ctx context.Context, productID string) error {
	tid, err := tenantFromCtx(ctx)
	if err != nil {
		return err
	}
	id := strings.TrimSpace(productID)
	if id == "" {
		return errorcodes.ErrValidationError.WithDetails(map[string]any{"message": "product id is required"})
	}

	old, err := u.repo.GetProductByID(ctx, tid, id)
	if err != nil {
		return err
	}

	has, err := u.repo.HasPositiveStock(ctx, tid, id)
	if err != nil {
		return err
	}
	if has {
		return errorcodes.ErrProductHasStock.WithDetails(map[string]any{"product_id": id})
	}

	if err := u.repo.SoftDeleteProduct(ctx, tid, id); err != nil {
		return err
	}
	if u.auditLog != nil {
		_ = u.auditLog.Log(ctx, logwriter.Params{
			Action:   "product.delete",
			Entity:   "product",
			EntityID: id,
			Before:   toAuditMap(old),
			After:    map[string]any{"deleted": true},
		})
	}
	u.invalidateAfterProductWrite(ctx, tid, id)
	return nil
}

func (u *usecase) RestoreProduct(ctx context.Context, productID string) (repository.Product, error) {
	tid, err := tenantFromCtx(ctx)
	if err != nil {
		return repository.Product{}, err
	}
	id := strings.TrimSpace(productID)
	if id == "" {
		return repository.Product{}, errorcodes.ErrValidationError.WithDetails(map[string]any{"message": "product id is required"})
	}

	p, err := u.repo.RestoreProduct(ctx, tid, id)
	if err != nil {
		return repository.Product{}, err
	}
	if u.auditLog != nil {
		_ = u.auditLog.Log(ctx, logwriter.Params{
			Action:   "product.restore",
			Entity:   "product",
			EntityID: id,
			Before:   map[string]any{"product_id": id, "deleted": true},
			After:    toAuditMap(p),
		})
	}
	u.invalidateAfterProductWrite(ctx, tid, id)
	return p, nil
}
