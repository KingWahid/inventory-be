package fx

import (
	uberfx "go.uber.org/fx"

	infpostgres "github.com/KingWahid/inventory/backend/infra/postgres"
	infraredis "github.com/KingWahid/inventory/backend/infra/redis"
	commonlogger "github.com/KingWahid/inventory/backend/pkg/common/logger"
	"github.com/KingWahid/inventory/backend/services/inventory/api"
	"github.com/KingWahid/inventory/backend/services/inventory/config"
	"github.com/KingWahid/inventory/backend/services/inventory/domains/audit"
	"github.com/KingWahid/inventory/backend/services/inventory/domains/catalog"
	"github.com/KingWahid/inventory/backend/services/inventory/domains/movement"
	"github.com/KingWahid/inventory/backend/services/inventory/domains/stock"
	"github.com/KingWahid/inventory/backend/services/inventory/domains/warehouse"
	"github.com/KingWahid/inventory/backend/services/inventory/service"
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
	audit.Module,
	catalog.Module,
	movement.Module,
	stock.Module,
	warehouse.Module,
	service.Module,
	api.EchoModule,
	HandlerModule,
)
