package api

import (
	"fmt"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	warehouserepo "github.com/KingWahid/inventory/backend/services/inventory/domains/warehouse/repository"
	"github.com/KingWahid/inventory/backend/services/inventory/stub"
)

func warehouseRepoToStub(w warehouserepo.Warehouse) (stub.Warehouse, error) {
	id, err := uuid.Parse(w.ID)
	if err != nil {
		return stub.Warehouse{}, fmt.Errorf("warehouse id: %w", err)
	}
	tid, err := uuid.Parse(w.TenantID)
	if err != nil {
		return stub.Warehouse{}, fmt.Errorf("tenant id: %w", err)
	}
	code := w.Code
	ia := w.IsActive
	return stub.Warehouse{
		Id:        openapi_types.UUID(id),
		TenantId:  openapi_types.UUID(tid),
		Code:      &code,
		Name:      w.Name,
		Address:   w.Address,
		IsActive:  &ia,
		CreatedAt: w.CreatedAt,
		UpdatedAt: w.UpdatedAt,
		DeletedAt: w.DeletedAt,
	}, nil
}
