package eventbus

import (
	"context"
	"errors"
	"fmt"
	"time"
)

func BuildIdempotencyKey(message EventMessage) string {
	eventID := message.ID
	if event, err := DecodeEvent(message); err == nil && event.ID != "" {
		eventID = event.ID
	}
	return fmt.Sprintf("eventbus:idempotency:%s:%s:%s", message.Stream, message.Group, eventID)
}

func (c *Client) AcquireIdempotency(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	if key == "" {
		return false, errors.New("eventbus: idempotency key is required")
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}

	ok, err := c.rdb.SetNX(ctx, key, "1", ttl).Result()
	if err != nil {
		return false, err
	}
	return ok, nil
}
