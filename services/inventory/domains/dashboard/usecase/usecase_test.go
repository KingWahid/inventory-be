package usecase

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	cachepkg "github.com/KingWahid/inventory/backend/pkg/cache"
	commonjwt "github.com/KingWahid/inventory/backend/pkg/common/jwt"
	dashrepo "github.com/KingWahid/inventory/backend/services/inventory/domains/dashboard/repository"
)

type spyDashRepo struct {
	calls int
	sum   dashrepo.Summary
	err   error

	chartCalls int
	chartPts   []dashrepo.MovementChartPoint
	chartErr   error
}

func (s *spyDashRepo) GetDashboardSummary(context.Context, string) (dashrepo.Summary, error) {
	s.calls++
	return s.sum, s.err
}

func (s *spyDashRepo) GetMovementChart(context.Context, string, dashrepo.MovementChartPeriod) ([]dashrepo.MovementChartPoint, error) {
	s.chartCalls++
	return s.chartPts, s.chartErr
}

func TestGetDashboardSummary_secondCallUsesCache(t *testing.T) {
	t.Parallel()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()
	c := cachepkg.NewRedis(rdb)

	repo := &spyDashRepo{
		sum: dashrepo.Summary{
			TotalProducts:   3,
			TotalWarehouses: 2,
			MovementsToday:  1,
			LowStockCount:   0,
		},
	}
	u := New(repo, c)
	ctx := commonjwt.ContextWithClaims(context.Background(), &commonjwt.Claims{TenantID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"})

	s1, err := u.GetDashboardSummary(ctx)
	if err != nil {
		t.Fatal(err)
	}
	s2, err := u.GetDashboardSummary(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if s1.TotalProducts != s2.TotalProducts || repo.calls != 1 {
		t.Fatalf("repo.calls=%d sum1=%+v sum2=%+v", repo.calls, s1, s2)
	}
}

func TestGetDashboardMovementsChart_secondCallUsesCache(t *testing.T) {
	t.Parallel()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()
	c := cachepkg.NewRedis(rdb)

	repo := &spyDashRepo{
		chartPts: []dashrepo.MovementChartPoint{{BucketStart: "2026-04-01", MovementCount: 7}},
	}
	u := New(repo, c)
	ctx := commonjwt.ContextWithClaims(context.Background(), &commonjwt.Claims{TenantID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"})

	mc1, err := u.GetDashboardMovementsChart(ctx, "daily")
	if err != nil {
		t.Fatal(err)
	}
	mc2, err := u.GetDashboardMovementsChart(ctx, "daily")
	if err != nil {
		t.Fatal(err)
	}
	if mc1.Period != "daily" || mc2.Period != "daily" || len(mc1.Points) != len(mc2.Points) || repo.chartCalls != 1 {
		t.Fatalf("repo.chartCalls=%d mc1=%+v mc2=%+v", repo.chartCalls, mc1, mc2)
	}
}
