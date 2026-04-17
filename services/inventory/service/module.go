package service

import (
	"database/sql"

	"go.uber.org/fx"
)

// Module provides the inventory application Service.
var Module = fx.Module("service",
	fx.Provide(func(db *sql.DB) Service {
		return NewInventoryService(db)
	}),
)
