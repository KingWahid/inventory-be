package postgres

import (
	"database/sql"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DBConfig is satisfied by service configs that expose a Postgres DSN.
type DBConfig interface {
	GetDBDSN() string
}

// OpenGORM wraps an existing *sql.DB with GORM.
func OpenGORM(sqlDB *sql.DB) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.New(postgres.Config{
		Conn: sqlDB,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}
	return db, nil
}

// Open opens a new *sql.DB and wraps it with GORM using the given DSN.
func Open(dsn string) (*gorm.DB, error) {
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	return OpenGORM(sqlDB)
}
