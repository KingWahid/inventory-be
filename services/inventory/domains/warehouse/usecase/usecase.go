package usecase

import (
	"context"
	"strings"

	"github.com/KingWahid/inventory/backend/pkg/common/errorcodes"
	commonjwt "github.com/KingWahid/inventory/backend/pkg/common/jwt"
	"github.com/KingWahid/inventory/backend/pkg/common/pagination"
	"github.com/KingWahid/inventory/backend/services/inventory/domains/warehouse/repository"
)

// Usecase defines application logic for warehouses.
type Usecase interface {
	Ping() error
	ListWarehouses(ctx context.Context, in ListWarehousesInput) (ListWarehousesOutput, error)
	GetWarehouse(ctx context.Context, warehouseID string) (repository.Warehouse, error)
	CreateWarehouse(ctx context.Context, in CreateWarehouseInput) (repository.Warehouse, error)
	UpdateWarehouse(ctx context.Context, warehouseID string, in UpdateWarehouseInput) (repository.Warehouse, error)
	DeleteWarehouse(ctx context.Context, warehouseID string) error
}

// ListWarehousesInput maps from HTTP query params.
type ListWarehousesInput struct {
	Page    *int
	PerPage *int
	Search  *string
	Sort    *string
	Order   *string
}

// ListWarehousesOutput for §9 list + pagination.
type ListWarehousesOutput struct {
	Items   []repository.Warehouse
	Total   int64
	Page    int32
	PerPage int32
}

// CreateWarehouseInput is validated create payload.
type CreateWarehouseInput struct {
	Code     string
	Name     string
	Address  *string
	IsActive *bool
}

// UpdateWarehouseInput is validated update payload.
type UpdateWarehouseInput struct {
	Code     *string
	Name     *string
	Address  *string
	IsActive *bool
}

type usecase struct {
	repo repository.Repository
}

// New creates warehouse usecase implementation.
func New(repo repository.Repository) Usecase {
	return &usecase{repo: repo}
}

func (u *usecase) Ping() error {
	return u.repo.Ping()
}

func tenantFromCtx(ctx context.Context) (string, error) {
	return commonjwt.TenantIDFromContext(ctx)
}

func (u *usecase) ListWarehouses(ctx context.Context, in ListWarehousesInput) (ListWarehousesOutput, error) {
	tid, err := tenantFromCtx(ctx)
	if err != nil {
		return ListWarehousesOutput{}, err
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

	items, total, err := u.repo.List(ctx, tid, repository.ListFilter{
		Page:    page,
		PerPage: perPage,
		Search:  search,
		Sort:    sort,
		Order:   order,
	})
	if err != nil {
		return ListWarehousesOutput{}, err
	}

	return ListWarehousesOutput{
		Items:   items,
		Total:   total,
		Page:    int32(page),
		PerPage: int32(perPage),
	}, nil
}

func (u *usecase) GetWarehouse(ctx context.Context, warehouseID string) (repository.Warehouse, error) {
	tid, err := tenantFromCtx(ctx)
	if err != nil {
		return repository.Warehouse{}, err
	}
	return u.repo.GetByID(ctx, tid, strings.TrimSpace(warehouseID))
}

func (u *usecase) CreateWarehouse(ctx context.Context, in CreateWarehouseInput) (repository.Warehouse, error) {
	tid, err := tenantFromCtx(ctx)
	if err != nil {
		return repository.Warehouse{}, err
	}
	code := strings.TrimSpace(in.Code)
	name := strings.TrimSpace(in.Name)
	if code == "" {
		return repository.Warehouse{}, errorcodes.ErrValidationError.WithDetails(map[string]any{"message": "code is required"})
	}
	if name == "" {
		return repository.Warehouse{}, errorcodes.ErrValidationError.WithDetails(map[string]any{"message": "name is required"})
	}

	return u.repo.Create(ctx, tid, repository.CreateInput{
		Code:     code,
		Name:     name,
		Address:  in.Address,
		IsActive: in.IsActive,
	})
}

func (u *usecase) UpdateWarehouse(ctx context.Context, warehouseID string, in UpdateWarehouseInput) (repository.Warehouse, error) {
	tid, err := tenantFromCtx(ctx)
	if err != nil {
		return repository.Warehouse{}, err
	}
	id := strings.TrimSpace(warehouseID)
	if id == "" {
		return repository.Warehouse{}, errorcodes.ErrValidationError.WithDetails(map[string]any{"message": "warehouse id is required"})
	}

	if in.Code != nil && strings.TrimSpace(*in.Code) == "" {
		return repository.Warehouse{}, errorcodes.ErrValidationError.WithDetails(map[string]any{"message": "code cannot be empty"})
	}
	if in.Name != nil && strings.TrimSpace(*in.Name) == "" {
		return repository.Warehouse{}, errorcodes.ErrValidationError.WithDetails(map[string]any{"message": "name cannot be empty"})
	}

	return u.repo.Update(ctx, tid, id, repository.UpdateInput{
		Code:     in.Code,
		Name:     in.Name,
		Address:  in.Address,
		IsActive: in.IsActive,
	})
}

func (u *usecase) DeleteWarehouse(ctx context.Context, warehouseID string) error {
	tid, err := tenantFromCtx(ctx)
	if err != nil {
		return err
	}
	id := strings.TrimSpace(warehouseID)
	if id == "" {
		return errorcodes.ErrValidationError.WithDetails(map[string]any{"message": "warehouse id is required"})
	}

	if _, err := u.repo.GetByID(ctx, tid, id); err != nil {
		return err
	}

	has, err := u.repo.HasPositiveStock(ctx, tid, id)
	if err != nil {
		return err
	}
	if has {
		return errorcodes.ErrWarehouseStock.WithDetails(map[string]any{"warehouse_id": id})
	}

	return u.repo.SoftDelete(ctx, tid, id)
}
