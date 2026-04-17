package logger

import (
	"context"

	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/your-org/inventory/backend/services/inventory/config"
)

// New builds a Zap logger from APP_ENV (development → development config, else production).
func New(lc fx.Lifecycle, cfg *config.Config) (*zap.Logger, error) {
	var (
		log *zap.Logger
		err error
	)
	if cfg.AppEnv == "development" {
		log, err = zap.NewDevelopment()
	} else {
		log, err = zap.NewProduction()
	}
	if err != nil {
		return nil, err
	}

	_ = zap.ReplaceGlobals(log)

	lc.Append(fx.Hook{
		OnStop: func(context.Context) error {
			_ = log.Sync()
			return nil
		},
	})

	return log, nil
}
