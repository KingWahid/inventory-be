package jwt

import "github.com/golang-jwt/jwt/v4"

const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

type Claims struct {
	TenantID  string `json:"tenant_id"`
	TokenType string `json:"token_type"`
	jwt.RegisteredClaims
}
