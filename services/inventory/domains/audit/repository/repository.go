package repository

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/KingWahid/inventory/backend/pkg/database/transaction"
	"gorm.io/gorm"
)

// Repository defines data-access contract for audit domain.
type Repository interface {
	Ping() error
	Insert(ctx context.Context, in InsertInput) error
	List(ctx context.Context, tenantID string, f ListFilter) ([]Entry, int64, error)
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

// ListFilter scopes audit log queries (tenant always enforced by caller).
type ListFilter struct {
	Page    int
	PerPage int
	Entity    *string
	EntityID  *string
	Action    *string
	UserID    *string
	CreatedFrom *time.Time // inclusive
	CreatedTo   *time.Time // inclusive end-of-day handled by caller or use <
}

// Entry is one audit_logs row for API/read models.
type Entry struct {
	ID         string
	TenantID   string
	UserID     *string
	Action     string
	Entity     string
	EntityID   string
	BeforeData []byte
	AfterData  []byte
	IPAddress  *string
	UserAgent  *string
	RequestID  *string
	CreatedAt  time.Time
}

type auditLogRow struct {
	ID         string    `gorm:"column:id;type:uuid;primaryKey"`
	TenantID   string    `gorm:"column:tenant_id;type:uuid"`
	UserID     *string   `gorm:"column:user_id;type:uuid"`
	Action     string    `gorm:"column:action"`
	Entity     string    `gorm:"column:entity"`
	EntityID   string    `gorm:"column:entity_id;type:uuid"`
	BeforeData []byte    `gorm:"column:before_data;type:jsonb"`
	AfterData  []byte    `gorm:"column:after_data;type:jsonb"`
	IPAddress  *string   `gorm:"column:ip_address"`
	UserAgent  *string   `gorm:"column:user_agent"`
	RequestID  *string   `gorm:"column:request_id"`
	CreatedAt  time.Time `gorm:"column:created_at"`
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

func (r *repository) List(ctx context.Context, tenantID string, f ListFilter) ([]Entry, int64, error) {
	db := transaction.GetDB(ctx, r.db).WithContext(ctx)
	tid := strings.TrimSpace(tenantID)
	q := db.Model(&auditLogRow{}).Where("tenant_id = ?", tid)

	if f.Entity != nil && strings.TrimSpace(*f.Entity) != "" {
		q = q.Where("entity = ?", strings.TrimSpace(*f.Entity))
	}
	if f.EntityID != nil && strings.TrimSpace(*f.EntityID) != "" {
		q = q.Where("entity_id = ?", strings.TrimSpace(*f.EntityID))
	}
	if f.Action != nil && strings.TrimSpace(*f.Action) != "" {
		q = q.Where("action = ?", strings.TrimSpace(*f.Action))
	}
	if f.UserID != nil && strings.TrimSpace(*f.UserID) != "" {
		q = q.Where("user_id = ?", strings.TrimSpace(*f.UserID))
	}
	if f.CreatedFrom != nil {
		q = q.Where("created_at >= ?", *f.CreatedFrom)
	}
	if f.CreatedTo != nil {
		q = q.Where("created_at <= ?", *f.CreatedTo)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page := f.Page
	per := f.PerPage
	if page < 1 {
		page = 1
	}
	if per < 1 {
		per = 20
	}

	var rows []auditLogRow
	err := q.Order("created_at DESC, id ASC").
		Offset((page - 1) * per).
		Limit(per).
		Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	out := make([]Entry, 0, len(rows))
	for i := range rows {
		out = append(out, rowToEntry(rows[i]))
	}
	return out, total, nil
}

func rowToEntry(r auditLogRow) Entry {
	return Entry{
		ID:         r.ID,
		TenantID:   r.TenantID,
		UserID:     r.UserID,
		Action:     r.Action,
		Entity:     r.Entity,
		EntityID:   r.EntityID,
		BeforeData: r.BeforeData,
		AfterData:  r.AfterData,
		IPAddress:  r.IPAddress,
		UserAgent:  r.UserAgent,
		RequestID:  r.RequestID,
		CreatedAt:  r.CreatedAt,
	}
}
