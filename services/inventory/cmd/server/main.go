package main

import (
	uberfx "go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"

	"github.com/KingWahid/inventory/backend/services/inventory/config"
	invfx "github.com/KingWahid/inventory/backend/services/inventory/fx"
)

func main() {
	uberfx.New(
		uberfx.WithLogger(func(log *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: log}
		}),
		invfx.Module,
		uberfx.Invoke(func(log *zap.Logger, cfg *config.Config) {
			log.Info("starting inventory-api",
				zap.String("env", cfg.AppEnv),
				zap.String("addr", ":"+cfg.AppPort),
			)
		}),
	).Run()
}
