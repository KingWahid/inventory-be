package fx

import (
	uberfx "go.uber.org/fx"

	infpostgres "github.com/your-org/inventory/backend/infra/postgres"
	infraredis "github.com/your-org/inventory/backend/infra/redis"
	commonlogger "github.com/your-org/inventory/backend/pkg/common/logger"
	"github.com/your-org/inventory/backend/services/inventory/api"
	"github.com/your-org/inventory/backend/services/inventory/config"
	"github.com/your-org/inventory/backend/services/inventory/service"
)

// Module composes all inventory fx modules (infra + HTTP echo + handler wiring).
var Module = uberfx.Options(
	config.Module,
	uberfx.Provide(
		uberfx.Annotate(
			func(c *config.Config) infpostgres.DBConfig { return c },
			uberfx.As(new(infpostgres.DBConfig)),
		),
		uberfx.Annotate(
			func(c *config.Config) infraredis.RedisConfig { return c },
			uberfx.As(new(infraredis.RedisConfig)),
		),
		uberfx.Annotate(
			func(c *config.Config) commonlogger.AppEnvProvider { return c },
			uberfx.As(new(commonlogger.AppEnvProvider)),
		),
	),
	commonlogger.Module,
	infpostgres.FxModule(),
	infraredis.FxModule(),
	service.Module,
	api.EchoModule,
	HandlerModule,
)
