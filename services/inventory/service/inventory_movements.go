package service

import (
	"context"

	movementuc "github.com/KingWahid/inventory/backend/services/inventory/domains/movement/usecase"
	movrepo "github.com/KingWahid/inventory/backend/services/inventory/domains/movement/repository"
)

func (s *InventoryService) CreateInbound(ctx context.Context, destinationWarehouseID string, in movementuc.CreateMovementBase) (movrepo.Movement, error) {
	return s.movement.CreateInbound(ctx, destinationWarehouseID, in)
}

func (s *InventoryService) CreateOutbound(ctx context.Context, sourceWarehouseID string, in movementuc.CreateMovementBase) (movrepo.Movement, error) {
	return s.movement.CreateOutbound(ctx, sourceWarehouseID, in)
}

func (s *InventoryService) CreateTransfer(ctx context.Context, sourceWarehouseID, destinationWarehouseID string, in movementuc.CreateMovementBase) (movrepo.Movement, error) {
	return s.movement.CreateTransfer(ctx, sourceWarehouseID, destinationWarehouseID, in)
}

func (s *InventoryService) CreateAdjustment(ctx context.Context, sourceWarehouseID, destinationWarehouseID *string, in movementuc.CreateMovementBase) (movrepo.Movement, error) {
	return s.movement.CreateAdjustment(ctx, sourceWarehouseID, destinationWarehouseID, in)
}

func (s *InventoryService) GetMovement(ctx context.Context, movementID string) (movrepo.Movement, error) {
	return s.movement.GetMovement(ctx, movementID)
}

func (s *InventoryService) ListMovements(ctx context.Context, in movementuc.ListMovementsInput) (movementuc.ListMovementsOutput, error) {
	return s.movement.ListMovements(ctx, in)
}

func (s *InventoryService) ConfirmMovement(ctx context.Context, movementID string) (movrepo.Movement, error) {
	return s.movement.ConfirmMovement(ctx, movementID)
}

func (s *InventoryService) CancelMovement(ctx context.Context, movementID string) (movrepo.Movement, error) {
	return s.movement.CancelMovement(ctx, movementID)
}
