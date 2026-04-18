package api

import (
	"github.com/KingWahid/inventory/backend/services/inventory/config"
	"go.uber.org/fx"
)

// EchoModule provides *echo.Echo without route registration (routes: see inventory/fx handler).
var EchoModule = fx.Module("api-echo",
	fx.Provide(config.NewJWTService),
	fx.Provide(NewEcho),
)
