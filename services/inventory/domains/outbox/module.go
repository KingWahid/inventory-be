package outbox

import (
	"go.uber.org/fx"

	"github.com/KingWahid/inventory/backend/services/inventory/domains/outbox/repository"
)

// Module wires outbox_events persistence.
var Module = fx.Module("outbox",
	fx.Provide(repository.New),
)
