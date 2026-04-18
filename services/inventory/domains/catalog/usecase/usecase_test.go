package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/KingWahid/inventory/backend/pkg/common/errorcodes"
	commonjwt "github.com/KingWahid/inventory/backend/pkg/common/jwt"
	"github.com/KingWahid/inventory/backend/services/inventory/domains/catalog/repository"
)

type fakeRepo struct {
	getByIDFunc func(ctx context.Context, tenantID, id string) (repository.Category, error)
	count       int64
	countErr    error
	deleteCalls int

	hasStock               bool
	softDeleteProductCalls int
}

func (f *fakeRepo) Ping() error { return nil }

func (f *fakeRepo) List(context.Context, string, repository.ListFilter) ([]repository.Category, int64, error) {
	return nil, 0, errors.New("not implemented")
}

func (f *fakeRepo) GetByID(ctx context.Context, tenantID, id string) (repository.Category, error) {
	if f.getByIDFunc != nil {
		return f.getByIDFunc(ctx, tenantID, id)
	}
	return repository.Category{
		ID:        id,
		TenantID:  tenantID,
		Name:      "x",
		SortOrder: 0,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}, nil
}

func (f *fakeRepo) Create(context.Context, string, repository.CreateInput) (repository.Category, error) {
	return repository.Category{}, errors.New("not implemented")
}

func (f *fakeRepo) Update(context.Context, string, string, repository.UpdateInput) (repository.Category, error) {
	return repository.Category{}, errors.New("not implemented")
}

func (f *fakeRepo) SoftDelete(context.Context, string, string) error {
	f.deleteCalls++
	return nil
}

func (f *fakeRepo) CountActiveProductsByCategoryID(context.Context, string, string) (int64, error) {
	return f.count, f.countErr
}

func (f *fakeRepo) ListProducts(context.Context, string, repository.ListProductsFilter) ([]repository.Product, int64, error) {
	return nil, 0, errors.New("not implemented")
}

func (f *fakeRepo) GetProductByID(_ context.Context, tenantID, id string) (repository.Product, error) {
	return repository.Product{
		ID:        id,
		TenantID:  tenantID,
		SKU:       "SKU",
		Name:      "Product",
		Unit:      "pcs",
		Metadata:  json.RawMessage("{}"),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}, nil
}

func (f *fakeRepo) CreateProduct(context.Context, string, repository.CreateProductInput) (repository.Product, error) {
	return repository.Product{}, errors.New("not implemented")
}

func (f *fakeRepo) UpdateProduct(context.Context, string, string, repository.UpdateProductInput) (repository.Product, error) {
	return repository.Product{}, errors.New("not implemented")
}

func (f *fakeRepo) SoftDeleteProduct(context.Context, string, string) error {
	f.softDeleteProductCalls++
	return nil
}

func (f *fakeRepo) RestoreProduct(context.Context, string, string) (repository.Product, error) {
	return repository.Product{}, errors.New("not implemented")
}

func (f *fakeRepo) HasPositiveStock(context.Context, string, string) (bool, error) {
	return f.hasStock, nil
}

func ctxWithTenant(tenant string) context.Context {
	return commonjwt.ContextWithClaims(context.Background(), &commonjwt.Claims{TenantID: tenant})
}

func TestDeleteCategory_RejectsWhenActiveProducts(t *testing.T) {
	catID := uuid.New().String()
	fr := &fakeRepo{count: 3}
	u := New(fr, nil)
	err := u.DeleteCategory(ctxWithTenant("tenant-a"), catID)
	if !errors.Is(err, errorcodes.ErrCategoryHasActiveProducts) {
		t.Fatalf("want ErrCategoryHasActiveProducts got %v", err)
	}
	if fr.deleteCalls != 0 {
		t.Fatalf("SoftDelete should not run, calls=%d", fr.deleteCalls)
	}
}

func TestDeleteCategory_SoftDeletesWhenNoProducts(t *testing.T) {
	catID := uuid.New().String()
	fr := &fakeRepo{count: 0}
	u := New(fr, nil)
	if err := u.DeleteCategory(ctxWithTenant("tenant-a"), catID); err != nil {
		t.Fatal(err)
	}
	if fr.deleteCalls != 1 {
		t.Fatalf("want 1 SoftDelete got %d", fr.deleteCalls)
	}
}

func TestDeleteCategory_TenantMissing(t *testing.T) {
	u := New(&fakeRepo{}, nil)
	err := u.DeleteCategory(context.Background(), uuid.New().String())
	if !errors.Is(err, errorcodes.ErrTenantContextMissing) {
		t.Fatalf("want ErrTenantContextMissing got %v", err)
	}
}

func TestDeleteProduct_RejectsWhenStock(t *testing.T) {
	pid := uuid.New().String()
	fr := &fakeRepo{hasStock: true}
	u := New(fr, nil)
	err := u.DeleteProduct(ctxWithTenant("tenant-a"), pid)
	if !errors.Is(err, errorcodes.ErrProductHasStock) {
		t.Fatalf("want ErrProductHasStock got %v", err)
	}
	if fr.softDeleteProductCalls != 0 {
		t.Fatalf("SoftDeleteProduct should not run")
	}
}

func TestDeleteProduct_SoftDeletesWhenNoStock(t *testing.T) {
	pid := uuid.New().String()
	fr := &fakeRepo{hasStock: false}
	u := New(fr, nil)
	if err := u.DeleteProduct(ctxWithTenant("tenant-a"), pid); err != nil {
		t.Fatal(err)
	}
	if fr.softDeleteProductCalls != 1 {
		t.Fatalf("want 1 SoftDeleteProduct got %d", fr.softDeleteProductCalls)
	}
}

func TestDeleteProduct_TenantMissing(t *testing.T) {
	u := New(&fakeRepo{}, nil)
	err := u.DeleteProduct(context.Background(), uuid.New().String())
	if !errors.Is(err, errorcodes.ErrTenantContextMissing) {
		t.Fatalf("want ErrTenantContextMissing got %v", err)
	}
}
