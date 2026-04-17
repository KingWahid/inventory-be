package base

import (
	"context"
	"encoding/json"
	"time"
)

type Cache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
}

// QueryFunc executes a DB query and fills data into the provided result pointer.
type QueryFunc[T any] func(ctx context.Context, out *T) error

func GetFromCacheOrDB[T any](
	ctx context.Context,
	c Cache,
	key string,
	ttl time.Duration,
	dbFn func(context.Context) (T, error),
) (T, error) {
	var zero T

	if c != nil {
		if raw, err := c.Get(ctx, key); err == nil && raw != "" {
			var out T
			if unmarshalErr := json.Unmarshal([]byte(raw), &out); unmarshalErr == nil {
				return out, nil
			}
		}
	}

	fromDB, err := dbFn(ctx)
	if err != nil {
		return zero, err
	}

	if c != nil && ttl > 0 {
		if encoded, marshalErr := json.Marshal(fromDB); marshalErr == nil {
			_ = c.Set(ctx, key, string(encoded), ttl)
		}
	}

	return fromDB, nil
}

// GetFromCacheOrDBInto is docs-friendly when query code naturally fills a pointer.
func GetFromCacheOrDBInto[T any](
	ctx context.Context,
	c Cache,
	key string,
	ttl time.Duration,
	queryFn QueryFunc[T],
) (T, error) {
	return GetFromCacheOrDB(
		ctx,
		c,
		key,
		ttl,
		func(ctx context.Context) (T, error) {
			var out T
			if err := queryFn(ctx, &out); err != nil {
				var zero T
				return zero, err
			}
			return out, nil
		},
	)
}
