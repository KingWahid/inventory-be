package fx

import (
	"github.com/KingWahid/inventory/backend/services/authentication/api"
	"github.com/KingWahid/inventory/backend/services/authentication/stub"
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
	stub.RegisterHandlers(params.Echo, params.H)
	params.Log.Info("authentication routes registered")
}
