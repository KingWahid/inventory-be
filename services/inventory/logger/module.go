package logger

import "go.uber.org/fx"

// Module exports the Zap logger for fx.
var Module = fx.Module("logger",
	fx.Provide(New),
)
