package handler

import "github.com/KingWahid/inventory/backend/services/inventory/domains/catalog/usecase"

// Handler exposes delivery adapter for catalog domain.
type Handler struct {
	uc usecase.Usecase
}

// New creates catalog delivery handler.
func New(uc usecase.Usecase) *Handler {
	return &Handler{uc: uc}
}

// Ping is a placeholder method to prove dependency wiring.
func (h *Handler) Ping() error {
	return h.uc.Ping()
}
