package usecase

import "github.com/KingWahid/inventory/backend/services/inventory/domains/warehouse/repository"

// Usecase defines application logic contract for warehouse domain.
type Usecase interface {
	Ping() error
}

type usecase struct {
	repo repository.Repository
}

// New creates warehouse usecase implementation.
func New(repo repository.Repository) Usecase {
	return &usecase{repo: repo}
}

func (u *usecase) Ping() error {
	return u.repo.Ping()
}
