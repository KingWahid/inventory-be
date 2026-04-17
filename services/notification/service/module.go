package service

import (
	"go.uber.org/fx"
	"gorm.io/gorm"
)

// Module provides notification service.
var Module = fx.Module("notification-service",
	fx.Provide(func(db *gorm.DB) Service {
		return NewNotificationService(db)
	}),
)
