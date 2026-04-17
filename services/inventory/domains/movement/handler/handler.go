package handler

import "github.com/KingWahid/inventory/backend/services/inventory/domains/movement/usecase"

// Handler exposes delivery adapter for movement domain.
type Handler struct {
	uc usecase.Usecase
}

// New creates movement delivery handler.
func New(uc usecase.Usecase) *Handler {
	return &Handler{uc: uc}
}

// Ping is a placeholder method to prove dependency wiring.
func (h *Handler) Ping() error {
	return h.uc.Ping()
}
