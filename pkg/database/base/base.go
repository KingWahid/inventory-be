package base

import (
	"context"
	"time"
)

// Repository is the base struct intended to be embedded by repositories.
type Repository struct {
	timeout time.Duration
}

// RepositoryBase is kept as a backward-compatible alias.
type RepositoryBase = Repository

// NewRepository creates a Repository with sane default timeout.
func NewRepository(timeout time.Duration) *Repository {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &Repository{timeout: timeout}
}

// New is kept for backward compatibility.
// Deprecated: use NewRepository.
func New(timeout time.Duration) *Repository {
	return NewRepository(timeout)
}

func (b *Repository) WithTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, b.timeout)
}
