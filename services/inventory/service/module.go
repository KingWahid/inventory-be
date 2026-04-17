package service

import (
	"go.uber.org/fx"
	"gorm.io/gorm"
)

// Module provides the inventory application Service.
var Module = fx.Module("service",
	fx.Provide(func(db *gorm.DB) Service {
		return NewInventoryService(db)
	}),
)
