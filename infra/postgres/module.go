package postgres

import (
	"database/sql"

	"go.uber.org/fx"
)

// DBConfig is satisfied by service configs that expose a Postgres DSN (e.g. DB_DSN).
type DBConfig interface {
	GetDBDSN() string
}

// FxModule registers *sql.DB and *gorm.DB (same pool) from any provided DBConfig.
func FxModule() fx.Option {
	return fx.Module("postgres",
		fx.Provide(
			func(lc fx.Lifecycle, cfg DBConfig) (*sql.DB, error) {
				return NewDB(lc, cfg.GetDBDSN())
			},
			OpenGORM,
		),
	)
}
