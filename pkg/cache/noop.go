package cache

import (
	"context"
	"time"
)

// Noop implements Cache with no persistence (always miss).
type Noop struct{}

func (Noop) Get(context.Context, string) ([]byte, bool, error) {
	return nil, false, nil
}

func (Noop) Set(context.Context, string, []byte, time.Duration) error {
	return nil
}

func (Noop) Delete(context.Context, ...string) error {
	return nil
}

func (Noop) DeletePattern(context.Context, string) error {
	return nil
}
