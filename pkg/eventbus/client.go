package eventbus

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

type Option func(*redis.Options)

type Client struct {
	rdb *redis.Client
}

func New(redisAddr string, opts ...Option) (*Client, error) {
	if redisAddr == "" {
		return nil, errors.New("eventbus: redis address is required")
	}

	options := &redis.Options{
		Addr:         redisAddr,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	for _, opt := range opts {
		opt(options)
	}

	return &Client{
		rdb: redis.NewClient(options),
	}, nil
}

func (c *Client) Close() error {
	return c.rdb.Close()
}

func (c *Client) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}
