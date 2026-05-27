package postgres

import (
	"context"

	"go.uber.org/fx"
	"gorm.io/gorm"
)

// FxModule registers a *gorm.DB from any provided DBConfig.
func FxModule() fx.Option {
	return fx.Module("postgres",
		fx.Provide(func(lc fx.Lifecycle, cfg DBConfig) (*gorm.DB, error) {
			db, err := Open(cfg.GetDBDSN())
			if err != nil {
				return nil, err
			}

			lc.Append(fx.Hook{
				OnStop: func(_ context.Context) error {
					sqlDB, err := db.DB()
					if err != nil {
						return err
					}
					return sqlDB.Close()
				},
			})

			return db, nil
		}),
	)
}
