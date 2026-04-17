package api

import (
	"github.com/your-org/inventory/backend/services/inventory/service"
)

// ServerHandler holds HTTP handlers and their application dependencies (billing-style).
type ServerHandler struct {
	svc service.Service
}

// NewServerHandler constructs the API handler bundle.
func NewServerHandler(svc service.Service) *ServerHandler {
	return &ServerHandler{svc: svc}
}
