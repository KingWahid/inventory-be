package usecase

import (
	"context"

	"github.com/KingWahid/inventory/backend/services/inventory/domains/audit/repository"
)

// Usecase defines application logic contract for audit domain.
type Usecase interface {
	Ping() error
	ListAuditLogs(ctx context.Context, in ListAuditLogsInput) (ListAuditLogsOutput, error)
}

type usecase struct {
	repo repository.Repository
}

// New creates audit usecase implementation.
func New(repo repository.Repository) Usecase {
	return &usecase{repo: repo}
}

func (u *usecase) Ping() error {
	return u.repo.Ping()
}
