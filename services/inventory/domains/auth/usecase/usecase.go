package usecase

import "github.com/KingWahid/inventory/backend/services/inventory/domains/auth/repository"

// Usecase defines application logic contract for auth domain.
type Usecase interface {
	Ping() error
}

type usecase struct {
	repo repository.Repository
}

// New creates auth usecase implementation.
func New(repo repository.Repository) Usecase {
	return &usecase{repo: repo}
}

func (u *usecase) Ping() error {
	return u.repo.Ping()
}
