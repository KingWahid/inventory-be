package service

import (
	"github.com/your-org/inventory/backend/pkg/database/transaction"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

// Module provides the inventory application Service.
var Module = fx.Module("service",
	fx.Provide(transaction.NewManager),
	fx.Provide(func(db *gorm.DB, txManager transaction.Manager) Service {
		return NewInventoryService(db, txManager)
	}),
)
