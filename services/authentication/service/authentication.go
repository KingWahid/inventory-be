package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/KingWahid/inventory/backend/pkg/common/errorcodes"
	commonjwt "github.com/KingWahid/inventory/backend/pkg/common/jwt"
	"github.com/KingWahid/inventory/backend/pkg/database/schemas"
	"github.com/KingWahid/inventory/backend/services/authentication/repository"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AuthenticationService implements Service using PostgreSQL.
type AuthenticationService struct {
	repo             repository.Repository
	jwt              *commonjwt.Service
	accessTTLSeconds int64
}

// NewAuthenticationService constructs the default authentication service.
func NewAuthenticationService(repo repository.Repository, jwt *commonjwt.Service, accessTTLSeconds int64) *AuthenticationService {
	return &AuthenticationService{
		repo:             repo,
		jwt:              jwt,
		accessTTLSeconds: accessTTLSeconds,
	}
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
		TenantName:    strings.TrimSpace(in.TenantName),
		TenantSlug:    generateTenantSlug(in.TenantName),
		AdminEmail:    email,
		AdminFullName: strings.TrimSpace(in.AdminName),
		PasswordHash:  string(passwordHash),
		Role:          "owner",
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

// Login validates credential and issues an access token.
func (s *AuthenticationService) Login(ctx context.Context, in LoginInput) (LoginResult, error) {
	email := strings.TrimSpace(strings.ToLower(in.Email))
	if email == "" || strings.TrimSpace(in.Password) == "" {
		return LoginResult{}, errorcodes.ErrValidationError.WithDetails(map[string]any{
			"message": "email and password are required",
		})
	}

	user, err := s.repo.FindUserCredentialByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return LoginResult{}, errorcodes.ErrUnauthorized
		}
		return LoginResult{}, fmt.Errorf("login: find credential: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(in.Password)); err != nil {
		return LoginResult{}, errorcodes.ErrUnauthorized
	}

	token, err := s.jwt.GenerateAccessToken(user.ID, user.TenantID)
	if err != nil {
		return LoginResult{}, fmt.Errorf("login: generate access token: %w", err)
	}

	return LoginResult{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   s.accessTTLSeconds,
	}, nil
}

var nonSlugChars = regexp.MustCompile(`[^a-z0-9]+`)

// generateTenantSlug builds a unique URL-safe slug (ARCHITECTURE §7 tenants.slug).
func generateTenantSlug(name string) string {
	base := strings.TrimSpace(strings.ToLower(name))
	base = strings.Trim(nonSlugChars.ReplaceAllString(base, "-"), "-")
	if base == "" {
		base = "tenant"
	}
	if len(base) > 80 {
		base = base[:80]
	}
	return base + "-" + uuid.New().String()[:8]
}
