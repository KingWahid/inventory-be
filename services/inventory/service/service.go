package service

import "context"

// Service is the application facade used by HTTP handlers (expand per domain module).
type Service interface {
	PingDB(ctx context.Context) error
}
