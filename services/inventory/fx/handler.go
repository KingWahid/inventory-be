// Package fx provides fx modules for the inventory service.
package fx

import (
	"github.com/labstack/echo/v4"
	uberfx "go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/your-org/inventory/backend/services/inventory/api"
	"github.com/your-org/inventory/backend/services/inventory/service"
)

// HandlerParams holds dependencies for route registration (billing-style fx.In bundle).
type HandlerParams struct {
	uberfx.In

	Echo *echo.Echo
	Svc  service.Service
	Log  *zap.Logger
}

// RegisterRoutes wires ServerHandler and mounts routes on Echo.
// When OpenAPI codegen exists, call stub.RegisterHandlers(params.Echo, handler) here instead of api.Register.
func RegisterRoutes(params HandlerParams) {
	params.Log.Debug("registering inventory routes")

	handler := api.NewServerHandler(params.Svc)
	api.Register(params.Echo, handler)

	params.Log.Info("inventory routes registered")
}

// HandlerModule invokes route registration after Echo and Service are available.
var HandlerModule = uberfx.Module("handler",
	uberfx.Invoke(RegisterRoutes),
)
