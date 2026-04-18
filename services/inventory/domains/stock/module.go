package stock

import (
	"go.uber.org/fx"

	"github.com/KingWahid/inventory/backend/services/inventory/domains/stock/repository"
)

// Module wires stock balance repository (no HTTP surface until movement API exists).
var Module = fx.Module("stock",
	fx.Provide(repository.New),
)
