package fx

import (
	goredis "github.com/redis/go-redis/v9"
	uberfx "go.uber.org/fx"

	infpostgres "github.com/KingWahid/inventory/backend/infra/postgres"
	infraredis "github.com/KingWahid/inventory/backend/infra/redis"
	cachepkg "github.com/KingWahid/inventory/backend/pkg/cache"
	commonlogger "github.com/KingWahid/inventory/backend/pkg/common/logger"
	"github.com/KingWahid/inventory/backend/services/inventory/api"
	"github.com/KingWahid/inventory/backend/services/inventory/config"
	"github.com/KingWahid/inventory/backend/services/inventory/domains/audit"
	"github.com/KingWahid/inventory/backend/services/inventory/domains/catalog"
	"github.com/KingWahid/inventory/backend/services/inventory/domains/dashboard"
	"github.com/KingWahid/inventory/backend/services/inventory/domains/movement"
	"github.com/KingWahid/inventory/backend/services/inventory/domains/outbox"
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
		func(rdb *goredis.Client) cachepkg.Cache {
			return cachepkg.NewRedis(rdb)
		},
	),
	commonlogger.Module,
	infpostgres.FxModule(),
	infraredis.FxModule(),
	audit.Module,
	catalog.Module,
	dashboard.Module,
	movement.Module,
	outbox.Module,
	stock.Module,
	warehouse.Module,
	service.Module,
	api.EchoModule,
	HandlerModule,
)
