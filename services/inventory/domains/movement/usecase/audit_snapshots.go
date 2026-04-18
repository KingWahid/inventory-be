package usecase

import (
	movrepo "github.com/KingWahid/inventory/backend/services/inventory/domains/movement/repository"
)

func movementCreateAuditSnapshot(m movrepo.Movement) map[string]any {
	out := map[string]any{
		"movement_id":       m.ID,
		"type":              m.Type,
		"status":            m.Status,
		"reference_number":  m.ReferenceNumber,
		"line_count":        len(m.Lines),
	}
	if m.SourceWarehouseID != nil {
		out["source_warehouse_id"] = *m.SourceWarehouseID
	}
	if m.DestinationWarehouseID != nil {
		out["destination_warehouse_id"] = *m.DestinationWarehouseID
	}
	return out
}
