package repository

import (
	"context"
	"strings"

	"github.com/KingWahid/inventory/backend/pkg/common/errorcodes"
	"github.com/KingWahid/inventory/backend/pkg/database/base"
	"github.com/KingWahid/inventory/backend/pkg/database/schemas"
	"gorm.io/gorm"
)

// Repository is authentication data-access contract.
type Repository interface {
	PingDB(ctx context.Context) error
	CreateTenantAdmin(ctx context.Context, in schemas.CreateTenantAdminInput) (schemas.CreateTenantAdminResult, error)
	FindUserCredentialByEmail(ctx context.Context, email string) (schemas.AuthUserCredential, error)
}

type repository struct {
	*base.GormRepository
}

// New creates authentication repository implementation.
func New(db *gorm.DB) Repository {
	return &repository{
		GormRepository: base.NewGormRepository(db),
	}
}

func (r *repository) CreateTenantAdmin(ctx context.Context, in schemas.CreateTenantAdminInput) (schemas.CreateTenantAdminResult, error) {
	var tenantID string
	var userID string

	err := r.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Raw(
			`INSERT INTO tenants (name, slug, is_active, settings) VALUES (?, ?, true, '{}'::jsonb) RETURNING id`,
			strings.TrimSpace(in.TenantName),
			strings.TrimSpace(strings.ToLower(in.TenantSlug)),
		).Scan(&tenantID).Error; err != nil {
			return err
		}

		if err := tx.Raw(
			`INSERT INTO users (tenant_id, email, password_hash, role, full_name) VALUES (?, ?, ?, ?, ?) RETURNING id`,
			tenantID,
			strings.TrimSpace(strings.ToLower(in.AdminEmail)),
			in.PasswordHash,
			in.Role,
			strings.TrimSpace(in.AdminFullName),
		).Scan(&userID).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate key") {
			return schemas.CreateTenantAdminResult{}, errorcodes.ErrConflict.WithDetails(map[string]any{
				"message": "email already exists",
			})
		}
		return schemas.CreateTenantAdminResult{}, err
	}

	return schemas.CreateTenantAdminResult{
		TenantID: tenantID,
		UserID:   userID,
		Email:    strings.TrimSpace(strings.ToLower(in.AdminEmail)),
	}, nil
}

func (r *repository) FindUserCredentialByEmail(ctx context.Context, email string) (schemas.AuthUserCredential, error) {
	var out schemas.AuthUserCredential
	err := r.DB().WithContext(ctx).Raw(
		`SELECT id, tenant_id, email, password_hash, role FROM users WHERE lower(email) = lower(?) LIMIT 1`,
		strings.TrimSpace(email),
	).Scan(&out).Error
	if err != nil {
		return schemas.AuthUserCredential{}, err
	}
	if out.ID == "" || out.TenantID == "" {
		return schemas.AuthUserCredential{}, gorm.ErrRecordNotFound
	}
	return out, nil
}
