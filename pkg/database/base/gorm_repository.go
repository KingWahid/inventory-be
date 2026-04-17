package base

import (
	"context"

	"gorm.io/gorm"
)

// GormRepository provides shared helpers for repositories backed by gorm.DB.
type GormRepository struct {
	db *gorm.DB
}

// NewGormRepository constructs a reusable gorm-backed base repository.
func NewGormRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

// DB exposes the underlying gorm.DB for repository query methods.
func (r *GormRepository) DB() *gorm.DB {
	return r.db
}

// PingDB checks connectivity using the shared SQL pool under gorm.DB.
func (r *GormRepository) PingDB(ctx context.Context) error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}
