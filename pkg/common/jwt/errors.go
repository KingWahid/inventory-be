package jwt

import "errors"

var (
	ErrInvalidSecret    = errors.New("jwt: secret is required")
	ErrInvalidTTL       = errors.New("jwt: ttl must be greater than zero")
	ErrInvalidSubject   = errors.New("jwt: subject is required")
	ErrInvalidTenantID  = errors.New("jwt: tenant_id is required")
	ErrInvalidTokenType = errors.New("jwt: invalid token type")
	ErrParseToken       = errors.New("jwt: failed to parse token")
	ErrInvalidClaims    = errors.New("jwt: invalid claims")
	ErrInvalidSigning   = errors.New("jwt: invalid signing method")
)
