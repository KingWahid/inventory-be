package api

import "github.com/KingWahid/inventory/backend/services/notification/service"

// ServerHandler holds HTTP handlers and service dependencies.
type ServerHandler struct {
	svc service.Service
}

// NewServerHandler constructs the API handler bundle.
func NewServerHandler(svc service.Service) *ServerHandler {
	return &ServerHandler{svc: svc}
}
