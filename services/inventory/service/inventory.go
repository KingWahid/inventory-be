package service

import (
	"context"
	"database/sql"
)

// InventoryService implements Service using PostgreSQL.
type InventoryService struct {
	db *sql.DB
}

// NewInventoryService constructs the default inventory application service.
func NewInventoryService(db *sql.DB) *InventoryService {
	return &InventoryService{db: db}
}

// PingDB checks database connectivity (readiness).
func (s *InventoryService) PingDB(ctx context.Context) error {
	return s.db.PingContext(ctx)
}
