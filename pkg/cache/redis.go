package cache

import (
	"context"
	"errors"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// Redis implements Cache using Redis strings with TTL.
type Redis struct {
	rdb *goredis.Client
}

func (r Redis) Get(ctx context.Context, key string) ([]byte, bool, error) {
	s, err := r.rdb.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return s, true, nil
}

func (r Redis) Set(ctx context.Context, key string, val []byte, ttl time.Duration) error {
	if ttl <= 0 {
		return r.rdb.Set(ctx, key, val, 0).Err()
	}
	return r.rdb.Set(ctx, key, val, ttl).Err()
}

func (r Redis) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return r.rdb.Del(ctx, keys...).Err()
}

// DeletePattern removes keys matching pattern using SCAN (non-blocking for huge keyspaces).
func (r Redis) DeletePattern(ctx context.Context, pattern string) error {
	if pattern == "" {
		return nil
	}
	var cursor uint64
	for {
		var keys []string
		var err error
		keys, cursor, err = r.rdb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			if err := r.rdb.Del(ctx, keys...).Err(); err != nil {
				return err
			}
		}
		if cursor == 0 {
			break
		}
	}
	return nil
}
