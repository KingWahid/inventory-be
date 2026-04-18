package usecase

import (
	"context"
	"encoding/json"

	"github.com/KingWahid/inventory/backend/services/inventory/domains/audit/logwriter"
	movrepo "github.com/KingWahid/inventory/backend/services/inventory/domains/movement/repository"
	outboxrepo "github.com/KingWahid/inventory/backend/services/inventory/domains/outbox/repository"
)

func (u *usecase) emitAudit(ctx context.Context, movementID string, mv movrepo.Movement) error {
	if u.auditLog == nil {
		return nil
	}
	before := map[string]any{
		"movement_id":      movementID,
		"status":           mv.Status,
		"type":             mv.Type,
		"reference_number": mv.ReferenceNumber,
	}
	after := map[string]any{
		"movement_id":      movementID,
		"status":           movrepo.StatusConfirmed,
		"type":             mv.Type,
		"reference_number": mv.ReferenceNumber,
		"line_count":       len(mv.Lines),
	}
	return u.auditLog.Log(ctx, logwriter.Params{
		Action:   "movement.confirm",
		Entity:   "movement",
		EntityID: movementID,
		Before:   before,
		After:    after,
	})
}

type movementCreatedPayload struct {
	TenantID        string `json:"tenant_id"`
	MovementID      string `json:"movement_id"`
	Type            string `json:"type"`
	ReferenceNumber string `json:"reference_number"`
	LineCount       int    `json:"line_count"`
}

type stockChangedPayload struct {
	TenantID    string `json:"tenant_id"`
	WarehouseID string `json:"warehouse_id"`
	ProductID   string `json:"product_id"`
	OldQty      int32  `json:"old_qty"`
	NewQty      int32  `json:"new_qty"`
	MovementID  string `json:"movement_id"`
}

type stockBelowThresholdPayload struct {
	TenantID     string `json:"tenant_id"`
	WarehouseID  string `json:"warehouse_id"`
	ProductID    string `json:"product_id"`
	CurrentQty   int32  `json:"current_qty"`
	ReorderLevel int32  `json:"reorder_level"`
}

// emitOutbox inserts outbox_events (published=false via repository) using the caller's tx context.
// Event types and JSON fields follow ARCHITECTURE.md §10.
func (u *usecase) emitOutbox(ctx context.Context, tenantID, movementID string, mv movrepo.Movement, changes []stockQtyChange) error {
	mc := movementCreatedPayload{
		TenantID:        tenantID,
		MovementID:      movementID,
		Type:            mv.Type,
		ReferenceNumber: mv.ReferenceNumber,
		LineCount:       len(mv.Lines),
	}
	mb, err := json.Marshal(mc)
	if err != nil {
		return err
	}
	if err := u.outbox.Insert(ctx, outboxrepo.InsertInput{
		TenantID:      tenantID,
		EventType:     EventTypeMovementCreated,
		AggregateType: "movement",
		AggregateID:   movementID,
		Payload:       mb,
	}); err != nil {
		return err
	}

	for _, ch := range changes {
		p := stockChangedPayload{
			TenantID:    tenantID,
			WarehouseID: ch.WarehouseID,
			ProductID:   ch.ProductID,
			OldQty:      ch.OldQty,
			NewQty:      ch.NewQty,
			MovementID:  movementID,
		}
		pb, err := json.Marshal(p)
		if err != nil {
			return err
		}
		if err := u.outbox.Insert(ctx, outboxrepo.InsertInput{
			TenantID:      tenantID,
			EventType:     EventTypeStockChanged,
			AggregateType: "movement",
			AggregateID:   movementID,
			Payload:       pb,
		}); err != nil {
			return err
		}

		if u.catalog != nil {
			prod, err := u.catalog.GetProduct(ctx, ch.ProductID)
			if err != nil {
				return err
			}
			rl := prod.ReorderLevel
			if rl > 0 && ch.NewQty < rl {
				sb := stockBelowThresholdPayload{
					TenantID:     tenantID,
					WarehouseID:  ch.WarehouseID,
					ProductID:    ch.ProductID,
					CurrentQty:   ch.NewQty,
					ReorderLevel: rl,
				}
				sbb, err := json.Marshal(sb)
				if err != nil {
					return err
				}
				if err := u.outbox.Insert(ctx, outboxrepo.InsertInput{
					TenantID:      tenantID,
					EventType:     EventTypeStockBelowThreshold,
					AggregateType: "movement",
					AggregateID:   movementID,
					Payload:       sbb,
				}); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
