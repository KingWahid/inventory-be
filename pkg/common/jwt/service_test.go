package jwt

import (
	"errors"
	"testing"
	"time"

	"github.com/KingWahid/inventory/backend/pkg/common/errorcodes"
)

func TestGenerateAndParseAccessToken(t *testing.T) {
	svc, err := NewService("secret", 15*time.Minute, 24*time.Hour)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	token, err := svc.GenerateAccessToken(ClaimsInput{Subject: "user-1", TenantID: "tenant-1"})
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

func TestRoleAndPermissionsRoundTrip(t *testing.T) {
	svc, err := NewService("secret", 15*time.Minute, 24*time.Hour)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	in := ClaimsInput{
		Subject:     "u1",
		TenantID:    "t1",
		Role:        "admin",
		Permissions: []string{"product:write", "report:read"},
	}
	tok, err := svc.GenerateAccessToken(in)
	if err != nil {
		t.Fatal(err)
	}
	c, err := svc.ParseAccess(tok)
	if err != nil {
		t.Fatal(err)
	}
	if c.Role != "admin" {
		t.Fatalf("role want admin got %q", c.Role)
	}
	if len(c.Permissions) != 2 || c.Permissions[0] != "product:write" {
		t.Fatalf("permissions %+v", c.Permissions)
	}
}

func TestGenerateAndParseRefreshToken(t *testing.T) {
	svc, err := NewService("secret", 15*time.Minute, 24*time.Hour)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	token, err := svc.GenerateRefreshToken(ClaimsInput{Subject: "user-2", TenantID: "tenant-2"})
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

	accessToken, err := svc.GenerateAccessToken(ClaimsInput{Subject: "user-1", TenantID: "tenant-1"})
	if err != nil {
		t.Fatalf("generate access token: %v", err)
	}
	if _, err := svc.ParseRefresh(accessToken); !errors.Is(err, errorcodes.ErrJWTInvalidTokenType) {
		t.Fatalf("expected ErrInvalidTokenType, got %v", err)
	}

	refreshToken, err := svc.GenerateRefreshToken(ClaimsInput{Subject: "user-1", TenantID: "tenant-1"})
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

	expiredToken, err := svc.GenerateAccessToken(ClaimsInput{Subject: "user-1", TenantID: "tenant-1"})
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

	token, err := issuer.GenerateAccessToken(ClaimsInput{Subject: "user-1", TenantID: "tenant-1"})
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	if _, err := verifier.ParseAccess(token); err == nil {
		t.Fatal("expected parsing error for wrong secret")
	}
}

func TestDualSecretsAccessVsRefreshKeys(t *testing.T) {
	svc, err := NewServiceOptions(ServiceOptions{
		AccessSecret:  "access-key-16bytes!!",
		RefreshSecret: "refresh-key-16bytes!",
		AccessTTL:     15 * time.Minute,
		RefreshTTL:    24 * time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	accessTok, err := svc.GenerateAccessToken(ClaimsInput{Subject: "u", TenantID: "t"})
	if err != nil {
		t.Fatal(err)
	}
	refreshTok, err := svc.GenerateRefreshToken(ClaimsInput{Subject: "u", TenantID: "t"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ParseAccess(accessTok); err != nil {
		t.Fatalf("parse access: %v", err)
	}
	if _, err := svc.ParseRefresh(refreshTok); err != nil {
		t.Fatalf("parse refresh: %v", err)
	}
	// refresh JWT must not satisfy ParseAccess (wrong type)
	if _, err := svc.ParseAccess(refreshTok); !errors.Is(err, errorcodes.ErrJWTInvalidTokenType) {
		t.Fatalf("want invalid type got %v", err)
	}
}

func TestIssuerAudienceEnforced(t *testing.T) {
	svc, err := NewServiceOptions(ServiceOptions{
		SharedSecret: "same-secret-for-test",
		AccessTTL:    time.Hour,
		RefreshTTL:   time.Hour,
		Issuer:       "inventory-api",
		Audience:     "inventory-clients",
	})
	if err != nil {
		t.Fatal(err)
	}
	tok, err := svc.GenerateAccessToken(ClaimsInput{Subject: "u", TenantID: "t"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ParseAccess(tok); err != nil {
		t.Fatal(err)
	}

	bad, err := NewServiceOptions(ServiceOptions{
		SharedSecret: "same-secret-for-test",
		AccessTTL:    time.Hour,
		RefreshTTL:   time.Hour,
		Issuer:       "other",
		Audience:     "inventory-clients",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bad.ParseAccess(tok); err == nil {
		t.Fatal("expected issuer mismatch")
	}
}
