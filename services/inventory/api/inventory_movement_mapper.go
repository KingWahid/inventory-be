package api

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/KingWahid/inventory/backend/services/inventory/domains/movement/repository"
	"github.com/KingWahid/inventory/backend/services/inventory/stub"
)

func movementRepoToStub(m repository.Movement) (stub.Movement, error) {
	id, err := uuid.Parse(m.ID)
	if err != nil {
		return stub.Movement{}, fmt.Errorf("movement id: %w", err)
	}
	tid, err := uuid.Parse(m.TenantID)
	if err != nil {
		return stub.Movement{}, fmt.Errorf("tenant id: %w", err)
	}
	creator, err := uuid.Parse(m.CreatedBy)
	if err != nil {
		return stub.Movement{}, fmt.Errorf("created_by: %w", err)
	}

	out := stub.Movement{
		Id:              openapi_types.UUID(id),
		TenantId:        openapi_types.UUID(tid),
		Type:            stub.MovementType(m.Type),
		ReferenceNumber: m.ReferenceNumber,
		CreatedBy:       openapi_types.UUID(creator),
		Status:          stub.MovementStatus(m.Status),
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
		Notes:           m.Notes,
		IdempotencyKey:  m.IdempotencyKey,
	}
	out.SourceWarehouseId = uuidPtrFromStringOptional(m.SourceWarehouseID)
	out.DestinationWarehouseId = uuidPtrFromStringOptional(m.DestinationWarehouseID)

	if len(m.Lines) > 0 {
		ls := make([]stub.MovementLine, 0, len(m.Lines))
		for i := range m.Lines {
			row, err := movementLineRepoToStub(m.Lines[i])
			if err != nil {
				return stub.Movement{}, err
			}
			ls = append(ls, row)
		}
		out.Lines = &ls
	}
	return out, nil
}

func movementLineRepoToStub(l repository.MovementLine) (stub.MovementLine, error) {
	id, err := uuid.Parse(l.ID)
	if err != nil {
		return stub.MovementLine{}, err
	}
	mid, err := uuid.Parse(l.MovementID)
	if err != nil {
		return stub.MovementLine{}, err
	}
	pid, err := uuid.Parse(l.ProductID)
	if err != nil {
		return stub.MovementLine{}, err
	}
	return stub.MovementLine{
		Id:         openapi_types.UUID(id),
		MovementId: openapi_types.UUID(mid),
		ProductId:  openapi_types.UUID(pid),
		Quantity:   l.Quantity,
		Notes:      l.Notes,
		CreatedAt:  l.CreatedAt,
	}, nil
}

func uuidPtrFromStringOptional(s *string) *openapi_types.UUID {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	u, err := uuid.Parse(t)
	if err != nil {
		return nil
	}
	x := openapi_types.UUID(u)
	return &x
}
