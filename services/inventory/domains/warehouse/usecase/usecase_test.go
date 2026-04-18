package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/KingWahid/inventory/backend/pkg/common/errorcodes"
	commonjwt "github.com/KingWahid/inventory/backend/pkg/common/jwt"
	"github.com/KingWahid/inventory/backend/services/inventory/domains/warehouse/repository"
)

type fakeRepo struct {
	hasStock        bool
	softDeleteCalls int
	getByIDErr      error
}

func (f *fakeRepo) Ping() error { return nil }

func (f *fakeRepo) List(context.Context, string, repository.ListFilter) ([]repository.Warehouse, int64, error) {
	return nil, 0, errors.New("not implemented")
}

func (f *fakeRepo) GetByID(context.Context, string, string) (repository.Warehouse, error) {
	if f.getByIDErr != nil {
		return repository.Warehouse{}, f.getByIDErr
	}
	return repository.Warehouse{
		ID:        uuid.New().String(),
		TenantID:  "t",
		Code:      "WH",
		Name:      "Main",
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}, nil
}

func (f *fakeRepo) Create(context.Context, string, repository.CreateInput) (repository.Warehouse, error) {
	return repository.Warehouse{}, errors.New("not implemented")
}

func (f *fakeRepo) Update(context.Context, string, string, repository.UpdateInput) (repository.Warehouse, error) {
	return repository.Warehouse{}, errors.New("not implemented")
}

func (f *fakeRepo) SoftDelete(context.Context, string, string) error {
	f.softDeleteCalls++
	return nil
}

func (f *fakeRepo) HasPositiveStock(context.Context, string, string) (bool, error) {
	return f.hasStock, nil
}

func ctxTenant(tenant string) context.Context {
	return commonjwt.ContextWithClaims(context.Background(), &commonjwt.Claims{TenantID: tenant})
}

func TestDeleteWarehouse_RejectsWhenStock(t *testing.T) {
	wid := uuid.New().String()
	fr := &fakeRepo{hasStock: true}
	u := New(fr, nil)
	err := u.DeleteWarehouse(ctxTenant("tenant-a"), wid)
	if !errors.Is(err, errorcodes.ErrWarehouseStock) {
		t.Fatalf("want ErrWarehouseStock got %v", err)
	}
	if fr.softDeleteCalls != 0 {
		t.Fatal("SoftDelete should not run")
	}
}

func TestDeleteWarehouse_SoftDeletesWhenNoStock(t *testing.T) {
	wid := uuid.New().String()
	fr := &fakeRepo{hasStock: false}
	u := New(fr, nil)
	if err := u.DeleteWarehouse(ctxTenant("tenant-a"), wid); err != nil {
		t.Fatal(err)
	}
	if fr.softDeleteCalls != 1 {
		t.Fatalf("want 1 SoftDelete got %d", fr.softDeleteCalls)
	}
}

func TestDeleteWarehouse_TenantMissing(t *testing.T) {
	u := New(&fakeRepo{}, nil)
	err := u.DeleteWarehouse(context.Background(), uuid.New().String())
	if !errors.Is(err, errorcodes.ErrTenantContextMissing) {
		t.Fatalf("want ErrTenantContextMissing got %v", err)
	}
}
