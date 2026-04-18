package repository

import (
	"context"
	"crypto/subtle"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/KingWahid/inventory/backend/pkg/common/errorcodes"
	"github.com/KingWahid/inventory/backend/pkg/database/transaction"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Repository defines movement persistence.
type Repository interface {
	Ping() error
	UserBelongsToTenant(ctx context.Context, tenantID, userID string) (bool, error)
	Create(ctx context.Context, in CreateMovementInput) (Movement, error)
	GetByTenantAndIdempotencyKey(ctx context.Context, tenantID, idempotencyKey string) (Movement, error)
	GetByID(ctx context.Context, tenantID, movementID string) (Movement, error)
	GetByIDForUpdate(ctx context.Context, tenantID, movementID string) (Movement, error)
	List(ctx context.Context, tenantID string, f ListFilter) ([]Movement, int64, error)
	UpdateStatus(ctx context.Context, tenantID, movementID, fromStatus, toStatus string) error
}

// CreateMovementInput inserts a draft movement and its lines (same DB session / tx via ctx).
type CreateMovementInput struct {
	TenantID               string
	Type                   string
	ReferenceNumber        string
	SourceWarehouseID      *string
	DestinationWarehouseID *string
	CreatedBy              string
	Notes                  *string
	IdempotencyKey           string // normalized non-empty when using §9 header flow
	IdempotencyRequestHash   string // 64-char lowercase hex SHA-256 of raw JSON body
	Lines                    []MovementLineInput
}

// MovementLineInput is one line for Create.
type MovementLineInput struct {
	ProductID string
	Quantity  int32
	Notes     *string
}

// ListFilter lists movements (optional filters).
type ListFilter struct {
	Page    int
	PerPage int
	Type    *string // movement_type
	Status  *string // movement_status
	Search  string  // reference_number ILIKE
	Sort    string
	Order   string
}

type repository struct {
	db *gorm.DB
}

// New creates movement repository implementation.
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

func (r *repository) UserBelongsToTenant(ctx context.Context, tenantID, userID string) (bool, error) {
	db := transaction.GetDB(ctx, r.db).WithContext(ctx)
	var n int64
	err := db.Raw(`
		SELECT COUNT(*) FROM users
		WHERE id = ?::uuid AND tenant_id = ?::uuid`,
		strings.TrimSpace(userID), strings.TrimSpace(tenantID),
	).Scan(&n).Error
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *repository) Create(ctx context.Context, in CreateMovementInput) (Movement, error) {
	db := transaction.GetDB(ctx, r.db).WithContext(ctx)
	mid := uuid.New().String()
	now := time.Now().UTC()

	keyTrim := strings.TrimSpace(in.IdempotencyKey)
	hashTrim := strings.TrimSpace(strings.ToLower(in.IdempotencyRequestHash))
	idemPtr := trimNonEmpty(keyTrim)
	var hashPtr *string
	if hashTrim != "" {
		hashPtr = &hashTrim
	}

	m := movementRow{
		ID:                       mid,
		TenantID:                 strings.TrimSpace(in.TenantID),
		Type:                     in.Type,
		ReferenceNumber:          strings.TrimSpace(in.ReferenceNumber),
		SourceWarehouseID:      trimStringPtr(in.SourceWarehouseID),
		DestinationWarehouseID:   trimStringPtr(in.DestinationWarehouseID),
		CreatedBy:                strings.TrimSpace(in.CreatedBy),
		Status:                   StatusDraft,
		Notes:                    trimStringPtr(in.Notes),
		IdempotencyKey:           idemPtr,
		IdempotencyRequestHash:    hashPtr,
		CreatedAt:                now,
		UpdatedAt:                now,
	}
	if err := db.Create(&m).Error; err != nil {
		if !isDuplicateErr(err) {
			return Movement{}, err
		}
		return r.handleCreateDuplicate(ctx, in, keyTrim, hashTrim)
	}
	for i := range in.Lines {
		l := in.Lines[i]
		lr := movementLineRow{
			ID:         uuid.New().String(),
			MovementID: mid,
			ProductID:  strings.TrimSpace(l.ProductID),
			Quantity:   l.Quantity,
			Notes:      trimStringPtr(l.Notes),
			CreatedAt:  now,
		}
		if err := db.Create(&lr).Error; err != nil {
			return Movement{}, err
		}
	}
	return r.GetByID(ctx, in.TenantID, mid)
}

func (r *repository) GetByTenantAndIdempotencyKey(ctx context.Context, tenantID, idempotencyKey string) (Movement, error) {
	tid := strings.TrimSpace(tenantID)
	key := strings.TrimSpace(idempotencyKey)
	if key == "" {
		return Movement{}, errorcodes.ErrNotFound
	}
	db := transaction.GetDB(ctx, r.db).WithContext(ctx)
	var row movementRow
	err := db.Where("tenant_id = ? AND idempotency_key = ?", tid, key).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Movement{}, errorcodes.ErrNotFound
		}
		return Movement{}, err
	}
	return r.GetByID(ctx, tid, row.ID)
}

