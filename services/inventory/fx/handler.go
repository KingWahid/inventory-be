// Package fx provides fx modules for the inventory service.
package fx

import (
	"github.com/labstack/echo/v4"
	goredis "github.com/redis/go-redis/v9"
	uberfx "go.uber.org/fx"
	"go.uber.org/zap"

	commonjwt "github.com/KingWahid/inventory/backend/pkg/common/jwt"
	"github.com/KingWahid/inventory/backend/services/inventory/api"
	"github.com/KingWahid/inventory/backend/services/inventory/service"
	"github.com/KingWahid/inventory/backend/services/inventory/stub"
)

// HandlerParams holds dependencies for route registration (billing-style fx.In bundle).
type HandlerParams struct {
	uberfx.In

	Echo *echo.Echo
	Svc  service.Service
	Log  *zap.Logger
	Rdb  *goredis.Client
	Jwt  *commonjwt.Service
}

// RegisterRoutes wires ServerHandler and mounts routes from OpenAPI (stub.RegisterHandlers).
func RegisterRoutes(params HandlerParams) {
	params.Log.Debug("registering inventory routes")

	handler := api.NewServerHandler(params.Svc)
	stub.RegisterHandlers(params.Echo, handler)
	params.Echo.GET("/api/v1/inventory/sse/stock", api.SSEStock(params.Rdb, params.Jwt))
	params.Echo.GET("/api/v1/inventory/sse/activity", api.SSEActivity(params.Svc, params.Jwt))
	params.Echo.GET("/api/v1/inventory/dashboard/storage-utilization", handler.GetDashboardStorageUtilization)

	params.Log.Info("inventory routes registered")
}

// HandlerModule invokes route registration after Echo and Service are available.
var HandlerModule = uberfx.Module("handler",
	uberfx.Invoke(RegisterRoutes),
)
