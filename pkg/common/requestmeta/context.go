package requestmeta

import (
	"context"

	"github.com/labstack/echo/v4"
)

type ctxKey struct{}

// Meta holds HTTP metadata for audit logging (§14).
type Meta struct {
	IP        *string
	UserAgent *string
	RequestID *string
}

// WithContext attaches Meta to ctx (typically request context).
func WithContext(ctx context.Context, m Meta) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, ctxKey{}, m)
}

// FromContext returns Meta stored by EchoMiddleware; empty Meta if none.
func FromContext(ctx context.Context) Meta {
	if ctx == nil {
		return Meta{}
	}
	v, ok := ctx.Value(ctxKey{}).(Meta)
	if !ok {
		return Meta{}
	}
	return v
}

// EchoMiddleware stores client IP, User-Agent, and X-Request-ID on the request context.
// Place after JWT middleware so the chain remains: ... -> JWT -> requestmeta -> handler.
func EchoMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			reqID := c.Response().Header().Get(echo.HeaderXRequestID)
			if reqID == "" {
				reqID = req.Header.Get(echo.HeaderXRequestID)
			}
			ip := c.RealIP()
			ua := req.UserAgent()
			var m Meta
			if ip != "" {
				s := ip
				m.IP = &s
			}
			if ua != "" {
				s := ua
				m.UserAgent = &s
			}
			if reqID != "" {
				s := reqID
				m.RequestID = &s
			}
			ctx := WithContext(req.Context(), m)
			c.SetRequest(req.WithContext(ctx))
			return next(c)
		}
	}
}
