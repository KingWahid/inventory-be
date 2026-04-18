package api

import (
	commonjwt "github.com/KingWahid/inventory/backend/pkg/common/jwt"
	"github.com/labstack/echo/v4"
)

// AuthPublicPaths are routes that skip JWT validation (must match URL.Path exactly).
var AuthPublicPaths = map[string]struct{}{
	"/health":                  {},
	"/ready":                   {},
	"/api/v1/auth/health":      {},
	"/api/v1/auth/login":       {},
	"/api/v1/auth/register":    {},
	"/api/v1/auth/refresh":     {},
}

// RequireAccessJWT validates Bearer access tokens and attaches claims to request.Context().
func RequireAccessJWT(jwtSvc *commonjwt.Service) echo.MiddlewareFunc {
	return commonjwt.RequireBearerAccessJWT(jwtSvc, AuthPublicPaths)
}
