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
