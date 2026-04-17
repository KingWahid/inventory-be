package service

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// AuthenticationService implements Service using PostgreSQL.
type AuthenticationService struct {
	db *gorm.DB
}

// NewAuthenticationService constructs the default authentication service.
func NewAuthenticationService(db *gorm.DB) *AuthenticationService {
	return &AuthenticationService{db: db}
}

// PingDB checks database connectivity.
func (s *AuthenticationService) PingDB(ctx context.Context) error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return fmt.Errorf("authentication service: sql db: %w", err)
	}
	return sqlDB.PingContext(ctx)
}
