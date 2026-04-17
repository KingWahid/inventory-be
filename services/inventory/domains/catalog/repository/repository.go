package repository

import "gorm.io/gorm"

// Repository defines data-access contract for catalog domain.
type Repository interface {
	Ping() error
}

type repository struct {
	db *gorm.DB
}

// New creates catalog repository implementation.
func New(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Ping() error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}
