package service

import (
	"context"
	"fmt"

	"github.com/KingWahid/inventory/backend/pkg/database/transaction"
	audituc "github.com/KingWahid/inventory/backend/services/inventory/domains/audit/usecase"
	cataloguc "github.com/KingWahid/inventory/backend/services/inventory/domains/catalog/usecase"
	dashboarduc "github.com/KingWahid/inventory/backend/services/inventory/domains/dashboard/usecase"
	movementuc "github.com/KingWahid/inventory/backend/services/inventory/domains/movement/usecase"
	warehouseuc "github.com/KingWahid/inventory/backend/services/inventory/domains/warehouse/usecase"
	"gorm.io/gorm"
)

// InventoryService implements Service using PostgreSQL.
type InventoryService struct {
	db        *gorm.DB
	txManager transaction.Manager
	catalog   cataloguc.Usecase
	warehouse warehouseuc.Usecase
	movement  movementuc.Usecase
	dashboard dashboarduc.Usecase
	audit     audituc.Usecase
}

// NewInventoryService constructs the default inventory application service.
func NewInventoryService(db *gorm.DB, txManager transaction.Manager, catalog cataloguc.Usecase, warehouse warehouseuc.Usecase, movement movementuc.Usecase, dashboard dashboarduc.Usecase, audit audituc.Usecase) *InventoryService {
	return &InventoryService{
		db:        db,
		txManager: txManager,
		catalog:   catalog,
		warehouse: warehouse,
		movement:  movement,
		dashboard: dashboard,
		audit:     audit,
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
