package api

import (
	"github.com/KingWahid/inventory/backend/pkg/common/errorcodes"
	commonjwt "github.com/KingWahid/inventory/backend/pkg/common/jwt"
	"github.com/labstack/echo/v4"
)

// AuthPublicPaths are routes that skip JWT validation (must match URL.Path exactly).
var AuthPublicPaths = map[string]struct{}{
	"/health":               {},
	"/ready":                {},
	"/api/v1/auth/health":   {},
	"/api/v1/auth/login":    {},
	"/api/v1/auth/register": {},
}

// RequireAccessJWT validates Bearer access tokens and attaches claims to request.Context().
func RequireAccessJWT(jwtSvc *commonjwt.Service) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			path := c.Request().URL.Path
			if _, skip := AuthPublicPaths[path]; skip {
				return next(c)
			}

			token, err := commonjwt.GetJWTFromEchoContext(c)
			if err != nil {
				return err
			}

			claims, err := jwtSvc.ParseAccess(token)
			if err != nil {
				return errorcodes.ErrUnauthorized
			}

			ctx := commonjwt.ContextWithClaims(c.Request().Context(), claims)
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	}
}
