package service

import (
	"github.com/KingWahid/inventory/backend/pkg/database/transaction"
	audituc "github.com/KingWahid/inventory/backend/services/inventory/domains/audit/usecase"
	cataloguc "github.com/KingWahid/inventory/backend/services/inventory/domains/catalog/usecase"
	dashboarduc "github.com/KingWahid/inventory/backend/services/inventory/domains/dashboard/usecase"
	movementuc "github.com/KingWahid/inventory/backend/services/inventory/domains/movement/usecase"
	warehouseuc "github.com/KingWahid/inventory/backend/services/inventory/domains/warehouse/usecase"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

// Module provides the inventory application Service.
var Module = fx.Module("service",
	fx.Provide(transaction.NewManager),
	fx.Provide(func(db *gorm.DB, txManager transaction.Manager, catalog cataloguc.Usecase, warehouse warehouseuc.Usecase, movement movementuc.Usecase, dashboard dashboarduc.Usecase, audit audituc.Usecase) Service {
		return NewInventoryService(db, txManager, catalog, warehouse, movement, dashboard, audit)
	}),
)
