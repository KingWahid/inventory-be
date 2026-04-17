package eventbus

import (
	"context"
	"errors"

	"github.com/redis/go-redis/v9"
)

func (c *Client) Publish(ctx context.Context, stream string, payload map[string]any) (string, error) {
	if stream == "" {
		return "", errors.New("eventbus: stream is required")
	}
	if len(payload) == 0 {
		return "", errors.New("eventbus: payload is required")
	}

	id, err := c.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		ID:     "*",
		Values: payload,
	}).Result()
	if err != nil {
		return "", err
	}

	return id, nil
}
