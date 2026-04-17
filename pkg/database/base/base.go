package base

import (
	"context"
	"time"
)

type RepositoryBase struct {
	timeout time.Duration
}

func New(timeout time.Duration) *RepositoryBase {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &RepositoryBase{timeout: timeout}
}

func (b *RepositoryBase) WithTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, b.timeout)
}
