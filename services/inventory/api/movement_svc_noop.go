package api

import (
	"context"

	"github.com/KingWahid/inventory/backend/pkg/common/errorcodes"
	movrepo "github.com/KingWahid/inventory/backend/services/inventory/domains/movement/repository"
	movementuc "github.com/KingWahid/inventory/backend/services/inventory/domains/movement/usecase"
)

// movementSvcNoop satisfies movement-related Service methods for partial test doubles.
type movementSvcNoop struct{}

func (movementSvcNoop) CreateInbound(context.Context, string, movementuc.CreateMovementBase) (movrepo.Movement, error) {
	return movrepo.Movement{}, errorcodes.ErrNotImplemented
}

func (movementSvcNoop) CreateOutbound(context.Context, string, movementuc.CreateMovementBase) (movrepo.Movement, error) {
	return movrepo.Movement{}, errorcodes.ErrNotImplemented
}

func (movementSvcNoop) CreateTransfer(context.Context, string, string, movementuc.CreateMovementBase) (movrepo.Movement, error) {
	return movrepo.Movement{}, errorcodes.ErrNotImplemented
}

func (movementSvcNoop) CreateAdjustment(context.Context, *string, *string, movementuc.CreateMovementBase) (movrepo.Movement, error) {
	return movrepo.Movement{}, errorcodes.ErrNotImplemented
}

func (movementSvcNoop) GetMovement(context.Context, string) (movrepo.Movement, error) {
	return movrepo.Movement{}, errorcodes.ErrNotImplemented
}

func (movementSvcNoop) ListMovements(context.Context, movementuc.ListMovementsInput) (movementuc.ListMovementsOutput, error) {
	return movementuc.ListMovementsOutput{}, errorcodes.ErrNotImplemented
}

func (movementSvcNoop) ConfirmMovement(context.Context, string) (movrepo.Movement, error) {
	return movrepo.Movement{}, errorcodes.ErrNotImplemented
}

func (movementSvcNoop) CancelMovement(context.Context, string) (movrepo.Movement, error) {
	return movrepo.Movement{}, errorcodes.ErrNotImplemented
}
