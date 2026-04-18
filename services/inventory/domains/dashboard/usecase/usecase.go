package usecase

import (
	"context"
	"encoding/json"

	cachepkg "github.com/KingWahid/inventory/backend/pkg/cache"
	commonjwt "github.com/KingWahid/inventory/backend/pkg/common/jwt"
	dashrepo "github.com/KingWahid/inventory/backend/services/inventory/domains/dashboard/repository"
)

// Summary is the application-layer dashboard DTO (same shape as repository).
type Summary = dashrepo.Summary

// Usecase returns cached dashboard aggregates (TTL §13).
type Usecase interface {
	GetDashboardSummary(ctx context.Context) (Summary, error)
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
