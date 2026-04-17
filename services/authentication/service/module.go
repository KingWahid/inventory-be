package service

import (
	"github.com/KingWahid/inventory/backend/services/authentication/repository"
	"go.uber.org/fx"
)

// Module provides authentication service.
var Module = fx.Module("authentication-service",
	fx.Provide(repository.New),
	fx.Provide(func(repo repository.Repository) Service {
		return NewAuthenticationService(repo)
	}),
)
