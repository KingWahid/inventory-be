package api

import (
	"go.uber.org/fx"
)

// EchoModule provides Echo HTTP server.
var EchoModule = fx.Module("authentication-api",
	fx.Provide(NewEcho),
)
