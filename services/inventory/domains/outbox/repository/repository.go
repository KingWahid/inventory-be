// Package repository persists outbox_events using the tx from context (transaction.GetDB).
package repository

import (
	"context"

	"github.com/KingWahid/inventory/backend/pkg/database/transaction"
	"gorm.io/gorm"
)

// Repository inserts outbox rows within the caller's transaction.
type Repository interface {
	Ping() error
	Insert(ctx context.Context, in InsertInput) error
}

// InsertInput maps to outbox_events (published defaults false).
type InsertInput struct {
	TenantID      string
	EventType     string // movement/usecase: EventTypeMovementCreated, EventTypeStockChanged, EventTypeStockBelowThreshold (ARCHITECTURE §10)
	AggregateType string // e.g. movement
	AggregateID   string // UUID string
	Payload       []byte // JSON
}

type repository struct {
	db *gorm.DB
}

// New creates an outbox repository.
func New(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Ping() error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

type outboxEventRow struct {
	ID            int64  `gorm:"column:id;primaryKey;autoIncrement"`
	TenantID      string `gorm:"column:tenant_id;type:uuid"`
	EventType     string `gorm:"column:event_type"`
	AggregateType string `gorm:"column:aggregate_type"`
	AggregateID   string `gorm:"column:aggregate_id;type:uuid"`
	Payload       []byte `gorm:"column:payload;type:jsonb"`
	Published     bool   `gorm:"column:published"`
}

func (outboxEventRow) TableName() string { return "outbox_events" }

func (r *repository) Insert(ctx context.Context, in InsertInput) error {
	tx := transaction.GetDB(ctx, r.db).WithContext(ctx)
	row := outboxEventRow{
		TenantID:      in.TenantID,
		EventType:     in.EventType,
		AggregateType: in.AggregateType,
		AggregateID:   in.AggregateID,
		Payload:       in.Payload,
		Published:     false,
	}
	return tx.Create(&row).Error
}
