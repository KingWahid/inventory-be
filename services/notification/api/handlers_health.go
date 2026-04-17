package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// GetHealth handles GET /health.
func (h *ServerHandler) GetHealth(c echo.Context) error {
	return c.String(http.StatusOK, "ok")
}

// GetApiV1NotificationsHealth handles GET /api/v1/notifications/health.
func (h *ServerHandler) GetApiV1NotificationsHealth(c echo.Context) error {
	return c.String(http.StatusOK, "ok")
}

// GetReady handles GET /ready.
func (h *ServerHandler) GetReady(c echo.Context) error {
	if err := h.svc.PingDB(c.Request().Context()); err != nil {
		zap.L().Warn("notification readiness failed", zap.Error(err))
		return c.String(http.StatusServiceUnavailable, "db not ready")
	}
	return c.String(http.StatusOK, "ok")
}
