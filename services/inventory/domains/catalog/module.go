package catalog

import (
	"go.uber.org/fx"

	"github.com/KingWahid/inventory/backend/services/inventory/domains/catalog/handler"
	"github.com/KingWahid/inventory/backend/services/inventory/domains/catalog/repository"
	"github.com/KingWahid/inventory/backend/services/inventory/domains/catalog/usecase"
)

// Module wires catalog domain dependencies.
var Module = fx.Module("catalog",
	fx.Provide(
		repository.New,
		usecase.New,
		handler.New,
	),
)
