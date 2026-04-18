package service

import (
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/KingWahid/inventory/backend/services/notification/config"
)

// Module provides notification service.
var Module = fx.Module("notification-service",
	fx.Provide(
		func(db *gorm.DB) Service {
			return NewNotificationService(db)
		},
		func(cfg *config.Config, log *zap.Logger) *Dispatcher {
			return NewDispatcher(log, cfg.NotificationWebhookURL)
		},
	),
)
