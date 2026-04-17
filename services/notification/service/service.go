package service

import "context"

// Service is the notification application facade.
type Service interface {
	PingDB(ctx context.Context) error
}
