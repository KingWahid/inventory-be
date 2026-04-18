package api

import (
	"net/http"

	"github.com/KingWahid/inventory/backend/services/authentication/stub"
	"github.com/labstack/echo/v4"
)

const msgNotImplemented = "endpoint not implemented yet"

// PostApiV1AuthRefresh handles POST /api/v1/auth/refresh.
func (h *ServerHandler) PostApiV1AuthRefresh(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, stub.ErrorResponse{
		Code:    "NOT_IMPLEMENTED",
		Message: msgNotImplemented,
	})
}

// PostApiV1AuthLogout handles POST /api/v1/auth/logout.
func (h *ServerHandler) PostApiV1AuthLogout(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, stub.ErrorResponse{
		Code:    "NOT_IMPLEMENTED",
		Message: msgNotImplemented,
	})
}

// GetApiV1AuthMe handles GET /api/v1/auth/me.
func (h *ServerHandler) GetApiV1AuthMe(c echo.Context) error {
	return c.JSON(http.StatusNotImplemented, stub.ErrorResponse{
		Code:    "NOT_IMPLEMENTED",
		Message: msgNotImplemented,
	})
}
