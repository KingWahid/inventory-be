package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

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
	FindUserCredentialByID(ctx context.Context, userID string) (schemas.AuthUserCredential, error)
	FindUserProfileByID(ctx context.Context, userID string) (schemas.UserProfile, error)
	InsertRefreshSession(ctx context.Context, userID, tenantID, jti string, expiresAt time.Time) error
	FindActiveRefreshSession(ctx context.Context, jti string) (userID, tenantID string, ok bool, err error)
	RevokeRefreshSession(ctx context.Context, jti string) error
	RevokeAllRefreshSessionsForUser(ctx context.Context, userID string) error
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

func (r *repository) FindUserCredentialByID(ctx context.Context, userID string) (schemas.AuthUserCredential, error) {
	var out schemas.AuthUserCredential
	err := r.DB().WithContext(ctx).Raw(
		`SELECT id, tenant_id, email, password_hash, role FROM users WHERE id = ? LIMIT 1`,
		strings.TrimSpace(userID),
	).Scan(&out).Error
	if err != nil {
		return schemas.AuthUserCredential{}, err
	}
	if out.ID == "" || out.TenantID == "" {
		return schemas.AuthUserCredential{}, gorm.ErrRecordNotFound
	}
	return out, nil
}

func (r *repository) FindUserProfileByID(ctx context.Context, userID string) (schemas.UserProfile, error) {
	var out schemas.UserProfile
	err := r.DB().WithContext(ctx).Raw(
		`SELECT id::text AS user_id, tenant_id::text AS tenant_id, email, COALESCE(full_name, '') AS full_name FROM users WHERE id = ? LIMIT 1`,
		strings.TrimSpace(userID),
	).Scan(&out).Error
	if err != nil {
		return schemas.UserProfile{}, err
	}
	if out.UserID == "" || out.TenantID == "" {
		return schemas.UserProfile{}, gorm.ErrRecordNotFound
	}
	return out, nil
}

func (r *repository) InsertRefreshSession(ctx context.Context, userID, tenantID, jti string, expiresAt time.Time) error {
	return r.DB().WithContext(ctx).Exec(
		`INSERT INTO refresh_sessions (user_id, tenant_id, jti, expires_at) VALUES (?, ?, ?, ?)`,
		userID, tenantID, jti, expiresAt,
	).Error
}

func (r *repository) FindActiveRefreshSession(ctx context.Context, jti string) (userID, tenantID string, ok bool, err error) {
	var uid, tid string
	tx := r.DB().WithContext(ctx).Raw(
		`SELECT user_id::text, tenant_id::text FROM refresh_sessions
		 WHERE jti = ? AND revoked_at IS NULL AND expires_at > NOW()`,
		strings.TrimSpace(jti),
	).Row()
	err = tx.Scan(&uid, &tid)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", false, nil
	}
	if err != nil {
		return "", "", false, err
	}
	if uid == "" || tid == "" {
		return "", "", false, nil
	}
	return uid, tid, true, nil
}

func (r *repository) RevokeRefreshSession(ctx context.Context, jti string) error {
	return r.DB().WithContext(ctx).Exec(
		`UPDATE refresh_sessions SET revoked_at = NOW() WHERE jti = ? AND revoked_at IS NULL`,
		strings.TrimSpace(jti),
	).Error
}

func (r *repository) RevokeAllRefreshSessionsForUser(ctx context.Context, userID string) error {
	return r.DB().WithContext(ctx).Exec(
		`UPDATE refresh_sessions SET revoked_at = NOW() WHERE user_id = ? AND revoked_at IS NULL`,
		strings.TrimSpace(userID),
	).Error
}
