package usecase

import (
	"context"
	"encoding/json"
	"strings"

	cachepkg "github.com/KingWahid/inventory/backend/pkg/cache"
	"github.com/KingWahid/inventory/backend/pkg/common/errorcodes"
	commonjwt "github.com/KingWahid/inventory/backend/pkg/common/jwt"
	"github.com/KingWahid/inventory/backend/pkg/common/pagination"
	"github.com/KingWahid/inventory/backend/services/inventory/domains/audit/logwriter"
	"github.com/KingWahid/inventory/backend/services/inventory/domains/catalog/repository"
)

// Usecase defines catalog application logic.
type Usecase interface {
	Ping() error
	ListCategories(ctx context.Context, in ListCategoriesInput) (ListCategoriesOutput, error)
	GetCategory(ctx context.Context, categoryID string) (repository.Category, error)
	CreateCategory(ctx context.Context, in CreateCategoryInput) (repository.Category, error)
	UpdateCategory(ctx context.Context, categoryID string, in UpdateCategoryInput) (repository.Category, error)
	DeleteCategory(ctx context.Context, categoryID string) error

	ListProducts(ctx context.Context, in ListProductsInput) (ListProductsOutput, error)
	GetProduct(ctx context.Context, productID string) (repository.Product, error)
	CreateProduct(ctx context.Context, in CreateProductInput) (repository.Product, error)
	UpdateProduct(ctx context.Context, productID string, in UpdateProductInput) (repository.Product, error)
	DeleteProduct(ctx context.Context, productID string) error
	RestoreProduct(ctx context.Context, productID string) (repository.Product, error)
}

// ListCategoriesInput maps from HTTP query params.
type ListCategoriesInput struct {
	Page    *int
	PerPage *int
	Search  *string
	Sort    *string
	Order   *string
}

// ListCategoriesOutput is used for §9 list + pagination meta.
type ListCategoriesOutput struct {
	Items   []repository.Category
	Total   int64
	Page    int32
	PerPage int32
}

// CreateCategoryInput is validated create payload.
type CreateCategoryInput struct {
	Name        string
	Description *string
	ParentID    *string
	SortOrder   *int32
}

// UpdateCategoryInput is validated update payload.
type UpdateCategoryInput struct {
	Name        *string
	Description *string
	ParentID    *string
	SortOrder   *int32
}

// ListProductsInput maps from HTTP query params for products.
type ListProductsInput struct {
	Page       *int
	PerPage    *int
	Search     *string
	Sort       *string
	Order      *string
	CategoryID *string
}

// ListProductsOutput is §9 list + totals for products.
type ListProductsOutput struct {
	Items   []repository.Product
	Total   int64
	Page    int32
	PerPage int32
}

// CreateProductInput is validated create payload for products.
type CreateProductInput struct {
	CategoryID   *string
	SKU          string
	Name         string
	Description  *string
	Unit         *string
	Price        *float64
	ReorderLevel *int32
	MetadataJSON json.RawMessage
}

// UpdateProductInput is validated update payload for products.
type UpdateProductInput struct {
	CategoryID   *string
	SKU          *string
	Name         *string
	Description  *string
	Unit         *string
	Price        *float64
	ReorderLevel *int32
	MetadataJSON *json.RawMessage
}

type usecase struct {
	repo     repository.Repository
	auditLog *logwriter.Writer
	cache    cachepkg.Cache
}

// New creates catalog usecase implementation (cache may be nil → noop via provider).
func New(repo repository.Repository, audit *logwriter.Writer, c cachepkg.Cache) Usecase {
	if c == nil {
		c = cachepkg.Noop{}
	}
	return &usecase{repo: repo, auditLog: audit, cache: c}
}

func (u *usecase) Ping() error {
	return u.repo.Ping()
}

func tenantFromCtx(ctx context.Context) (string, error) {
	return commonjwt.TenantIDFromContext(ctx)
}

func (u *usecase) ListCategories(ctx context.Context, in ListCategoriesInput) (ListCategoriesOutput, error) {
	tid, err := tenantFromCtx(ctx)
	if err != nil {
		return ListCategoriesOutput{}, err
	}

	page := 1
	perPage := 10
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

	fp := cachepkg.CategoriesFP(page, perPage, search, sort, order)
	ckey := cachepkg.KeyCategoriesList(tid, fp)
	if raw, hit, err := u.cache.Get(ctx, ckey); err == nil && hit {
		var out ListCategoriesOutput
		if err := json.Unmarshal(raw, &out); err == nil {
			return out, nil
		}
	}

	items, total, err := u.repo.List(ctx, tid, repository.ListFilter{
		Page:    page,
		PerPage: perPage,
		Search:  search,
		Sort:    sort,
		Order:   order,
	})
	if err != nil {
		return ListCategoriesOutput{}, err
	}

	out := ListCategoriesOutput{
		Items:   items,
		Total:   total,
		Page:    int32(page),
		PerPage: int32(perPage),
	}
	if payload, err := json.Marshal(out); err == nil {
		_ = u.cache.Set(ctx, ckey, payload, cachepkg.TTLCategoryList)
	}
	return out, nil
}

