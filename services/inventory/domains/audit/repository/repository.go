package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/KingWahid/inventory/backend/pkg/database/transaction"
	"gorm.io/gorm"
)

// Repository defines data-access contract for audit domain.
type Repository interface {
	Ping() error
	Insert(ctx context.Context, in InsertInput) error
}

// InsertInput maps to audit_logs.
type InsertInput struct {
	TenantID   string
	UserID     *string
	Action     string
	Entity     string
	EntityID   string // UUID string
	BeforeData []byte
	AfterData  []byte
	IPAddress  *string
	UserAgent  *string
	RequestID  *string
}

type auditLogRow struct {
	ID         string  `gorm:"column:id;type:uuid;primaryKey"`
	TenantID   string  `gorm:"column:tenant_id;type:uuid"`
	UserID     *string `gorm:"column:user_id;type:uuid"`
	Action     string  `gorm:"column:action"`
	Entity     string  `gorm:"column:entity"`
	EntityID   string  `gorm:"column:entity_id;type:uuid"`
	BeforeData []byte  `gorm:"column:before_data;type:jsonb"`
	AfterData  []byte  `gorm:"column:after_data;type:jsonb"`
	IPAddress  *string `gorm:"column:ip_address"`
	UserAgent  *string `gorm:"column:user_agent"`
	RequestID  *string `gorm:"column:request_id"`
}

func (auditLogRow) TableName() string { return "audit_logs" }

type repository struct {
	db *gorm.DB
}

// New creates audit repository implementation.
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

func (r *repository) Insert(ctx context.Context, in InsertInput) error {
	tx := transaction.GetDB(ctx, r.db).WithContext(ctx)
	row := auditLogRow{
		ID:         uuid.New().String(),
		TenantID:   in.TenantID,
		UserID:     in.UserID,
		Action:     in.Action,
		Entity:     in.Entity,
		EntityID:   in.EntityID,
		BeforeData: in.BeforeData,
		AfterData:  in.AfterData,
		IPAddress:  in.IPAddress,
		UserAgent:  in.UserAgent,
		RequestID:  in.RequestID,
	}
	return tx.Create(&row).Error
}
