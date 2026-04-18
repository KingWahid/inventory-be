package jwt

import "github.com/golang-jwt/jwt/v4"

const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

// Claims matches ARCHITECTURE §7 (sub via RegisteredClaims.Subject).
type Claims struct {
	TenantID    string   `json:"tenant_id"`
	TokenType   string   `json:"token_type"`
	Role        string   `json:"role,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
	jwt.RegisteredClaims
}
