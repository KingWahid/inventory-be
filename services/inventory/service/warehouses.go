package service

import (
	"context"

	warehouserepo "github.com/KingWahid/inventory/backend/services/inventory/domains/warehouse/repository"
	warehouseuc "github.com/KingWahid/inventory/backend/services/inventory/domains/warehouse/usecase"
)

func (s *InventoryService) ListWarehouses(ctx context.Context, in warehouseuc.ListWarehousesInput) (warehouseuc.ListWarehousesOutput, error) {
	return s.warehouse.ListWarehouses(ctx, in)
}

func (s *InventoryService) GetWarehouse(ctx context.Context, warehouseID string) (warehouserepo.Warehouse, error) {
	return s.warehouse.GetWarehouse(ctx, warehouseID)
}

func (s *InventoryService) CreateWarehouse(ctx context.Context, in warehouseuc.CreateWarehouseInput) (warehouserepo.Warehouse, error) {
	return s.warehouse.CreateWarehouse(ctx, in)
}

func (s *InventoryService) UpdateWarehouse(ctx context.Context, warehouseID string, in warehouseuc.UpdateWarehouseInput) (warehouserepo.Warehouse, error) {
	return s.warehouse.UpdateWarehouse(ctx, warehouseID, in)
}

func (s *InventoryService) DeleteWarehouse(ctx context.Context, warehouseID string) error {
	return s.warehouse.DeleteWarehouse(ctx, warehouseID)
}
