package service

import (
	"context"
	"fmt"

	"github.com/KingWahid/inventory/backend/pkg/database/transaction"
	"gorm.io/gorm"
)

// InventoryService implements Service using PostgreSQL.
type InventoryService struct {
	db        *gorm.DB
	txManager transaction.Manager
}

// NewInventoryService constructs the default inventory application service.
func NewInventoryService(db *gorm.DB, txManager transaction.Manager) *InventoryService {
	return &InventoryService{
		db:        db,
		txManager: txManager,
	}
}

// PingDB checks database connectivity (readiness).
func (s *InventoryService) PingDB(ctx context.Context) error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return fmt.Errorf("inventory service: sql db: %w", err)
	}
	return sqlDB.PingContext(ctx)
}
