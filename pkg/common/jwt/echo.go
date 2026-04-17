package jwt

import (
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/your-org/inventory/backend/pkg/common/errorcodes"
)

// GetJWTFromEchoContext returns the raw Bearer token from the Authorization header.
func GetJWTFromEchoContext(c echo.Context) (string, error) {
	h := c.Request().Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(h, prefix) {
		return "", errorcodes.ErrUnauthorized
	}
	t := strings.TrimSpace(strings.TrimPrefix(h, prefix))
	if t == "" {
		return "", errorcodes.ErrUnauthorized
	}
	return t, nil
}
