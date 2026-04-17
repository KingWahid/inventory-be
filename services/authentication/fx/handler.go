package fx

import (
	"github.com/KingWahid/inventory/backend/services/authentication/api"
	"github.com/labstack/echo/v4"
	uberfx "go.uber.org/fx"
	"go.uber.org/zap"
)

// HandlerParams holds dependencies for route registration.
type HandlerParams struct {
	uberfx.In

	Echo *echo.Echo
	Log  *zap.Logger
	H    *api.ServerHandler
}

// RegisterRoutes mounts generated routes onto Echo.
func RegisterRoutes(params HandlerParams) {
	params.Log.Debug("registering authentication routes")
	params.Echo.GET("/health", params.H.GetHealth)
	params.Echo.GET("/ready", params.H.GetReady)
	params.Echo.GET("/api/v1/auth/health", params.H.GetApiV1AuthHealth)
	params.Echo.POST("/api/v1/auth/register", params.H.PostApiV1AuthRegister)
	params.Log.Info("authentication routes registered")
}
