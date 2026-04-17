package audit

import (
	"go.uber.org/fx"

	"github.com/KingWahid/inventory/backend/services/inventory/domains/audit/handler"
	"github.com/KingWahid/inventory/backend/services/inventory/domains/audit/repository"
	"github.com/KingWahid/inventory/backend/services/inventory/domains/audit/usecase"
)

// Module wires audit domain dependencies.
var Module = fx.Module("audit",
	fx.Provide(
		repository.New,
		usecase.New,
		handler.New,
	),
)
