package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// GetHealth handles GET /health (liveness).
func (h *ServerHandler) GetHealth(c echo.Context) error {
	return c.String(http.StatusOK, "ok")
}

// GetInventoryHealth handles GET /api/v1/inventory/health.
func (h *ServerHandler) GetInventoryHealth(c echo.Context) error {
	return c.String(http.StatusOK, "ok")
}

// GetReady handles GET /ready (readiness — DB ping via service).
func (h *ServerHandler) GetReady(c echo.Context) error {
	if err := h.svc.PingDB(c.Request().Context()); err != nil {
		zap.L().Warn("readiness check failed", zap.Error(err))
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "database unavailable",
		})
	}
	return c.String(http.StatusOK, "ok")
}
