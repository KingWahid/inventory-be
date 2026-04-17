package eventbus

import (
	"context"
	"fmt"
)

type RetryResult struct {
	Requeued   bool
	DeadLetter bool
	MessageID  string
	RetryCount int
}

func WithRetryMetadata(values map[string]any, retryCount int, lastErr error) map[string]any {
	out := make(map[string]any, len(values)+2)
	for k, v := range values {
		out[k] = v
	}
	out["retry_count"] = retryCount
	if lastErr != nil {
		out["last_error"] = lastErr.Error()
	}
	return out
}

func (c *Client) RequeueOrDLQ(
	ctx context.Context,
	stream string,
	dlqStream string,
	values map[string]any,
	maxRetry int,
	lastErr error,
) (RetryResult, error) {
	if dlqStream == "" {
		dlqStream = DLQStream(stream)
	}

	currentRetry := RetryCount(values)
	nextRetry := currentRetry + 1
	if nextRetry <= maxRetry {
		payload := WithRetryMetadata(values, nextRetry, lastErr)
		id, err := c.Publish(ctx, stream, payload)
		if err != nil {
			return RetryResult{}, err
		}
		return RetryResult{
			Requeued:   true,
			MessageID:  id,
			RetryCount: nextRetry,
		}, nil
	}

	payload := WithRetryMetadata(values, nextRetry, lastErr)
	payload["dlq_reason"] = fmt.Sprintf("retry exhausted (max=%d)", maxRetry)

	id, err := c.Publish(ctx, dlqStream, payload)
	if err != nil {
		return RetryResult{}, err
	}
	return RetryResult{
		DeadLetter: true,
		MessageID:  id,
		RetryCount: nextRetry,
	}, nil
}

func (c *Client) MoveToDLQ(
	ctx context.Context,
	dlqStream string,
	values map[string]any,
	lastErr error,
) (RetryResult, error) {
	if dlqStream == "" {
		dlqStream = DLQStream(StreamInventoryEvents())
	}

	payload := WithRetryMetadata(values, RetryCount(values), lastErr)
	payload["dlq_reason"] = "permanent_error"

	id, err := c.Publish(ctx, dlqStream, payload)
	if err != nil {
		return RetryResult{}, err
	}
	return RetryResult{
		DeadLetter: true,
		MessageID:  id,
		RetryCount: RetryCount(payload),
	}, nil
}
