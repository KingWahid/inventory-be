package api

import (
	"context"
	"encoding/json"
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
	"github.com/KingWahid/inventory/backend/services/inventory/stub"
)

// errOrListSvc implements service.Service for category handler tests.
type errOrListSvc struct {
	listOut cataloguc.ListCategoriesOutput
	listErr error
	delErr  error
}

func (e *errOrListSvc) PingDB(context.Context) error { return nil }

func (e *errOrListSvc) ListCategories(_ context.Context, _ cataloguc.ListCategoriesInput) (cataloguc.ListCategoriesOutput, error) {
	return e.listOut, e.listErr
}

func (e *errOrListSvc) GetCategory(context.Context, string) (catalogrepo.Category, error) {
	return catalogrepo.Category{}, errorcodes.ErrNotFound
}

func (e *errOrListSvc) CreateCategory(context.Context, cataloguc.CreateCategoryInput) (catalogrepo.Category, error) {
	return catalogrepo.Category{}, errorcodes.ErrNotFound
}

func (e *errOrListSvc) UpdateCategory(context.Context, string, cataloguc.UpdateCategoryInput) (catalogrepo.Category, error) {
	return catalogrepo.Category{}, errorcodes.ErrNotFound
}

func (e *errOrListSvc) DeleteCategory(context.Context, string) error { return e.delErr }

func (e *errOrListSvc) ListProducts(context.Context, cataloguc.ListProductsInput) (cataloguc.ListProductsOutput, error) {
	return cataloguc.ListProductsOutput{}, nil
}

func (e *errOrListSvc) GetProduct(context.Context, string) (catalogrepo.Product, error) {
	return catalogrepo.Product{}, errorcodes.ErrNotFound
}

func (e *errOrListSvc) CreateProduct(context.Context, cataloguc.CreateProductInput) (catalogrepo.Product, error) {
	return catalogrepo.Product{}, errorcodes.ErrNotFound
}

func (e *errOrListSvc) UpdateProduct(context.Context, string, cataloguc.UpdateProductInput) (catalogrepo.Product, error) {
	return catalogrepo.Product{}, errorcodes.ErrNotFound
}

func (e *errOrListSvc) DeleteProduct(context.Context, string) error { return nil }

func (e *errOrListSvc) RestoreProduct(context.Context, string) (catalogrepo.Product, error) {
	return catalogrepo.Product{}, errorcodes.ErrNotFound
}

func TestCategoryHandlers_ListOK(t *testing.T) {
	jwtSvc, err := commonjwt.NewService("cat-hdl-jwt-secret-32bytes-min", time.Hour, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	tid := uuid.New().String()
	cid := uuid.New().String()
	now := time.Now().UTC()
	svc := &errOrListSvc{
		listOut: cataloguc.ListCategoriesOutput{
			Items: []catalogrepo.Category{{
				ID:        cid,
				TenantID:  tid,
				Name:      "Alpha",
				SortOrder: 0,
				CreatedAt: now,
				UpdatedAt: now,
			}},
			Total:   1,
			Page:    1,
			PerPage: 20,
		},
	}
	e := echo.New()
	e.HTTPErrorHandler = httpErrorHandler
	e.Use(commonjwt.RequireBearerAccessJWT(jwtSvc, InventoryPublicPaths))
	h := NewServerHandler(svc)
	stub.RegisterHandlers(e, h)

	tok, err := jwtSvc.GenerateAccessToken(commonjwt.ClaimsInput{Subject: "u1", TenantID: tid})
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/inventory/categories?page=1&per_page=20", nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+tok)
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200 got %d body=%s", rec.Code, rec.Body.String())
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	if string(raw["success"]) != "true" {
		t.Fatalf("want success true: %s", rec.Body.String())
	}
}

func TestCategoryHandlers_DeleteBlocked422(t *testing.T) {
	jwtSvc, err := commonjwt.NewService("cat-hdl-jwt-secret-32bytes-min", time.Hour, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	e := echo.New()
	e.HTTPErrorHandler = httpErrorHandler
	e.Use(commonjwt.RequireBearerAccessJWT(jwtSvc, InventoryPublicPaths))
	h := NewServerHandler(&errOrListSvc{
		delErr: errorcodes.ErrCategoryHasActiveProducts.WithDetails(map[string]any{"category_id": "x"}),
	})
	stub.RegisterHandlers(e, h)

	tok, err := jwtSvc.GenerateAccessToken(commonjwt.ClaimsInput{Subject: "u1", TenantID: uuid.New().String()})
	if err != nil {
		t.Fatal(err)
	}
	catPath := uuid.New().String()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/inventory/categories/"+catPath, nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+tok)
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("want 422 got %d body=%s", rec.Code, rec.Body.String())
	}
}
