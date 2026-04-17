package jwt

import (
	"errors"
	"testing"
	"time"

	"github.com/your-org/inventory/backend/pkg/common/errorcodes"
)

func TestGenerateAndParseAccessToken(t *testing.T) {
	svc, err := NewService("secret", 15*time.Minute, 24*time.Hour)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	token, err := svc.GenerateAccessToken("user-1", "tenant-1")
	if err != nil {
		t.Fatalf("generate access token: %v", err)
	}

	claims, err := svc.ParseAccess(token)
	if err != nil {
		t.Fatalf("parse access token: %v", err)
	}

	if claims.Subject != "user-1" {
		t.Fatalf("expected subject user-1, got %s", claims.Subject)
	}
	if claims.TenantID != "tenant-1" {
		t.Fatalf("expected tenant tenant-1, got %s", claims.TenantID)
	}
	if claims.TokenType != TokenTypeAccess {
		t.Fatalf("expected token type access, got %s", claims.TokenType)
	}
}

func TestGenerateAndParseRefreshToken(t *testing.T) {
	svc, err := NewService("secret", 15*time.Minute, 24*time.Hour)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	token, err := svc.GenerateRefreshToken("user-2", "tenant-2")
	if err != nil {
		t.Fatalf("generate refresh token: %v", err)
	}

	claims, err := svc.ParseRefresh(token)
	if err != nil {
		t.Fatalf("parse refresh token: %v", err)
	}

	if claims.Subject != "user-2" {
		t.Fatalf("expected subject user-2, got %s", claims.Subject)
	}
	if claims.TenantID != "tenant-2" {
		t.Fatalf("expected tenant tenant-2, got %s", claims.TenantID)
	}
	if claims.TokenType != TokenTypeRefresh {
		t.Fatalf("expected token type refresh, got %s", claims.TokenType)
	}
}

func TestTokenTypeMismatch(t *testing.T) {
	svc, err := NewService("secret", 15*time.Minute, 24*time.Hour)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	accessToken, err := svc.GenerateAccessToken("user-1", "tenant-1")
	if err != nil {
		t.Fatalf("generate access token: %v", err)
	}
	if _, err := svc.ParseRefresh(accessToken); !errors.Is(err, errorcodes.ErrJWTInvalidTokenType) {
		t.Fatalf("expected ErrInvalidTokenType, got %v", err)
	}

	refreshToken, err := svc.GenerateRefreshToken("user-1", "tenant-1")
	if err != nil {
		t.Fatalf("generate refresh token: %v", err)
	}
	if _, err := svc.ParseAccess(refreshToken); !errors.Is(err, errorcodes.ErrJWTInvalidTokenType) {
		t.Fatalf("expected ErrInvalidTokenType, got %v", err)
	}
}

func TestExpiredTokenRejected(t *testing.T) {
	svc, err := NewService("secret", 15*time.Minute, 24*time.Hour)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	svc.now = func() time.Time {
		return time.Now().Add(-2 * time.Hour)
	}

	expiredToken, err := svc.GenerateAccessToken("user-1", "tenant-1")
	if err != nil {
		t.Fatalf("generate expired token: %v", err)
	}

	svc.now = time.Now
	if _, err := svc.ParseAccess(expiredToken); err == nil {
		t.Fatal("expected expired token parsing error")
	}
}

func TestWrongSecretRejected(t *testing.T) {
	issuer, err := NewService("secret-a", 15*time.Minute, 24*time.Hour)
	if err != nil {
		t.Fatalf("new issuer service: %v", err)
	}
	verifier, err := NewService("secret-b", 15*time.Minute, 24*time.Hour)
	if err != nil {
		t.Fatalf("new verifier service: %v", err)
	}

	token, err := issuer.GenerateAccessToken("user-1", "tenant-1")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	if _, err := verifier.ParseAccess(token); err == nil {
		t.Fatal("expected parsing error for wrong secret")
	}
}
