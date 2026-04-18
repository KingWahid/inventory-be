package jwt

import (
	"context"

	"github.com/KingWahid/inventory/backend/pkg/common/errorcodes"
)

type claimsContextKey struct{}

// ContextWithClaims attaches parsed JWT claims to ctx (typically request context).
func ContextWithClaims(ctx context.Context, claims *Claims) context.Context {
	if claims == nil {
		return ctx
	}
	return context.WithValue(ctx, claimsContextKey{}, claims)
}

// ClaimsFromContext returns claims previously stored with ContextWithClaims.
func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	if ctx == nil {
		return nil, false
	}
	v := ctx.Value(claimsContextKey{})
	c, ok := v.(*Claims)
	return c, ok && c != nil
}

// TenantIDFromContext returns tenant_id from JWT claims on ctx (see RequireBearerAccessJWT).
func TenantIDFromContext(ctx context.Context) (string, error) {
	claims, ok := ClaimsFromContext(ctx)
	if !ok || claims == nil || claims.TenantID == "" {
		return "", errorcodes.ErrTenantContextMissing
	}
	return claims.TenantID, nil
}
