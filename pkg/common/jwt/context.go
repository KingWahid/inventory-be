package jwt

import "context"

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
