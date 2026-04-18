package repository

import "time"

// Movement types and statuses align with DDL (0007_movements.sql).
const (
	TypeInbound     = "inbound"
	TypeOutbound    = "outbound"
	TypeTransfer    = "transfer"
	TypeAdjustment  = "adjustment"
	StatusDraft     = "draft"
	StatusConfirmed = "confirmed"
	StatusCancelled = "cancelled"
)

// Movement is a tenant-scoped stock movement header with lines.
type Movement struct {
	ID                     string
	TenantID               string
	Type                   string
	ReferenceNumber        string
	SourceWarehouseID      *string
	DestinationWarehouseID *string
	CreatedBy              string
	Status                 string
	Notes                  *string
	IdempotencyKey             *string
	IdempotencyRequestHash     *string // SHA-256 hex of raw POST body when Idempotency-Key header was used
	Lines                      []MovementLine
	CreatedAt                  time.Time
	UpdatedAt                  time.Time
}

// MovementLine is one product line on a movement.
type MovementLine struct {
	ID         string
	MovementID string
	ProductID  string
	Quantity   int32
	Notes      *string
	CreatedAt  time.Time
}

type movementRow struct {
	ID                     string    `gorm:"column:id;type:uuid;primaryKey"`
	TenantID               string    `gorm:"column:tenant_id;type:uuid"`
	Type                   string    `gorm:"column:type"`
	ReferenceNumber        string    `gorm:"column:reference_number"`
	SourceWarehouseID      *string   `gorm:"column:source_warehouse_id;type:uuid"`
	DestinationWarehouseID *string   `gorm:"column:destination_warehouse_id;type:uuid"`
	CreatedBy              string    `gorm:"column:created_by;type:uuid"`
	Status                 string    `gorm:"column:status"`
	Notes                  *string   `gorm:"column:notes"`
	IdempotencyKey             *string `gorm:"column:idempotency_key"`
	IdempotencyRequestHash     *string `gorm:"column:idempotency_request_hash"`
	CreatedAt                  time.Time `gorm:"column:created_at"`
	UpdatedAt                  time.Time `gorm:"column:updated_at"`
}

func (movementRow) TableName() string { return "movements" }

type movementLineRow struct {
	ID         string    `gorm:"column:id;type:uuid;primaryKey"`
	MovementID string    `gorm:"column:movement_id;type:uuid"`
	ProductID  string    `gorm:"column:product_id;type:uuid"`
	Quantity   int32     `gorm:"column:quantity"`
	Notes      *string   `gorm:"column:notes"`
	CreatedAt  time.Time `gorm:"column:created_at"`
}

func (movementLineRow) TableName() string { return "movement_lines" }

func rowToMovement(m movementRow, lines []movementLineRow) Movement {
	out := Movement{
		ID:                     m.ID,
		TenantID:               m.TenantID,
		Type:                   m.Type,
		ReferenceNumber:        m.ReferenceNumber,
		SourceWarehouseID:      m.SourceWarehouseID,
		DestinationWarehouseID: m.DestinationWarehouseID,
		CreatedBy:              m.CreatedBy,
		Status:                 m.Status,
		Notes:                  m.Notes,
		IdempotencyKey:             m.IdempotencyKey,
		IdempotencyRequestHash:     m.IdempotencyRequestHash,
		CreatedAt:                  m.CreatedAt,
		UpdatedAt:              m.UpdatedAt,
		Lines:                  make([]MovementLine, 0, len(lines)),
	}
	for i := range lines {
		out.Lines = append(out.Lines, MovementLine{
			ID:         lines[i].ID,
			MovementID: lines[i].MovementID,
			ProductID:  lines[i].ProductID,
			Quantity:   lines[i].Quantity,
			Notes:      lines[i].Notes,
			CreatedAt:  lines[i].CreatedAt,
		})
	}
	return out
}
