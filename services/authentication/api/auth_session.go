package api

import (
	"github.com/KingWahid/inventory/backend/pkg/common/errorcodes"
	"github.com/labstack/echo/v4"
)

// PostApiV1AuthRefresh handles POST /api/v1/auth/refresh.
func (h *ServerHandler) PostApiV1AuthRefresh(c echo.Context) error {
	_ = c
	return errorcodes.ErrNotImplemented
}

// PostApiV1AuthLogout handles POST /api/v1/auth/logout.
func (h *ServerHandler) PostApiV1AuthLogout(c echo.Context) error {
	_ = c
	return errorcodes.ErrNotImplemented
}

// GetApiV1AuthMe handles GET /api/v1/auth/me.
func (h *ServerHandler) GetApiV1AuthMe(c echo.Context) error {
	_ = c
	return errorcodes.ErrNotImplemented
}
