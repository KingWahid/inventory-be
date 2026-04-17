package warehouse

import (
	"go.uber.org/fx"

	"github.com/KingWahid/inventory/backend/services/inventory/domains/warehouse/handler"
	"github.com/KingWahid/inventory/backend/services/inventory/domains/warehouse/repository"
	"github.com/KingWahid/inventory/backend/services/inventory/domains/warehouse/usecase"
)

// Module wires warehouse domain dependencies.
var Module = fx.Module("warehouse",
	fx.Provide(
		repository.New,
		usecase.New,
		handler.New,
	),
)