func (r *repository) handleCreateDuplicate(ctx context.Context, in CreateMovementInput, keyTrim, hashTrim string) (Movement, error) {
	tid := strings.TrimSpace(in.TenantID)
	if keyTrim != "" {
		existing, err := r.GetByTenantAndIdempotencyKey(ctx, tid, keyTrim)
		if err == nil {
			if idempotencyHashesMatch(existing.IdempotencyRequestHash, hashTrim) {
				return existing, nil
			}
			return Movement{}, errorcodes.ErrIdempotency
		}
		if !errors.Is(err, errorcodes.ErrNotFound) {
			return Movement{}, err
		}
	}
	return Movement{}, errorcodes.ErrConflict.WithDetails(map[string]any{"message": "reference_number already exists"})
}

func idempotencyHashesMatch(stored *string, want string) bool {
	if len(want) != 64 {
		return false
	}
	if stored == nil {
		return false
	}
	got := strings.TrimSpace(strings.ToLower(*stored))
	if len(got) != 64 {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}

func trimNonEmpty(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

func (r *repository) GetByID(ctx context.Context, tenantID, movementID string) (Movement, error) {
	db := transaction.GetDB(ctx, r.db).WithContext(ctx)
	var m movementRow
	err := db.Where("id = ? AND tenant_id = ?", strings.TrimSpace(movementID), strings.TrimSpace(tenantID)).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Movement{}, errorcodes.ErrNotFound
		}
		return Movement{}, err
	}
	var lines []movementLineRow
	if err := db.Where("movement_id = ?", m.ID).Order("created_at ASC").Find(&lines).Error; err != nil {
		return Movement{}, err
	}
	return rowToMovement(m, lines), nil
}

func (r *repository) GetByIDForUpdate(ctx context.Context, tenantID, movementID string) (Movement, error) {
	db := transaction.GetDB(ctx, r.db).WithContext(ctx)
	var m movementRow
	err := db.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND tenant_id = ?", strings.TrimSpace(movementID), strings.TrimSpace(tenantID)).
		First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Movement{}, errorcodes.ErrNotFound
		}
		return Movement{}, err
	}
	var lines []movementLineRow
	if err := db.Where("movement_id = ?", m.ID).Order("created_at ASC").Find(&lines).Error; err != nil {
		return Movement{}, err
	}
	return rowToMovement(m, lines), nil
}

func (r *repository) List(ctx context.Context, tenantID string, f ListFilter) ([]Movement, int64, error) {
	db := transaction.GetDB(ctx, r.db).WithContext(ctx)
	tid := strings.TrimSpace(tenantID)
	q := db.Model(&movementRow{}).Where("tenant_id = ?", tid)

	if f.Type != nil && strings.TrimSpace(*f.Type) != "" {
		q = q.Where("type = ?", strings.TrimSpace(*f.Type))
	}
	if f.Status != nil && strings.TrimSpace(*f.Status) != "" {
		q = q.Where("status = ?", strings.TrimSpace(*f.Status))
	}
	if s := strings.TrimSpace(f.Search); s != "" {
		like := "%" + s + "%"
		q = q.Where("reference_number ILIKE ?", like)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	col := "created_at"
	if strings.TrimSpace(f.Sort) != "" {
		col = strings.TrimSpace(f.Sort)
	}
	ord := "DESC"
	if strings.EqualFold(strings.TrimSpace(f.Order), "asc") {
		ord = "ASC"
	}
	page := f.Page
	per := f.PerPage
	if page < 1 {
		page = 1
	}
	if per < 1 {
		per = 20
	}

	orderCol := listSortColumnSQL(col)
	var rows []movementRow
	err := q.Order(orderCol + " " + ord + ", id ASC").
		Offset((page - 1) * per).
		Limit(per).
		Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	out := make([]Movement, 0, len(rows))
	for i := range rows {
		out = append(out, rowToMovement(rows[i], nil))
	}
	return out, total, nil
}

func (r *repository) UpdateStatus(ctx context.Context, tenantID, movementID, fromStatus, toStatus string) error {
	db := transaction.GetDB(ctx, r.db).WithContext(ctx)
	now := time.Now().UTC()
	res := db.Model(&movementRow{}).
		Where("id = ? AND tenant_id = ? AND status = ?", strings.TrimSpace(movementID), strings.TrimSpace(tenantID), fromStatus).
		Updates(map[string]any{"status": toStatus, "updated_at": now})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected != 1 {
		return errorcodes.ErrMovementDraft
	}
	return nil
}

// listSortColumnSQL maps API sort field to safe quoted SQL identifier (PostgreSQL reserves `type`).
func listSortColumnSQL(name string) string {
	switch strings.TrimSpace(name) {
	case "type":
		return `"type"`
	case "status":
		return `"status"`
	default:
		return name
	}
}

func trimStringPtr(p *string) *string {
	if p == nil {
		return nil
	}
	s := strings.TrimSpace(*p)
	if s == "" {
		return nil
	}
	return &s
}

func isDuplicateErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate") || strings.Contains(msg, "unique") || strings.Contains(msg, "violates unique constraint")
}
