package usecase

import (
	"context"
	"encoding/json"

	movrepo "github.com/KingWahid/inventory/backend/services/inventory/domains/movement/repository"
	auditrepo "github.com/KingWahid/inventory/backend/services/inventory/domains/audit/repository"
	outboxrepo "github.com/KingWahid/inventory/backend/services/inventory/domains/outbox/repository"
)

func (u *usecase) emitAudit(ctx context.Context, tenantID, userID, movementID string, mv movrepo.Movement) error {
	after, err := json.Marshal(map[string]any{
		"movement_id":      movementID,
		"status":           movrepo.StatusConfirmed,
		"type":             mv.Type,
		"reference_number": mv.ReferenceNumber,
	})
	if err != nil {
		return err
	}
	return u.audit.Insert(ctx, auditrepo.InsertInput{
		TenantID:   tenantID,
		UserID:     &userID,
		Action:     "movement.confirm",
		Entity:     "movement",
		EntityID:   movementID,
		AfterData:  after,
		BeforeData: nil,
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
		EventType:     "MovementCreated",
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
			EventType:     "StockChanged",
			AggregateType: "movement",
			AggregateID:   movementID,
			Payload:       pb,
		}); err != nil {
			return err
		}
	}
	return nil
}
