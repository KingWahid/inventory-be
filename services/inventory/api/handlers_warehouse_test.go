package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/KingWahid/inventory/backend/pkg/common/errorcodes"
	commonjwt "github.com/KingWahid/inventory/backend/pkg/common/jwt"
	catalogrepo "github.com/KingWahid/inventory/backend/services/inventory/domains/catalog/repository"
	cataloguc "github.com/KingWahid/inventory/backend/services/inventory/domains/catalog/usecase"
	dashboarduc "github.com/KingWahid/inventory/backend/services/inventory/domains/dashboard/usecase"
	warehouserepo "github.com/KingWahid/inventory/backend/services/inventory/domains/warehouse/repository"
	warehouseuc "github.com/KingWahid/inventory/backend/services/inventory/domains/warehouse/usecase"
	"github.com/KingWahid/inventory/backend/services/inventory/stub"
)

type warehouseDelSvc struct {
	movementEmbedNoop
}

func (warehouseDelSvc) PingDB(context.Context) error { return nil }

func (warehouseDelSvc) ListCategories(context.Context, cataloguc.ListCategoriesInput) (cataloguc.ListCategoriesOutput, error) {
	return cataloguc.ListCategoriesOutput{}, errorcodes.ErrNotFound
}

func (warehouseDelSvc) GetCategory(context.Context, string) (catalogrepo.Category, error) {
	return catalogrepo.Category{}, errorcodes.ErrNotFound
}

func (warehouseDelSvc) CreateCategory(context.Context, cataloguc.CreateCategoryInput) (catalogrepo.Category, error) {
	return catalogrepo.Category{}, errorcodes.ErrNotFound
}

func (warehouseDelSvc) UpdateCategory(context.Context, string, cataloguc.UpdateCategoryInput) (catalogrepo.Category, error) {
	return catalogrepo.Category{}, errorcodes.ErrNotFound
}

func (warehouseDelSvc) DeleteCategory(context.Context, string) error { return nil }

func (warehouseDelSvc) ListProducts(context.Context, cataloguc.ListProductsInput) (cataloguc.ListProductsOutput, error) {
	return cataloguc.ListProductsOutput{}, nil
}

func (warehouseDelSvc) GetProduct(context.Context, string) (catalogrepo.Product, error) {
	return catalogrepo.Product{}, errorcodes.ErrNotFound
}

func (warehouseDelSvc) CreateProduct(context.Context, cataloguc.CreateProductInput) (catalogrepo.Product, error) {
	return catalogrepo.Product{}, errorcodes.ErrNotFound
}

func (warehouseDelSvc) UpdateProduct(context.Context, string, cataloguc.UpdateProductInput) (catalogrepo.Product, error) {
	return catalogrepo.Product{}, errorcodes.ErrNotFound
}

func (warehouseDelSvc) DeleteProduct(context.Context, string) error { return nil }

func (warehouseDelSvc) RestoreProduct(context.Context, string) (catalogrepo.Product, error) {
	return catalogrepo.Product{}, errorcodes.ErrNotFound
}

func (warehouseDelSvc) ListWarehouses(context.Context, warehouseuc.ListWarehousesInput) (warehouseuc.ListWarehousesOutput, error) {
	return warehouseuc.ListWarehousesOutput{}, nil
}

func (warehouseDelSvc) GetWarehouse(context.Context, string) (warehouserepo.Warehouse, error) {
	return warehouserepo.Warehouse{}, errorcodes.ErrNotFound
}

func (warehouseDelSvc) CreateWarehouse(context.Context, warehouseuc.CreateWarehouseInput) (warehouserepo.Warehouse, error) {
	return warehouserepo.Warehouse{}, errorcodes.ErrNotFound
}

func (warehouseDelSvc) UpdateWarehouse(context.Context, string, warehouseuc.UpdateWarehouseInput) (warehouserepo.Warehouse, error) {
	return warehouserepo.Warehouse{}, errorcodes.ErrNotFound
}

func (warehouseDelSvc) DeleteWarehouse(context.Context, string) error {
	return errorcodes.ErrWarehouseStock.WithDetails(map[string]any{"warehouse_id": "x"})
}

func (warehouseDelSvc) GetDashboardStorageUtilization(context.Context, int) ([]dashboarduc.StorageUtilizationRow, error) {
	return nil, nil
}

func TestWarehouseHandlers_DeleteBlocked422(t *testing.T) {
	jwtSvc, err := commonjwt.NewService("wh-hdl-jwt-secret---32bytes-min", time.Hour, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	e := echo.New()
	e.HTTPErrorHandler = httpErrorHandler
	e.Use(commonjwt.RequireBearerAccessJWT(jwtSvc, InventoryPublicPaths))
	h := NewServerHandler(warehouseDelSvc{})
	stub.RegisterHandlers(e, h)

	tok, err := jwtSvc.GenerateAccessToken(commonjwt.ClaimsInput{Subject: "u1", TenantID: uuid.New().String()})
	if err != nil {
		t.Fatal(err)
	}
	wid := uuid.New().String()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/inventory/warehouses/"+wid, nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+tok)
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("want 422 got %d body=%s", rec.Code, rec.Body.String())
	}
}
