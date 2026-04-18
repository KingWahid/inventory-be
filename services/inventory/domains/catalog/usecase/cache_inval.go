package usecase

import (
	"context"

	cachepkg "github.com/KingWahid/inventory/backend/pkg/cache"
)

func (u *usecase) invalidateAfterCategoryWrite(ctx context.Context, tenantID string) {
	_ = u.cache.DeletePattern(ctx, cachepkg.PatternCategories(tenantID))
	_ = u.cache.DeletePattern(ctx, cachepkg.PatternProducts(tenantID))
	u.invalidateDashboard(ctx, tenantID)
}

func (u *usecase) invalidateAfterProductWrite(ctx context.Context, tenantID, productID string) {
	_ = u.cache.Delete(ctx, cachepkg.KeyProduct(tenantID, productID))
	_ = u.cache.DeletePattern(ctx, cachepkg.PatternProducts(tenantID))
	u.invalidateDashboard(ctx, tenantID)
}

func (u *usecase) invalidateDashboard(ctx context.Context, tenantID string) {
	_ = u.cache.Delete(ctx, cachepkg.KeyDashboardSummary(tenantID))
}
