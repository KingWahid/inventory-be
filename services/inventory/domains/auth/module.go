package auth

import (
	"go.uber.org/fx"

	"github.com/KingWahid/inventory/backend/services/inventory/domains/auth/handler"
	"github.com/KingWahid/inventory/backend/services/inventory/domains/auth/repository"
	"github.com/KingWahid/inventory/backend/services/inventory/domains/auth/usecase"
)

// Module wires auth domain dependencies.
var Module = fx.Module("auth",
	fx.Provide(
		repository.New,
		usecase.New,
		handler.New,
	),
)
