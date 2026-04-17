package fx

import (
	"github.com/KingWahid/inventory/backend/services/authentication/api"
	"go.uber.org/fx"
)

// HandlerModule wires generated OpenAPI handlers.
var HandlerModule = fx.Module("authentication-handler",
	fx.Provide(api.NewServerHandler),
	fx.Invoke(RegisterRoutes),
)
