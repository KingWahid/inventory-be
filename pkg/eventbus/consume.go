package eventbus

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

func (c *Client) EnsureGroup(ctx context.Context, stream, group string) error {
	if stream == "" || group == "" {
		return errors.New("eventbus: stream and group are required")
	}

	err := c.rdb.XGroupCreateMkStream(ctx, stream, group, "0").Err()
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "BUSYGROUP") {
		return nil
	}
	return err
}

func (c *Client) ReadGroup(
	ctx context.Context,
	stream, group, consumer string,
	count int64,
	block time.Duration,
) ([]EventMessage, error) {
	if stream == "" || group == "" || consumer == "" {
		return nil, errors.New("eventbus: stream, group, and consumer are required")
	}
	if count <= 0 {
		count = 1
	}

	streams, err := c.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    group,
		Consumer: consumer,
		Streams:  []string{stream, ">"},
		Count:    count,
		Block:    block,
		NoAck:    false,
	}).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return []EventMessage{}, nil
		}
		return nil, err
	}

	result := make([]EventMessage, 0)
	for _, s := range streams {
		for _, msg := range s.Messages {
			result = append(result, messageFromRedis(s.Stream, group, consumer, msg))
		}
	}
	return result, nil
}

func (c *Client) Ack(ctx context.Context, stream, group string, ids ...string) error {
	if stream == "" || group == "" {
		return errors.New("eventbus: stream and group are required")
	}
	if len(ids) == 0 {
		return nil
	}
	return c.rdb.XAck(ctx, stream, group, ids...).Err()
}
