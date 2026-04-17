package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/fx"
)

// NewDB opens a PostgreSQL connection pool using pgx via database/sql.
func NewDB(lc fx.Lifecycle, dsn string) (*sql.DB, error) {
	if dsn == "" {
		return nil, errors.New("postgres: DB_DSN is required")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres: open: %w", err)
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return db.PingContext(ctx)
		},
		OnStop: func(ctx context.Context) error {
			return db.Close()
		},
	})

	return db, nil
}
