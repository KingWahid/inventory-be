package service

import (
	"context"

	dashboarduc "github.com/KingWahid/inventory/backend/services/inventory/domains/dashboard/usecase"
)

// GetDashboardSummary returns §9 aggregate counts with server-side Redis cache-aside (30s).
func (s *InventoryService) GetDashboardSummary(ctx context.Context) (dashboarduc.Summary, error) {
	return s.dashboard.GetDashboardSummary(ctx)
}

// GetDashboardMovementsChart returns confirmed movement counts per UTC bucket (§9); cached ~30s (§13).
func (s *InventoryService) GetDashboardMovementsChart(ctx context.Context, period string) (dashboarduc.MovementChart, error) {
	return s.dashboard.GetDashboardMovementsChart(ctx, period)
}

// GetDashboardStorageUtilization returns on-hand based warehouse utilization snapshot.
func (s *InventoryService) GetDashboardStorageUtilization(ctx context.Context, limit int) ([]dashboarduc.StorageUtilizationRow, error) {
	return s.dashboard.GetStorageUtilization(ctx, limit)
}
