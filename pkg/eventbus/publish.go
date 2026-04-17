package eventbus

import (
	"context"
	"errors"
	"time"

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

// PublishEvent publishes a typed event envelope to Redis Streams.
func (c *Client) PublishEvent(ctx context.Context, event BaseEvent) (string, error) {
	if event.Stream == "" {
		return "", errors.New("eventbus: event stream is required")
	}
	if event.Version <= 0 {
		event.Version = 1
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}
	if event.PublishedAt.IsZero() {
		event.PublishedAt = event.CreatedAt
	}
	values, err := event.ToValues()
	if err != nil {
		return "", err
	}
	return c.Publish(ctx, event.Stream, values)
}
