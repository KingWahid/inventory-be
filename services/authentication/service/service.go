package service

import "context"

// Service is the authentication application facade.
type Service interface {
	PingDB(ctx context.Context) error
}
