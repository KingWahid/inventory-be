package service

import (
	"context"

	audituc "github.com/KingWahid/inventory/backend/services/inventory/domains/audit/usecase"
	catalogrepo "github.com/KingWahid/inventory/backend/services/inventory/domains/catalog/repository"
	cataloguc "github.com/KingWahid/inventory/backend/services/inventory/domains/catalog/usecase"
	dashboarduc "github.com/KingWahid/inventory/backend/services/inventory/domains/dashboard/usecase"
	movrepo "github.com/KingWahid/inventory/backend/services/inventory/domains/movement/repository"
	movementuc "github.com/KingWahid/inventory/backend/services/inventory/domains/movement/usecase"
	warehouserepo "github.com/KingWahid/inventory/backend/services/inventory/domains/warehouse/repository"
	warehouseuc "github.com/KingWahid/inventory/backend/services/inventory/domains/warehouse/usecase"
)

// Service is the application facade used by HTTP handlers (expand per domain module).
type Service interface {
	PingDB(ctx context.Context) error

	ListCategories(ctx context.Context, in cataloguc.ListCategoriesInput) (cataloguc.ListCategoriesOutput, error)
	GetCategory(ctx context.Context, categoryID string) (catalogrepo.Category, error)
	CreateCategory(ctx context.Context, in cataloguc.CreateCategoryInput) (catalogrepo.Category, error)
	UpdateCategory(ctx context.Context, categoryID string, in cataloguc.UpdateCategoryInput) (catalogrepo.Category, error)
	DeleteCategory(ctx context.Context, categoryID string) error

	ListProducts(ctx context.Context, in cataloguc.ListProductsInput) (cataloguc.ListProductsOutput, error)
	GetProduct(ctx context.Context, productID string) (catalogrepo.Product, error)
	CreateProduct(ctx context.Context, in cataloguc.CreateProductInput) (catalogrepo.Product, error)
	UpdateProduct(ctx context.Context, productID string, in cataloguc.UpdateProductInput) (catalogrepo.Product, error)
	DeleteProduct(ctx context.Context, productID string) error
	RestoreProduct(ctx context.Context, productID string) (catalogrepo.Product, error)

	ListWarehouses(ctx context.Context, in warehouseuc.ListWarehousesInput) (warehouseuc.ListWarehousesOutput, error)
	GetWarehouse(ctx context.Context, warehouseID string) (warehouserepo.Warehouse, error)
	CreateWarehouse(ctx context.Context, in warehouseuc.CreateWarehouseInput) (warehouserepo.Warehouse, error)
	UpdateWarehouse(ctx context.Context, warehouseID string, in warehouseuc.UpdateWarehouseInput) (warehouserepo.Warehouse, error)
	DeleteWarehouse(ctx context.Context, warehouseID string) error

	CreateInbound(ctx context.Context, destinationWarehouseID string, in movementuc.CreateMovementBase) (movrepo.Movement, error)
	CreateOutbound(ctx context.Context, sourceWarehouseID string, in movementuc.CreateMovementBase) (movrepo.Movement, error)
	CreateTransfer(ctx context.Context, sourceWarehouseID, destinationWarehouseID string, in movementuc.CreateMovementBase) (movrepo.Movement, error)
	CreateAdjustment(ctx context.Context, sourceWarehouseID, destinationWarehouseID *string, in movementuc.CreateMovementBase) (movrepo.Movement, error)
	GetMovement(ctx context.Context, movementID string) (movrepo.Movement, error)
	ListMovements(ctx context.Context, in movementuc.ListMovementsInput) (movementuc.ListMovementsOutput, error)
	ConfirmMovement(ctx context.Context, movementID string) (movrepo.Movement, error)
	CancelMovement(ctx context.Context, movementID string) (movrepo.Movement, error)

	ListAuditLogs(ctx context.Context, in audituc.ListAuditLogsInput) (audituc.ListAuditLogsOutput, error)

	GetDashboardSummary(ctx context.Context) (dashboarduc.Summary, error)
	GetDashboardMovementsChart(ctx context.Context, period string) (dashboarduc.MovementChart, error)
}