func (u *usecase) GetCategory(ctx context.Context, categoryID string) (repository.Category, error) {
	tid, err := tenantFromCtx(ctx)
	if err != nil {
		return repository.Category{}, err
	}
	return u.repo.GetByID(ctx, tid, strings.TrimSpace(categoryID))
}

func (u *usecase) CreateCategory(ctx context.Context, in CreateCategoryInput) (repository.Category, error) {
	tid, err := tenantFromCtx(ctx)
	if err != nil {
		return repository.Category{}, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return repository.Category{}, errorcodes.ErrValidationError.WithDetails(map[string]any{"message": "name is required"})
	}

	if in.ParentID != nil && strings.TrimSpace(*in.ParentID) != "" {
		pid := strings.TrimSpace(*in.ParentID)
		if _, err := u.repo.GetByID(ctx, tid, pid); err != nil {
			return repository.Category{}, err
		}
	}

	cat, err := u.repo.Create(ctx, tid, repository.CreateInput{
		Name:        name,
		Description: in.Description,
		ParentID:    normalizeParentID(in.ParentID),
		SortOrder:   in.SortOrder,
	})
	if err != nil {
		return repository.Category{}, err
	}
	if u.auditLog != nil {
		_ = u.auditLog.Log(ctx, logwriter.Params{
			Action:   "category.create",
			Entity:   "category",
			EntityID: cat.ID,
			Before:   nil,
			After:    toAuditMap(cat),
		})
	}
	u.invalidateAfterCategoryWrite(ctx, tid)
	return cat, nil
}

func (u *usecase) UpdateCategory(ctx context.Context, categoryID string, in UpdateCategoryInput) (repository.Category, error) {
	tid, err := tenantFromCtx(ctx)
	if err != nil {
		return repository.Category{}, err
	}
	id := strings.TrimSpace(categoryID)
	if id == "" {
		return repository.Category{}, errorcodes.ErrValidationError.WithDetails(map[string]any{"message": "category id is required"})
	}

	if in.ParentID != nil {
		p := strings.TrimSpace(*in.ParentID)
		if p != "" {
			if p == id {
				return repository.Category{}, errorcodes.ErrValidationError.WithDetails(map[string]any{"message": "parent_id cannot equal category id"})
			}
			if _, err := u.repo.GetByID(ctx, tid, p); err != nil {
				return repository.Category{}, err
			}
		}
	}

	if in.Name != nil && strings.TrimSpace(*in.Name) == "" {
		return repository.Category{}, errorcodes.ErrValidationError.WithDetails(map[string]any{"message": "name cannot be empty"})
	}

	old, err := u.repo.GetByID(ctx, tid, id)
	if err != nil {
		return repository.Category{}, err
	}

	repoIn := repository.UpdateInput{
		Name:        in.Name,
		Description: in.Description,
		SortOrder:   in.SortOrder,
	}
	if in.ParentID != nil {
		pid := normalizeParentID(in.ParentID)
		repoIn.ParentID = pid
	}

	cat, err := u.repo.Update(ctx, tid, id, repoIn)
	if err != nil {
		return repository.Category{}, err
	}
	if u.auditLog != nil {
		_ = u.auditLog.Log(ctx, logwriter.Params{
			Action:   "category.update",
			Entity:   "category",
			EntityID: id,
			Before:   toAuditMap(old),
			After:    toAuditMap(cat),
		})
	}
	u.invalidateAfterCategoryWrite(ctx, tid)
	return cat, nil
}

func normalizeParentID(p *string) *string {
	if p == nil {
		return nil
	}
	s := strings.TrimSpace(*p)
	if s == "" {
		return nil
	}
	return &s
}

func (u *usecase) DeleteCategory(ctx context.Context, categoryID string) error {
	tid, err := tenantFromCtx(ctx)
	if err != nil {
		return err
	}
	id := strings.TrimSpace(categoryID)
	if id == "" {
		return errorcodes.ErrValidationError.WithDetails(map[string]any{"message": "category id is required"})
	}

	old, err := u.repo.GetByID(ctx, tid, id)
	if err != nil {
		return err
	}

	n, err := u.repo.CountActiveProductsByCategoryID(ctx, tid, id)
	if err != nil {
		return err
	}
	if n > 0 {
		return errorcodes.ErrCategoryHasActiveProducts.WithDetails(map[string]any{"category_id": id, "active_products": n})
	}

	if err := u.repo.SoftDelete(ctx, tid, id); err != nil {
		return err
	}
	if u.auditLog != nil {
		_ = u.auditLog.Log(ctx, logwriter.Params{
			Action:   "category.delete",
			Entity:   "category",
			EntityID: id,
			Before:   toAuditMap(old),
			After:    map[string]any{"deleted": true},
		})
	}
	u.invalidateAfterCategoryWrite(ctx, tid)
	return nil
}
