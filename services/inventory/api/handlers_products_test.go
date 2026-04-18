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
	warehouserepo "github.com/KingWahid/inventory/backend/services/inventory/domains/warehouse/repository"
	warehouseuc "github.com/KingWahid/inventory/backend/services/inventory/domains/warehouse/usecase"
	"github.com/KingWahid/inventory/backend/services/inventory/stub"
)

type productDelSvc struct {
	movementEmbedNoop
}

func (productDelSvc) PingDB(context.Context) error { return nil }

func (productDelSvc) ListCategories(context.Context, cataloguc.ListCategoriesInput) (cataloguc.ListCategoriesOutput, error) {
	return cataloguc.ListCategoriesOutput{}, errorcodes.ErrNotFound
}

func (productDelSvc) GetCategory(context.Context, string) (catalogrepo.Category, error) {
	return catalogrepo.Category{}, errorcodes.ErrNotFound
}

func (productDelSvc) CreateCategory(context.Context, cataloguc.CreateCategoryInput) (catalogrepo.Category, error) {
	return catalogrepo.Category{}, errorcodes.ErrNotFound
}

func (productDelSvc) UpdateCategory(context.Context, string, cataloguc.UpdateCategoryInput) (catalogrepo.Category, error) {
	return catalogrepo.Category{}, errorcodes.ErrNotFound
}

func (productDelSvc) DeleteCategory(context.Context, string) error { return nil }

func (productDelSvc) ListProducts(context.Context, cataloguc.ListProductsInput) (cataloguc.ListProductsOutput, error) {
	return cataloguc.ListProductsOutput{}, nil
}

func (productDelSvc) GetProduct(context.Context, string) (catalogrepo.Product, error) {
	return catalogrepo.Product{}, errorcodes.ErrNotFound
}

func (productDelSvc) CreateProduct(context.Context, cataloguc.CreateProductInput) (catalogrepo.Product, error) {
	return catalogrepo.Product{}, errorcodes.ErrNotFound
}

func (productDelSvc) UpdateProduct(context.Context, string, cataloguc.UpdateProductInput) (catalogrepo.Product, error) {
	return catalogrepo.Product{}, errorcodes.ErrNotFound
}

func (productDelSvc) DeleteProduct(context.Context, string) error {
	return errorcodes.ErrProductHasStock.WithDetails(map[string]any{"product_id": "x"})
}

func (productDelSvc) RestoreProduct(context.Context, string) (catalogrepo.Product, error) {
	return catalogrepo.Product{}, errorcodes.ErrNotFound
}

func (productDelSvc) ListWarehouses(context.Context, warehouseuc.ListWarehousesInput) (warehouseuc.ListWarehousesOutput, error) {
	return warehouseuc.ListWarehousesOutput{}, nil
}

func (productDelSvc) GetWarehouse(context.Context, string) (warehouserepo.Warehouse, error) {
	return warehouserepo.Warehouse{}, errorcodes.ErrNotFound
}

func (productDelSvc) CreateWarehouse(context.Context, warehouseuc.CreateWarehouseInput) (warehouserepo.Warehouse, error) {
	return warehouserepo.Warehouse{}, errorcodes.ErrNotFound
}

func (productDelSvc) UpdateWarehouse(context.Context, string, warehouseuc.UpdateWarehouseInput) (warehouserepo.Warehouse, error) {
	return warehouserepo.Warehouse{}, errorcodes.ErrNotFound
}

func (productDelSvc) DeleteWarehouse(context.Context, string) error { return nil }

func TestProductHandlers_DeleteBlocked422(t *testing.T) {
	jwtSvc, err := commonjwt.NewService("prod-hdl-jwt-secret--32bytes-min", time.Hour, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	e := echo.New()
	e.HTTPErrorHandler = httpErrorHandler
	e.Use(commonjwt.RequireBearerAccessJWT(jwtSvc, InventoryPublicPaths))
	h := NewServerHandler(productDelSvc{})
	stub.RegisterHandlers(e, h)

	tok, err := jwtSvc.GenerateAccessToken(commonjwt.ClaimsInput{Subject: "u1", TenantID: uuid.New().String()})
	if err != nil {
		t.Fatal(err)
	}
	pid := uuid.New().String()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/inventory/products/"+pid, nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+tok)
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("want 422 got %d body=%s", rec.Code, rec.Body.String())
	}
}
