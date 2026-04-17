package service

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// NotificationService implements Service using PostgreSQL.
type NotificationService struct {
	db *gorm.DB
}

// NewNotificationService constructs the default notification service.
func NewNotificationService(db *gorm.DB) *NotificationService {
	return &NotificationService{db: db}
}

// PingDB checks database connectivity.
func (s *NotificationService) PingDB(ctx context.Context) error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return fmt.Errorf("notification service: sql db: %w", err)
	}
	return sqlDB.PingContext(ctx)
}
