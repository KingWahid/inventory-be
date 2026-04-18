package jwt

import (
	"github.com/KingWahid/inventory/backend/pkg/common/errorcodes"
	"github.com/labstack/echo/v4"
)

// RequireBearerAccessJWT validates Bearer access JWTs (token_type=access) and attaches claims to request context.
// Paths present in publicPaths are skipped (exact URL.Path match).
func RequireBearerAccessJWT(jwtSvc *Service, publicPaths map[string]struct{}) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			path := c.Request().URL.Path
			if _, skip := publicPaths[path]; skip {
				return next(c)
			}

			token, err := GetJWTFromEchoContext(c)
			if err != nil {
				return err
			}

			claims, err := jwtSvc.ParseAccess(token)
			if err != nil {
				return errorcodes.ErrUnauthorized
			}

			ctx := ContextWithClaims(c.Request().Context(), claims)
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	}
}
