package api

import (
	"context"
	"net/http"

	"github.com/KingWahid/inventory/backend/pkg/common/httpresponse"
	"github.com/KingWahid/inventory/backend/services/authentication/service"
	"github.com/KingWahid/inventory/backend/services/authentication/stub"
	"github.com/labstack/echo/v4"
)

// PostApiV1AuthRefresh handles POST /api/v1/auth/refresh.
func (h *ServerHandler) PostApiV1AuthRefresh(c echo.Context) error {
	var req stub.RefreshRequest
	if err := c.Bind(&req); err != nil {
		return err
	}

	ctxTimeout, cancel := context.WithTimeout(c.Request().Context(), endpointTimeout)
	defer cancel()

	result, err := h.service.Refresh(ctxTimeout, service.RefreshInput{RefreshToken: req.RefreshToken})
	if err != nil {
		return err
	}

	return httpresponse.OK(c, http.StatusOK, map[string]any{
		"access_token":  result.AccessToken,
		"refresh_token": result.RefreshToken,
		"token_type":    result.TokenType,
		"expires_in":    result.ExpiresIn,
	})
}

// PostApiV1AuthLogout handles POST /api/v1/auth/logout.
func (h *ServerHandler) PostApiV1AuthLogout(c echo.Context) error {
	ctxTimeout, cancel := context.WithTimeout(c.Request().Context(), endpointTimeout)
	defer cancel()

	if err := h.service.Logout(ctxTimeout); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

// GetApiV1AuthMe handles GET /api/v1/auth/me.
func (h *ServerHandler) GetApiV1AuthMe(c echo.Context) error {
	ctxTimeout, cancel := context.WithTimeout(c.Request().Context(), endpointTimeout)
	defer cancel()

	me, err := h.service.Me(ctxTimeout)
	if err != nil {
		return err
	}

	return httpresponse.OK(c, http.StatusOK, map[string]any{
		"user_id":   me.UserID,
		"tenant_id": me.TenantID,
		"email":     me.Email,
		"full_name": me.FullName,
	})
}
