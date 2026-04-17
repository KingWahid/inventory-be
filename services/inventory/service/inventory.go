package service

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// InventoryService implements Service using PostgreSQL.
type InventoryService struct {
	db *gorm.DB
}

// NewInventoryService constructs the default inventory application service.
func NewInventoryService(db *gorm.DB) *InventoryService {
	return &InventoryService{db: db}
}

// PingDB checks database connectivity (readiness).
func (s *InventoryService) PingDB(ctx context.Context) error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return fmt.Errorf("inventory service: sql db: %w", err)
	}
	return sqlDB.PingContext(ctx)
}
