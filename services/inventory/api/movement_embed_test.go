package api

import (
	"context"

	"github.com/KingWahid/inventory/backend/pkg/common/errorcodes"
	audituc "github.com/KingWahid/inventory/backend/services/inventory/domains/audit/usecase"
	movrepo "github.com/KingWahid/inventory/backend/services/inventory/domains/movement/repository"
	movementuc "github.com/KingWahid/inventory/backend/services/inventory/domains/movement/usecase"
)

// movementEmbedNoop implements movement + default audit methods for handler test doubles (embedded in errOrListSvc, productDelSvc, warehouseDelSvc).
type movementEmbedNoop struct{}

func (movementEmbedNoop) CreateInbound(context.Context, string, movementuc.CreateMovementBase) (movrepo.Movement, error) {
	return movrepo.Movement{}, errorcodes.ErrNotFound
}

func (movementEmbedNoop) CreateOutbound(context.Context, string, movementuc.CreateMovementBase) (movrepo.Movement, error) {
	return movrepo.Movement{}, errorcodes.ErrNotFound
}

func (movementEmbedNoop) CreateTransfer(context.Context, string, string, movementuc.CreateMovementBase) (movrepo.Movement, error) {
	return movrepo.Movement{}, errorcodes.ErrNotFound
}

func (movementEmbedNoop) CreateAdjustment(context.Context, *string, *string, movementuc.CreateMovementBase) (movrepo.Movement, error) {
	return movrepo.Movement{}, errorcodes.ErrNotFound
}

func (movementEmbedNoop) GetMovement(context.Context, string) (movrepo.Movement, error) {
	return movrepo.Movement{}, errorcodes.ErrNotFound
}

func (movementEmbedNoop) ListMovements(context.Context, movementuc.ListMovementsInput) (movementuc.ListMovementsOutput, error) {
	return movementuc.ListMovementsOutput{}, nil
}

func (movementEmbedNoop) ConfirmMovement(context.Context, string) (movrepo.Movement, error) {
	return movrepo.Movement{}, errorcodes.ErrNotFound
}

func (movementEmbedNoop) CancelMovement(context.Context, string) (movrepo.Movement, error) {
	return movrepo.Movement{}, errorcodes.ErrNotFound
}

func (movementEmbedNoop) ListAuditLogs(context.Context, audituc.ListAuditLogsInput) (audituc.ListAuditLogsOutput, error) {
	return audituc.ListAuditLogsOutput{}, nil
}
