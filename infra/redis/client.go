package redis

import (
	"context"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/fx"
)

// NewClient returns a Redis client when addr is non-empty; otherwise nil.
func NewClient(lc fx.Lifecycle, addr string) *goredis.Client {
	if addr == "" {
		return nil
	}

	rdb := goredis.NewClient(&goredis.Options{
		Addr:         addr,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	})

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return rdb.Ping(ctx).Err()
		},
		OnStop: func(context.Context) error {
			return rdb.Close()
		},
	})

	return rdb
}
