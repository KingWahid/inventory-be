package handler

import "github.com/KingWahid/inventory/backend/services/inventory/domains/audit/usecase"

// Handler exposes delivery adapter for audit domain.
type Handler struct {
	uc usecase.Usecase
}

// New creates audit delivery handler.
func New(uc usecase.Usecase) *Handler {
	return &Handler{uc: uc}
}

// Ping is a placeholder method to prove dependency wiring.
func (h *Handler) Ping() error {
	return h.uc.Ping()
}
