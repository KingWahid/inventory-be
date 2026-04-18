package usecase

import (
	"context"
	"errors"
	"testing"

	cachepkg "github.com/KingWahid/inventory/backend/pkg/cache"
	"github.com/KingWahid/inventory/backend/pkg/common/errorcodes"
	commonjwt "github.com/KingWahid/inventory/backend/pkg/common/jwt"
	movrepo "github.com/KingWahid/inventory/backend/services/inventory/domains/movement/repository"
	"github.com/golang-jwt/jwt/v4"
)

type stubMovRepo struct {
	first  movrepo.Movement
	second movrepo.Movement
	calls  int
}

func (s *stubMovRepo) Ping() error { return nil }

func (s *stubMovRepo) UserBelongsToTenant(context.Context, string, string) (bool, error) {
	return true, nil
}

func (s *stubMovRepo) Create(context.Context, movrepo.CreateMovementInput) (movrepo.Movement, bool, error) {
	return movrepo.Movement{}, false, errorcodes.ErrInternal
}

func (s *stubMovRepo) GetByTenantAndIdempotencyKey(context.Context, string, string) (movrepo.Movement, error) {
	return movrepo.Movement{}, errorcodes.ErrNotFound
}

func (s *stubMovRepo) GetByID(context.Context, string, string) (movrepo.Movement, error) {
	s.calls++
	if s.calls == 1 {
		return s.first, nil
	}
	return s.second, nil
}

func (s *stubMovRepo) GetByIDForUpdate(context.Context, string, string) (movrepo.Movement, error) {
	return movrepo.Movement{}, errorcodes.ErrInternal
}

func (s *stubMovRepo) List(context.Context, string, movrepo.ListFilter) ([]movrepo.Movement, int64, error) {
	return nil, 0, nil
}

func (s *stubMovRepo) UpdateStatus(context.Context, string, string, string, string) error {
	return nil
}

func testClaimsCtx(tenantID, userID string) context.Context {
	cl := &commonjwt.Claims{
		TenantID: tenantID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject: userID,
		},
	}
	return commonjwt.ContextWithClaims(context.Background(), cl)
}

func TestCancelMovement_confirmedFails(t *testing.T) {
	t.Parallel()
	ctx := testClaimsCtx(
		"cccccccc-cccc-cccc-cccc-cccccccccccc",
		"dddddddd-dddd-dddd-dddd-dddddddddddd",
	)
	u := New(
		&stubMovRepo{
			first: movrepo.Movement{Status: movrepo.StatusConfirmed},
		},
		nil, nil, nil, nil, nil, nil, cachepkg.Noop{},
	)
	_, err := u.CancelMovement(ctx, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	if !errors.Is(err, errorcodes.ErrMovementDraft) {
		t.Fatalf("want ErrMovementDraft, got %v", err)
	}
}

func TestCancelMovement_draftOK(t *testing.T) {
	t.Parallel()
	ctx := testClaimsCtx(
		"cccccccc-cccc-cccc-cccc-cccccccccccc",
		"dddddddd-dddd-dddd-dddd-dddddddddddd",
	)
	mid := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	u := New(
		&stubMovRepo{
			first:  movrepo.Movement{ID: mid, Status: movrepo.StatusDraft},
			second: movrepo.Movement{ID: mid, Status: movrepo.StatusCancelled},
		},
		nil, nil, nil, nil, nil, nil, cachepkg.Noop{},
	)
	m, err := u.CancelMovement(ctx, mid)
	if err != nil {
		t.Fatal(err)
	}
	if m.Status != movrepo.StatusCancelled {
		t.Fatalf("status want cancelled got %s", m.Status)
	}
}
