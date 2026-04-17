package main

import (
	uberfx "go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"

	"github.com/KingWahid/inventory/backend/services/authentication/config"
	authfx "github.com/KingWahid/inventory/backend/services/authentication/fx"
)

func main() {
	uberfx.New(
		uberfx.WithLogger(func(log *zap.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: log}
		}),
		authfx.Module,
		uberfx.Invoke(func(log *zap.Logger, cfg *config.Config) {
			log.Info("starting authentication-api",
				zap.String("env", cfg.AppEnv),
				zap.String("addr", ":"+cfg.AppPort),
			)
		}),
	).Run()
}
