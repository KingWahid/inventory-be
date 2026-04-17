package service

import (
	"go.uber.org/fx"
	"gorm.io/gorm"
)

// Module provides authentication service.
var Module = fx.Module("authentication-service",
	fx.Provide(func(db *gorm.DB) Service {
		return NewAuthenticationService(db)
	}),
)
