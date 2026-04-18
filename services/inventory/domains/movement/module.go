package movement

import (
	"go.uber.org/fx"

	"github.com/KingWahid/inventory/backend/services/inventory/domains/movement/repository"
	"github.com/KingWahid/inventory/backend/services/inventory/domains/movement/usecase"
)

// Module wires movement domain dependencies (HTTP uses api.ServerHandler + OpenAPI stub).
var Module = fx.Module("movement",
	fx.Provide(
		repository.New,
		usecase.New,
	),
)
