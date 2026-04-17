package usecase

import "github.com/KingWahid/inventory/backend/services/inventory/domains/movement/repository"

// Usecase defines application logic contract for movement domain.
type Usecase interface {
	Ping() error
}

type usecase struct {
	repo repository.Repository
}

// New creates movement usecase implementation.
func New(repo repository.Repository) Usecase {
	return &usecase{repo: repo}
}

func (u *usecase) Ping() error {
	return u.repo.Ping()
}
