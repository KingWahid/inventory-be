package jwt

import (
	"context"
	"errors"
	"testing"

	"github.com/KingWahid/inventory/backend/pkg/common/errorcodes"
)

func TestTenantIDFromContext_MissingClaims(t *testing.T) {
	_, err := TenantIDFromContext(context.Background())
	if !errors.Is(err, errorcodes.ErrTenantContextMissing) {
		t.Fatalf("got %v", err)
	}
	st, ae := errorcodes.ToHTTP(err)
	if st != 401 || ae.Code != errorcodes.CodeUnauthorized {
		t.Fatalf("want 401 UNAUTHORIZED got %d %+v", st, ae)
	}
}

func TestTenantIDFromContext_OK(t *testing.T) {
	ctx := ContextWithClaims(context.Background(), &Claims{
		TenantID: "tenant-a",
	})
	tid, err := TenantIDFromContext(ctx)
	if err != nil || tid != "tenant-a" {
		t.Fatalf("got %q %v", tid, err)
	}
}

func TestPermissionsForRole(t *testing.T) {
	all := PermissionsForRole("admin")
	if len(all) < 5 {
		t.Fatalf("admin should have full set, got %v", all)
	}
	staff := PermissionsForRole("staff")
	var hasConfirm bool
	for _, p := range staff {
		if p == PermMovementConfirm {
			hasConfirm = true
		}
	}
	if hasConfirm {
		t.Fatalf("staff should not include movement:confirm, got %v", staff)
	}
}

func TestHasPermission_RoleShortcut(t *testing.T) {
	if !HasPermission(&Claims{Role: "owner"}, PermMovementConfirm) {
		t.Fatal("owner expects full access")
	}
	if HasPermission(&Claims{Role: "staff", Permissions: PermissionsForRole("staff")}, PermMovementConfirm) {
		t.Fatal("staff must not confirm without grant")
	}
}

func TestRequirePermission(t *testing.T) {
	ctx := ContextWithClaims(context.Background(), &Claims{
		TenantID:    "t1",
		Role:        "staff",
		Permissions: PermissionsForRole("staff"),
	})
	err := RequirePermission(ctx, PermMovementConfirm)
	var forbidden errorcodes.AppError
	if !errors.As(err, &forbidden) || forbidden.Code != errorcodes.CodeForbidden {
		t.Fatalf("want FORBIDDEN got %v", err)
	}
	ctx2 := ContextWithClaims(context.Background(), &Claims{
		TenantID:    "t1",
		Role:        "manager",
		Permissions: PermissionsForRole("manager"),
	})
	if err := RequirePermission(ctx2, PermMovementConfirm); err != nil {
		t.Fatal(err)
	}
}
