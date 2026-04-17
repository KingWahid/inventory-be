package postgres

import (
	"database/sql"
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// OpenGORM wraps an existing *sql.DB pool (e.g. from NewDB). Do not call AutoMigrate here—use goose migrations.
func OpenGORM(db *sql.DB) (*gorm.DB, error) {
	gdb, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("postgres: gorm open: %w", err)
	}
	return gdb, nil
}
