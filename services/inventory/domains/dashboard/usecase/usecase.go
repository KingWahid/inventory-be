package usecase

import (
	"context"
	"encoding/json"

	cachepkg "github.com/KingWahid/inventory/backend/pkg/cache"
	"github.com/KingWahid/inventory/backend/pkg/common/errorcodes"
	commonjwt "github.com/KingWahid/inventory/backend/pkg/common/jwt"
	dashrepo "github.com/KingWahid/inventory/backend/services/inventory/domains/dashboard/repository"
)

// Summary is the application-layer dashboard DTO (same shape as repository).
type Summary = dashrepo.Summary

// MovementChart is cached dashboard movement trend series (§9).
type MovementChart struct {
	Period string                        `json:"period"`
	Points []dashrepo.MovementChartPoint `json:"points"`
}

type StorageUtilizationRow struct {
	WarehouseID   string `json:"warehouse_id"`
	WarehouseCode string `json:"warehouse_code"`
	WarehouseName string `json:"warehouse_name"`
	OnHandQty     int64  `json:"on_hand_qty"`
	Percent       int32  `json:"percent"`
}

// Usecase returns cached dashboard aggregates (TTL §13).
type Usecase interface {
	GetDashboardSummary(ctx context.Context) (Summary, error)
	GetDashboardMovementsChart(ctx context.Context, periodParam string) (MovementChart, error)
	GetStorageUtilization(ctx context.Context, limit int) ([]StorageUtilizationRow, error)
}

type usecase struct {
	repo  dashrepo.Repository
	cache cachepkg.Cache
}

// New constructs dashboard usecase.
func New(repo dashrepo.Repository, c cachepkg.Cache) Usecase {
	if c == nil {
		c = cachepkg.Noop{}
	}
	return &usecase{repo: repo, cache: c}
}

func (u *usecase) GetDashboardSummary(ctx context.Context) (Summary, error) {
	tid, err := commonjwt.TenantIDFromContext(ctx)
	if err != nil {
		return Summary{}, err
	}
	key := cachepkg.KeyDashboardSummary(tid)
	if raw, hit, err := u.cache.Get(ctx, key); err == nil && hit {
		var s Summary
		if err := json.Unmarshal(raw, &s); err == nil {
			return s, nil
		}
	}
	s, err := u.repo.GetDashboardSummary(ctx, tid)
	if err != nil {
		return Summary{}, err
	}
	if b, err := json.Marshal(s); err == nil {
		_ = u.cache.Set(ctx, key, b, cachepkg.TTLDashboardSummary)
	}
	return s, nil
}

func (u *usecase) GetDashboardMovementsChart(ctx context.Context, periodParam string) (MovementChart, error) {
	period, err := dashrepo.NormalizeChartPeriod(periodParam)
	if err != nil {
		return MovementChart{}, errorcodes.ErrValidationError.WithDetails(map[string]any{"message": "period must be daily, weekly, or monthly"})
	}
	tid, err := commonjwt.TenantIDFromContext(ctx)
	if err != nil {
		return MovementChart{}, err
	}
	fp := cachepkg.ChartPeriodFingerprint(string(period))
	key := cachepkg.KeyDashboardMovementsChart(tid, fp)
	if raw, hit, err := u.cache.Get(ctx, key); err == nil && hit {
		var mc MovementChart
		if err := json.Unmarshal(raw, &mc); err == nil {
			return mc, nil
		}
	}
	points, err := u.repo.GetMovementChart(ctx, tid, period)
	if err != nil {
		return MovementChart{}, err
	}
	out := MovementChart{Period: string(period), Points: points}
	if b, err := json.Marshal(out); err == nil {
		_ = u.cache.Set(ctx, key, b, cachepkg.TTLDashboardChart)
	}
	return out, nil
}

func (u *usecase) GetStorageUtilization(ctx context.Context, limit int) ([]StorageUtilizationRow, error) {
	tid, err := commonjwt.TenantIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := u.repo.GetStorageUtilization(ctx, tid, limit)
	if err != nil {
		return nil, err
	}
	out := make([]StorageUtilizationRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, StorageUtilizationRow{
			WarehouseID:   r.WarehouseID,
			WarehouseCode: r.WarehouseCode,
			WarehouseName: r.WarehouseName,
			OnHandQty:     r.OnHandQty,
			Percent:       r.Percent,
		})
	}
	return out, nil
}
