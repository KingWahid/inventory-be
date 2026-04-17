package handler

import "github.com/KingWahid/inventory/backend/services/inventory/domains/auth/usecase"

// Handler exposes delivery adapter for auth domain.
type Handler struct {
	uc usecase.Usecase
}

// New creates auth delivery handler.
func New(uc usecase.Usecase) *Handler {
	return &Handler{uc: uc}
}

// Ping is a placeholder method to prove dependency wiring.
func (h *Handler) Ping() error {
	return h.uc.Ping()
}
