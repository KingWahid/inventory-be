package jwt

import (
	"errors"
	"fmt"
	"time"

	jwtv4 "github.com/golang-jwt/jwt/v4"

	"github.com/your-org/inventory/backend/pkg/common/errorcodes"
)

type Service struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
	now        func() time.Time
}

func NewService(secret string, accessTTL, refreshTTL time.Duration) (*Service, error) {
	if secret == "" {
		return nil, errorcodes.ErrJWTInvalidSecret
	}
	if accessTTL <= 0 || refreshTTL <= 0 {
		return nil, errorcodes.ErrJWTInvalidTTL
	}

	return &Service{
		secret:     []byte(secret),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
		now:        time.Now,
	}, nil
}

func (s *Service) GenerateAccessToken(subject, tenantID string) (string, error) {
	return s.generateToken(subject, tenantID, TokenTypeAccess, s.accessTTL)
}

func (s *Service) GenerateRefreshToken(subject, tenantID string) (string, error) {
	return s.generateToken(subject, tenantID, TokenTypeRefresh, s.refreshTTL)
}

func (s *Service) generateToken(subject, tenantID, tokenType string, ttl time.Duration) (string, error) {
	if subject == "" {
		return "", errorcodes.ErrJWTInvalidSubject
	}
	if tenantID == "" {
		return "", errorcodes.ErrJWTInvalidTenantID
	}
	if tokenType != TokenTypeAccess && tokenType != TokenTypeRefresh {
		return "", errorcodes.ErrJWTInvalidTokenType
	}

	now := s.now()
	claims := Claims{
		TenantID:  tenantID,
		TokenType: tokenType,
		RegisteredClaims: jwtv4.RegisteredClaims{
			Subject:   subject,
			ExpiresAt: jwtv4.NewNumericDate(now.Add(ttl)),
			IssuedAt:  jwtv4.NewNumericDate(now),
			NotBefore: jwtv4.NewNumericDate(now),
		},
	}

	token := jwtv4.NewWithClaims(jwtv4.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.secret)
	if err != nil {
		return "", fmt.Errorf("jwt: sign token: %w", err)
	}
	return signed, nil
}

func (s *Service) Parse(token string) (*Claims, error) {
	claims, err := s.parseToken(token)
	if err != nil {
		return nil, err
	}
	return claims, nil
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
	parsed, err := jwtv4.ParseWithClaims(token, &Claims{}, func(t *jwtv4.Token) (any, error) {
		if _, ok := t.Method.(*jwtv4.SigningMethodHMAC); !ok {
			return nil, errorcodes.ErrJWTInvalidSigning
		}
		return s.secret, nil
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
		return nil, errorcodes.ErrJWTInvalidTokenType
	}

	return claims, nil
}

func IsParseError(err error) bool {
	return errors.Is(err, errorcodes.ErrJWTParseToken) ||
		errors.Is(err, errorcodes.ErrJWTInvalidClaims) ||
		errors.Is(err, errorcodes.ErrJWTInvalidSigning)
}
