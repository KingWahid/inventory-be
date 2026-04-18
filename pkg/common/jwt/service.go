package jwt

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	jwtv4 "github.com/golang-jwt/jwt/v4"

	"github.com/KingWahid/inventory/backend/pkg/common/errorcodes"
)

// ClaimsInput carries identity and optional RBAC fields embedded in JWT (ARCHITECTURE §7).
type ClaimsInput struct {
	Subject     string
	TenantID    string
	Role        string
	Permissions []string
}

// Service issues and verifies HS256 JWTs with optional separate access vs refresh secrets.
type Service struct {
	accessSecret  []byte
	refreshSecret []byte
	accessTTL     time.Duration
	refreshTTL    time.Duration
	issuer        string
	audience      string
	now           func() time.Time
}

// ServiceOptions configures JWT issuance and verification.
type ServiceOptions struct {
	// SharedSecret sets both access and refresh signing keys (backward compatible).
	SharedSecret string

	// AccessSecret / RefreshSecret used when SharedSecret is empty (recommended for prod).
	AccessSecret  string
	RefreshSecret string

	AccessTTL  time.Duration
	RefreshTTL time.Duration

	Issuer   string
	Audience string
}

// NewService signs access and refresh with one HS256 secret (backward compatible).
func NewService(secret string, accessTTL, refreshTTL time.Duration) (*Service, error) {
	return NewServiceOptions(ServiceOptions{
		SharedSecret: secret,
		AccessTTL:    accessTTL,
		RefreshTTL:   refreshTTL,
	})
}

// NewServiceOptions builds Service from explicit options.
func NewServiceOptions(opt ServiceOptions) (*Service, error) {
	accessSec := opt.AccessSecret
	refreshSec := opt.RefreshSecret
	if opt.SharedSecret != "" {
		accessSec = opt.SharedSecret
		refreshSec = opt.SharedSecret
	}
	if accessSec == "" || refreshSec == "" {
		return nil, errorcodes.ErrJWTInvalidSecret
	}
	if opt.AccessTTL <= 0 || opt.RefreshTTL <= 0 {
		return nil, errorcodes.ErrJWTInvalidTTL
	}

	return &Service{
		accessSecret:  []byte(accessSec),
		refreshSecret: []byte(refreshSec),
		accessTTL:     opt.AccessTTL,
		refreshTTL:    opt.RefreshTTL,
		issuer:        opt.Issuer,
		audience:      opt.Audience,
		now:           time.Now,
	}, nil
}

func (s *Service) GenerateAccessToken(in ClaimsInput) (string, error) {
	return s.generateToken(in, TokenTypeAccess, s.accessTTL, s.accessSecret)
}

func (s *Service) GenerateRefreshToken(in ClaimsInput) (string, error) {
	return s.generateToken(in, TokenTypeRefresh, s.refreshTTL, s.refreshSecret)
}

func (s *Service) generateToken(in ClaimsInput, tokenType string, ttl time.Duration, signingKey []byte) (string, error) {
	if in.Subject == "" {
		return "", errorcodes.ErrJWTInvalidSubject
	}
	if in.TenantID == "" {
		return "", errorcodes.ErrJWTInvalidTenantID
	}
	if tokenType != TokenTypeAccess && tokenType != TokenTypeRefresh {
		return "", errorcodes.ErrJWTInvalidTokenType
	}

	now := s.now()
	rc := jwtv4.RegisteredClaims{
		Subject:   in.Subject,
		ExpiresAt: jwtv4.NewNumericDate(now.Add(ttl)),
		IssuedAt:  jwtv4.NewNumericDate(now),
		NotBefore: jwtv4.NewNumericDate(now),
	}
	if s.issuer != "" {
		rc.Issuer = s.issuer
	}
	if s.audience != "" {
		rc.Audience = jwtv4.ClaimStrings{s.audience}
	}

	perms := append([]string(nil), in.Permissions...)
	claims := Claims{
		TenantID:         in.TenantID,
		TokenType:        tokenType,
		Role:             in.Role,
		Permissions:      perms,
		RegisteredClaims: rc,
	}

	token := jwtv4.NewWithClaims(jwtv4.SigningMethodHS256, claims)
	signed, err := token.SignedString(signingKey)
	if err != nil {
		return "", fmt.Errorf("jwt: sign token: %w", err)
	}
	return signed, nil
}

func (s *Service) Parse(token string) (*Claims, error) {
	return s.parseToken(token)
}

func (s *Service) ParseAccess(token string) (*Claims, error) {
	claims, err := s.parseToken(token)
	if err != nil {
		return nil, err
	}
	if claims.TokenType != TokenTypeAccess {
		return nil, errorcodes.ErrJWTInvalidTokenType
	}
	return claims, nil
}

func (s *Service) ParseRefresh(token string) (*Claims, error) {
	claims, err := s.parseToken(token)
	if err != nil {
		return nil, err
	}
	if claims.TokenType != TokenTypeRefresh {
		return nil, errorcodes.ErrJWTInvalidTokenType
	}
	return claims, nil
}

func (s *Service) parseToken(token string) (*Claims, error) {
	secrets := uniqueSecretKeys(s.accessSecret, s.refreshSecret)
	var lastErr error
	for _, sec := range secrets {
		c, err := s.parseTokenWithKey(token, sec)
		if err == nil {
			return c, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

func uniqueSecretKeys(a, b []byte) [][]byte {
	out := [][]byte{a}
	if !bytes.Equal(a, b) {
		out = append(out, b)
	}
	return out
}

func (s *Service) parseTokenWithKey(token string, secret []byte) (*Claims, error) {
	parsed, err := jwtv4.ParseWithClaims(token, &Claims{}, func(t *jwtv4.Token) (any, error) {
		if _, ok := t.Method.(*jwtv4.SigningMethodHMAC); !ok {
			return nil, errorcodes.ErrJWTInvalidSigning
		}
		return secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errorcodes.ErrJWTParseToken, err)
	}
	if !parsed.Valid {
		return nil, errorcodes.ErrJWTParseToken
	}

	claims, ok := parsed.Claims.(*Claims)
	if !ok {
		return nil, errorcodes.ErrJWTInvalidClaims
	}
	if claims.Subject == "" || claims.TenantID == "" || claims.ExpiresAt == nil {
		return nil, errorcodes.ErrJWTInvalidClaims
	}

	if err := claims.Valid(); err != nil {
		return nil, fmt.Errorf("%w: %v", errorcodes.ErrJWTParseToken, err)
	}

	if claims.TokenType != TokenTypeAccess && claims.TokenType != TokenTypeRefresh {
		return nil, errorcodes.ErrJWTInvalidClaims
	}

	if err := s.validateIssuerAudience(claims); err != nil {
		return nil, err
	}

	return claims, nil
}

func (s *Service) validateIssuerAudience(c *Claims) error {
	if s.issuer != "" && c.Issuer != s.issuer {
		return errorcodes.ErrJWTInvalidClaims
	}
	if s.audience != "" {
		ok := false
		for _, a := range c.Audience {
			if a == s.audience {
				ok = true
				break
			}
		}
		if !ok {
			return errorcodes.ErrJWTInvalidClaims
		}
	}
	return nil
}

func IsParseError(err error) bool {
	return errors.Is(err, errorcodes.ErrJWTParseToken) ||
		errors.Is(err, errorcodes.ErrJWTInvalidClaims) ||
		errors.Is(err, errorcodes.ErrJWTInvalidSigning)
}
