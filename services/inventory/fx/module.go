package fx

import (
	uberfx "go.uber.org/fx"

	"github.com/your-org/inventory/backend/services/inventory/api"
	"github.com/your-org/inventory/backend/services/inventory/config"
	"github.com/your-org/inventory/backend/services/inventory/database"
	"github.com/your-org/inventory/backend/services/inventory/logger"
	"github.com/your-org/inventory/backend/services/inventory/redis"
	"github.com/your-org/inventory/backend/services/inventory/service"
)

// Module composes all inventory fx modules (infra + HTTP echo + handler wiring).
var Module = uberfx.Options(
	config.Module,
	logger.Module,
	database.Module,
	redis.Module,
	service.Module,
	api.EchoModule,
	HandlerModule,
)
