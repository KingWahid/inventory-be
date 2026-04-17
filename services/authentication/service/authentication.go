package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/KingWahid/inventory/backend/pkg/common/errorcodes"
	"github.com/KingWahid/inventory/backend/pkg/database/schemas"
	"github.com/KingWahid/inventory/backend/services/authentication/repository"
	"golang.org/x/crypto/bcrypt"
)

// AuthenticationService implements Service using PostgreSQL.
type AuthenticationService struct {
	repo repository.Repository
}

// NewAuthenticationService constructs the default authentication service.
func NewAuthenticationService(repo repository.Repository) *AuthenticationService {
	return &AuthenticationService{repo: repo}
}

// PingDB checks database connectivity.
func (s *AuthenticationService) PingDB(ctx context.Context) error {
	if err := s.repo.PingDB(ctx); err != nil {
		return fmt.Errorf("authentication service: ping db: %w", err)
	}
	return nil
}

// RegisterTenantAdmin creates a tenant and first admin user in a single transaction.
func (s *AuthenticationService) RegisterTenantAdmin(ctx context.Context, in RegisterInput) (RegisterResult, error) {
	if strings.TrimSpace(in.TenantName) == "" ||
		strings.TrimSpace(in.AdminName) == "" ||
		strings.TrimSpace(in.AdminEmail) == "" ||
		len(in.Password) < 8 {
		return RegisterResult{}, errorcodes.ErrValidationError.WithDetails(map[string]any{
			"message": "tenant_name, admin_name, admin_email are required and password min 8 chars",
		})
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return RegisterResult{}, fmt.Errorf("%w: hash password", errorcodes.ErrInternal)
	}

	email := strings.TrimSpace(strings.ToLower(in.AdminEmail))

	repoResult, err := s.repo.CreateTenantAdmin(ctx, schemas.CreateTenantAdminInput{
		TenantName:   strings.TrimSpace(in.TenantName),
		AdminEmail:   email,
		PasswordHash: string(passwordHash),
		Role:         "owner",
	})
	if err != nil {
		return RegisterResult{}, fmt.Errorf("register tenant admin: %w", err)
	}

	return RegisterResult{
		TenantID: repoResult.TenantID,
		UserID:   repoResult.UserID,
		Email:    repoResult.Email,
	}, nil
}
