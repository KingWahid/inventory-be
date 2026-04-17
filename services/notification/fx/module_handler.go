package fx

import (
	"github.com/KingWahid/inventory/backend/services/notification/api"
	"go.uber.org/fx"
)

// HandlerModule wires generated OpenAPI handlers.
var HandlerModule = fx.Module("notification-handler",
	fx.Provide(api.NewServerHandler),
	fx.Invoke(RegisterRoutes),
)
