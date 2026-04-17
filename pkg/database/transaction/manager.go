package transaction

import (
	"context"

	"gorm.io/gorm"
)

// Manager provides service-layer transaction orchestration.
type Manager interface {
	RunInTx(ctx context.Context, fn func(context.Context) error) error
}

type manager struct {
	db *gorm.DB
}

// NewManager creates a transaction manager backed by gorm.DB.
func NewManager(db *gorm.DB) Manager {
	return &manager{db: db}
}

func (m *manager) RunInTx(ctx context.Context, fn func(context.Context) error) error {
	return RunInTx(ctx, m.db, fn)
}
