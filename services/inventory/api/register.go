package api

import (
	"github.com/labstack/echo/v4"
)

// Register attaches inventory HTTP routes to Echo (called from fx/handler after OpenAPI, or manually).
func Register(e *echo.Echo, h *ServerHandler) {
	e.GET("/health", h.GetHealth)
	e.GET("/ready", h.GetReady)

	g := e.Group("/api/v1/inventory")
	g.GET("/health", h.GetInventoryHealth)
}
