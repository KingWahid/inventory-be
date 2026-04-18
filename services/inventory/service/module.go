package service

import (
	"github.com/KingWahid/inventory/backend/pkg/database/transaction"
	cataloguc "github.com/KingWahid/inventory/backend/services/inventory/domains/catalog/usecase"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

// Module provides the inventory application Service.
var Module = fx.Module("service",
	fx.Provide(transaction.NewManager),
	fx.Provide(func(db *gorm.DB, txManager transaction.Manager, catalog cataloguc.Usecase) Service {
		return NewInventoryService(db, txManager, catalog)
	}),
)
