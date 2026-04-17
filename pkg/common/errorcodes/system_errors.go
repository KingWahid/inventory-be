package errorcodes

import "errors"

var (
	ErrJWTInvalidSecret    = errors.New("jwt: secret is required")
	ErrJWTInvalidTTL       = errors.New("jwt: ttl must be greater than zero")
	ErrJWTInvalidSubject   = errors.New("jwt: subject is required")
	ErrJWTInvalidTenantID  = errors.New("jwt: tenant_id is required")
	ErrJWTInvalidTokenType = errors.New("jwt: invalid token type")
	ErrJWTParseToken       = errors.New("jwt: failed to parse token")
	ErrJWTInvalidClaims    = errors.New("jwt: invalid claims")
	ErrJWTInvalidSigning   = errors.New("jwt: invalid signing method")

	ErrTxBegin    = errors.New("transaction: begin tx")
	ErrTxCommit   = errors.New("transaction: commit tx")
	ErrTxRollback = errors.New("transaction: rollback tx")
)

const (
	ValidationRuleBind     = "bind"
	ValidationRuleValidate = "validate"
)
