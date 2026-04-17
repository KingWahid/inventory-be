package config

import "go.uber.org/fx"

// Module exports config loading for fx.
var Module = fx.Module("config",
	fx.Provide(New),
)
