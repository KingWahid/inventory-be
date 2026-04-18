package fx

import (
	uberfx "go.uber.org/fx"

	infpostgres "github.com/KingWahid/inventory/backend/infra/postgres"
	infraredis "github.com/KingWahid/inventory/backend/infra/redis"
	commonlogger "github.com/KingWahid/inventory/backend/pkg/common/logger"
	"github.com/KingWahid/inventory/backend/services/notification/api"
	"github.com/KingWahid/inventory/backend/services/notification/config"
	"github.com/KingWahid/inventory/backend/services/notification/service"
)

// Module composes notification service dependencies.
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
	uberfx.Invoke(RegisterStreamConsumer),
)
