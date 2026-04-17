package logger

import (
	"context"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

// AppEnvProvider is satisfied by service configs that expose APP_ENV (or equivalent).
type AppEnvProvider interface {
	GetAppEnv() string
}

// New builds a Zap logger: development config when GetAppEnv() == "development", else production.
func New(lc fx.Lifecycle, env AppEnvProvider) (*zap.Logger, error) {
	var (
		log *zap.Logger
		err error
	)
	if env.GetAppEnv() == "development" {
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
