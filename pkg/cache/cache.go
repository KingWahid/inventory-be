// Package cache provides cache-aside helpers for ARCHITECTURE §13 (Redis TTL + invalidation).
//
// Manual QA: after product/category/warehouse writes, list endpoints must not serve stale rows;
// after movement confirm, dashboard summary should refresh within TTL or immediately after invalidate.
package cache

import (
	"context"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// Cache is a minimal key/value cache (typically Redis behind cache-aside).
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Set(ctx context.Context, key string, val []byte, ttl time.Duration) error
	Delete(ctx context.Context, keys ...string) error
	DeletePattern(ctx context.Context, pattern string) error
}

// NewRedis wraps a Redis client; returns Noop when client is nil (dev without REDIS_ADDR).
func NewRedis(rdb *goredis.Client) Cache {
	if rdb == nil {
		return Noop{}
	}
	return Redis{rdb: rdb}
}
