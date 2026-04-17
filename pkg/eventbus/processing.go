package eventbus

import (
	"context"
	"time"
)

type HandleOptions struct {
	// IdempotencyKey, if non-empty, gates processing with SETNX + TTL.
	// Use a stable business key (e.g. outbox row id) so duplicates ACK without side effects.
	IdempotencyKey string
	IdempotencyTTL time.Duration

	MaxRetry  int
	DLQStream string

	// OnDuplicate is optional; called when IdempotencyKey already exists (duplicate delivery).
	OnDuplicate func(ctx context.Context, msg EventMessage)
}

type HandleResult struct {
	Acked       bool
	Skipped     bool
	RetryResult RetryResult
}

func (c *Client) HandleMessage(
	ctx context.Context,
	msg EventMessage,
	opts HandleOptions,
	handler func(context.Context, EventMessage) error,
) (HandleResult, error) {
	if opts.DLQStream == "" {
		opts.DLQStream = DLQStream(msg.Stream)
	}

	if opts.IdempotencyKey != "" {
		acquired, err := c.AcquireIdempotency(ctx, opts.IdempotencyKey, opts.IdempotencyTTL)
		if err != nil {
			return HandleResult{}, err
		}
		if !acquired {
			if opts.OnDuplicate != nil {
				opts.OnDuplicate(ctx, msg)
			}
			if err := c.Ack(ctx, msg.Stream, msg.Group, msg.ID); err != nil {
				return HandleResult{}, err
			}
			return HandleResult{Acked: true, Skipped: true}, nil
		}
	}

	processErr := handler(ctx, msg)
	if processErr == nil {
		if err := c.Ack(ctx, msg.Stream, msg.Group, msg.ID); err != nil {
			return HandleResult{}, err
		}
		return HandleResult{Acked: true}, nil
	}

	var rr RetryResult
	var err error
	if IsPermanent(processErr) {
		rr, err = c.MoveToDLQ(ctx, opts.DLQStream, msg.Values, processErr)
	} else {
		rr, err = c.RequeueOrDLQ(ctx, msg.Stream, opts.DLQStream, msg.Values, opts.MaxRetry, processErr)
	}
	if err != nil {
		return HandleResult{}, err
	}

	if err := c.Ack(ctx, msg.Stream, msg.Group, msg.ID); err != nil {
		return HandleResult{}, err
	}

	return HandleResult{
		Acked:       true,
		RetryResult: rr,
	}, nil
}

func (c *Client) HandleWithRetry(
	ctx context.Context,
	msg EventMessage,
	maxRetry int,
	dlqStream string,
	handler func(context.Context, EventMessage) error,
) (HandleResult, error) {
	return c.HandleMessage(ctx, msg, HandleOptions{
		MaxRetry:  maxRetry,
		DLQStream: dlqStream,
	}, handler)
}
