package redis

import (
	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/fx"
)

// RedisConfig is satisfied by service configs that expose a Redis address (e.g. REDIS_ADDR).
type RedisConfig interface {
	GetRedisAddr() string
}

// FxModule registers an optional *goredis.Client from any provided RedisConfig.
func FxModule() fx.Option {
	return fx.Module("redis",
		fx.Provide(func(lc fx.Lifecycle, cfg RedisConfig) *goredis.Client {
			return NewClient(lc, cfg.GetRedisAddr())
		}),
		fx.Invoke(func(*goredis.Client) {}),
	)
}
