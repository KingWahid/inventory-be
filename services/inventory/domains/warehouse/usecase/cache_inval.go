package usecase

import (
	"context"

	cachepkg "github.com/KingWahid/inventory/backend/pkg/cache"
)

func (u *usecase) invalidateAfterWarehouseWrite(ctx context.Context, tenantID string) {
	_ = u.cache.DeletePattern(ctx, cachepkg.PatternWarehouses(tenantID))
	_ = u.cache.Delete(ctx, cachepkg.KeyDashboardSummary(tenantID))
}
