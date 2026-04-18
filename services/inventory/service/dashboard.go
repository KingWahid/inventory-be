package service

import (
	"context"

	dashboarduc "github.com/KingWahid/inventory/backend/services/inventory/domains/dashboard/usecase"
)

// GetDashboardSummary returns §9 aggregate counts with server-side Redis cache-aside (30s).
func (s *InventoryService) GetDashboardSummary(ctx context.Context) (dashboarduc.Summary, error) {
	return s.dashboard.GetDashboardSummary(ctx)
}
