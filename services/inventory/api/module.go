package api

import "go.uber.org/fx"

// EchoModule provides *echo.Echo without route registration (routes: see inventory/fx handler).
var EchoModule = fx.Module("api-echo",
	fx.Provide(NewEcho),
)
