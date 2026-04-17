package config

import "go.uber.org/fx"

// Module provides authentication config.
var Module = fx.Module("authentication-config",
	fx.Provide(New),
)
