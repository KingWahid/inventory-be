package base

import (
	"context"

	"github.com/KingWahid/inventory/backend/pkg/common/jwt"
	"gorm.io/gorm"
)

// TenantDB scopes a GORM chain to the current tenant from ctx (JWT claims).
// Cross-tenant admin paths must not use this (see ARCHITECTURE §6).
func TenantDB(ctx context.Context, db *gorm.DB) (*gorm.DB, error) {
	tid, err := jwt.TenantIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	return db.Where("tenant_id = ?", tid), nil
}
