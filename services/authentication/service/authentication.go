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
		var ae errorcodes.AppError
		if errors.As(err, &ae) {
			return RegisterResult{}, err
		}
		return RegisterResult{}, errorcodes.ErrInternal
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

	ci := commonjwt.ClaimsInput{
		Subject:     user.ID,
		TenantID:    user.TenantID,
		Role:        user.Role,
		Permissions: commonjwt.PermissionsForRole(user.Role),
	}
	access, err := s.jwt.GenerateAccessToken(ci)
	if err != nil {
		return LoginResult{}, err
	}
	refresh, err := s.jwt.GenerateRefreshToken(ci)
	if err != nil {
		return LoginResult{}, err
	}

	rClaims, err := s.jwt.ParseRefresh(refresh)
	if err != nil || rClaims.ID == "" || rClaims.ExpiresAt == nil {
		return LoginResult{}, fmt.Errorf("%w: parse issued refresh", errorcodes.ErrInternal)
	}
	if err := s.repo.InsertRefreshSession(ctx, user.ID, user.TenantID, rClaims.ID, rClaims.ExpiresAt.Time); err != nil {
		return LoginResult{}, fmt.Errorf("login: persist refresh session: %w", err)
	}

	return LoginResult{
		AccessToken:  access,
		RefreshToken: refresh,
		TokenType:    "Bearer",
		ExpiresIn:    s.accessTTLSeconds,
	}, nil
}

var nonSlugChars = regexp.MustCompile(`[^a-z0-9]+`)

// generateTenantSlug builds a unique URL-safe slug (ARCHITECTURE §7 tenants.slug).
// Refresh validates a refresh JWT and session row, rotates refresh, and issues new tokens.
func (s *AuthenticationService) Refresh(ctx context.Context, in RefreshInput) (LoginResult, error) {
	rt := strings.TrimSpace(in.RefreshToken)
	if rt == "" {
		return LoginResult{}, errorcodes.ErrValidationError.WithDetails(map[string]any{
			"message": "refresh_token is required",
		})
	}

	claims, err := s.jwt.ParseRefresh(rt)
	if err != nil {
		return LoginResult{}, errorcodes.ErrUnauthorized
	}

	uid, tid, active, err := s.repo.FindActiveRefreshSession(ctx, claims.ID)
	if err != nil {
		return LoginResult{}, fmt.Errorf("refresh: session lookup: %w", err)
	}
	if !active || uid != claims.Subject || tid != claims.TenantID {
		return LoginResult{}, errorcodes.ErrUnauthorized
	}

	user, err := s.repo.FindUserCredentialByID(ctx, uid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return LoginResult{}, errorcodes.ErrUnauthorized
		}
		return LoginResult{}, err
	}

	if err := s.repo.RevokeRefreshSession(ctx, claims.ID); err != nil {
		return LoginResult{}, err
	}

	ci := commonjwt.ClaimsInput{
		Subject:     user.ID,
		TenantID:    user.TenantID,
		Role:        user.Role,
		Permissions: commonjwt.PermissionsForRole(user.Role),
	}
	access, err := s.jwt.GenerateAccessToken(ci)
	if err != nil {
		return LoginResult{}, err
	}
	newRefresh, err := s.jwt.GenerateRefreshToken(ci)
	if err != nil {
		return LoginResult{}, err
	}
	nrClaims, err := s.jwt.ParseRefresh(newRefresh)
	if err != nil || nrClaims.ID == "" || nrClaims.ExpiresAt == nil {
		return LoginResult{}, fmt.Errorf("%w: parse issued refresh", errorcodes.ErrInternal)
	}
	if err := s.repo.InsertRefreshSession(ctx, user.ID, user.TenantID, nrClaims.ID, nrClaims.ExpiresAt.Time); err != nil {
		return LoginResult{}, fmt.Errorf("refresh: persist session: %w", err)
	}

	return LoginResult{
		AccessToken:  access,
		RefreshToken: newRefresh,
		TokenType:    "Bearer",
		ExpiresIn:    s.accessTTLSeconds,
	}, nil
}

// Logout revokes all refresh sessions for the access-token subject.
func (s *AuthenticationService) Logout(ctx context.Context) error {
	claims, ok := commonjwt.ClaimsFromContext(ctx)
	if !ok || claims == nil || claims.Subject == "" {
		return errorcodes.ErrUnauthorized
	}
	return s.repo.RevokeAllRefreshSessionsForUser(ctx, claims.Subject)
}

// Me returns the profile for the authenticated access token.
func (s *AuthenticationService) Me(ctx context.Context) (MeResult, error) {
	claims, ok := commonjwt.ClaimsFromContext(ctx)
	if !ok || claims == nil || claims.Subject == "" {
		return MeResult{}, errorcodes.ErrUnauthorized
	}

	prof, err := s.repo.FindUserProfileByID(ctx, claims.Subject)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return MeResult{}, errorcodes.ErrNotFound
		}
		return MeResult{}, err
	}
	if prof.TenantID != claims.TenantID {
		return MeResult{}, errorcodes.ErrForbidden
	}

	return MeResult{
		UserID:   prof.UserID,
		TenantID: prof.TenantID,
		Email:    prof.Email,
		FullName: prof.FullName,
	}, nil
}

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
