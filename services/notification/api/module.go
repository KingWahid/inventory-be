package api

import "go.uber.org/fx"

// EchoModule provides Echo HTTP server.
var EchoModule = fx.Module("notification-api",
	fx.Provide(NewEcho),
)
