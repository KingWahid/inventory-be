package config

import "go.uber.org/fx"

// Module provides notification config.
var Module = fx.Module("notification-config",
	fx.Provide(New),
)
