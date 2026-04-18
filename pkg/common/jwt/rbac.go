package jwt

import (
	"context"
	"strings"

	"github.com/KingWahid/inventory/backend/pkg/common/errorcodes"
)

// Permission constants (ARCHITECTURE §7). Use these instead of raw strings in usecases.
const (
	PermProductRead       = "product:read"
	PermProductWrite      = "product:write"
	PermMovementWrite     = "movement:write"
	PermMovementConfirm   = "movement:confirm"
	PermReportRead        = "report:read"
)

// PermissionsForRole expands a DB role string into JWT permission claims at login time.
func PermissionsForRole(role string) []string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "super_admin":
		return append([]string(nil), allTenantPermissions()...)
	case "admin", "owner":
		return append([]string(nil), allTenantPermissions()...)
	case "manager":
		return []string{
			PermProductRead, PermProductWrite, PermMovementWrite,
			PermMovementConfirm, PermReportRead,
		}
	case "staff":
		return []string{PermProductRead, PermMovementWrite}
	default:
		return []string{PermProductRead}
	}
}

func allTenantPermissions() []string {
	return []string{
		PermProductRead, PermProductWrite, PermMovementWrite,
		PermMovementConfirm, PermReportRead,
	}
}

// HasPermission applies ARCHITECTURE §7: admin/owner/super_admin have full tenant access;
// otherwise JWT permission strings must include the requested permission.
func HasPermission(c *Claims, permission string) bool {
	if c == nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(c.Role)) {
	case "super_admin", "admin", "owner":
		return true
	}
	for _, p := range c.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// RequirePermission returns ErrTenantContextMissing if there are no claims, or ErrForbidden if the permission is denied.
func RequirePermission(ctx context.Context, permission string) error {
	claims, ok := ClaimsFromContext(ctx)
	if !ok || claims == nil {
		return errorcodes.ErrTenantContextMissing
	}
	if claims.TenantID == "" {
		return errorcodes.ErrTenantContextMissing
	}
	if !HasPermission(claims, permission) {
		return errorcodes.ErrForbidden
	}
	return nil
}
