---
paths:
  - "infra/database/migrations/**"
  - "pkg/database/repositories/roles/**"
  - "pkg/services/access/**"
---

# Built-in Roles

The system has two built-in roles that are seeded via migrations. These roles have `type = 'built-in'` and `organization_id = NULL` (they are not scoped to any organization).

## User Role

**Migration:** `000135_seed_universal_user_role.up.sql`
**Type migration:** `000157_add_role_type_enum.up.sql` (converted `is_universal` to `type = 'built-in'`)

| Property | Value | Purpose |
|----------|-------|---------|
| `name` | `User` | Display name |
| `type` | `built-in` | Cannot be created/deleted by org admins |
| `organization_id` | `NULL` | Applies across all organizations |
| `assigned_by_default` | `true` | Auto-assigned to every new user |
| `is_mutable` | `false` | Cannot be modified or deleted |
| `is_assignable` | `false` | Hidden from manual role-picker UI; auto-assigned via `assigned_by_default` |

### Purpose

The User role holds **universal permissions** — permissions every user needs regardless of their organization or custom role assignments. Examples:
- Notification permissions (list, show, delete, count, mark read)
- Platform access checks (check admin/my/app access)

### When to Assign Permissions to this Role

Assign a permission to the User role when:
- **Every user** needs the permission, not just specific roles
- The permission is **not configurable** — org admins should not be able to revoke it
- The permission should be **hidden** from the role management UI (`hide = true`)

Use the CROSS JOIN pattern documented in `.claude/rules/database/permissions.md` Step 6.

### Permission Check Flow

When `CheckEndpointPermission` runs, it JOINs through:
```
user_roles → roles → role_permissions → permissions
```
The WHERE clause includes `r.type = 'built-in'` alongside `r.organization_id = ?`, so built-in role permissions are always checked for every user.

## Organization Owner Role

**Migration:** `000155_seed_organization_owner_role.up.sql`

| Property | Value | Purpose |
|----------|-------|---------|
| `name` | `Organization Owner` | Display name |
| `type` | `built-in` | Cannot be created/deleted by org admins |
| `organization_id` | `NULL` | Applies across all organizations |
| `grants_all_org_permissions` | `true` | Bypasses `role_permissions` lookup |
| `is_mutable` | `false` | Cannot be modified or deleted |
| `is_assignable` | `false` | Hidden from manual role-picker UI; granted only through the org-ownership flow (`POST /admin/organizations`) |

### Purpose

The Organization Owner role grants **all permissions** available to an organization. It does NOT use `role_permissions` — instead, the permission check logic has a special bypass:

1. `UserAllowedToAccessEndpoint` checks `UserHasGrantsAllOrgPermissionsRole`
2. If true AND admin org → **full bypass** (superadmin)
3. If true AND non-admin org → only checks `CheckEndpointEnabledForOrg` (feature toggle gate)
4. Regular users → full `CheckEndpointPermission` + `CheckEndpointEnabledForOrg`

### When to Use

You do NOT assign permissions to this role. Its `grants_all_org_permissions = true` flag means it automatically has access to everything the organization's feature set allows. No `role_permissions` entries are needed.

## Role Type Constants

Defined in `pkg/database/constants/roles.go`:

```go
const (
    RoleTypeBuiltIn      RoleType = "built-in"
    RoleTypeOrganization RoleType = "organization"
    RoleTypeLicense      RoleType = "license"
)
```

- `built-in` — System roles (User, Organization Owner). Cannot be created or deleted by users.
- `organization` — Custom roles created by org admins. Scoped to a specific organization.
- `license` — Auto-generated shadow roles created by the billing service for per-user-priced subscriptions. Scoped to a specific organization. Created and hard-deleted by the billing service only — org admins cannot manage them. Always created with `is_assignable = false` so they are excluded from the manual role-picker UI; licenses are granted only through the subscription/billing flow. See `pkg/services/billing/shadow_role.go` for lifecycle management.

## Role Assignability

The `is_assignable` boolean column (migration `000377_add_is_assignable_to_roles.up.sql`) controls whether a role can be assigned to a user through the normal role-assignment endpoints. Defaults to `true`. Built-in User + Organization Owner and all `license`-type roles are seeded/created with `is_assignable = false`.

### Listing-side filter

Three listing endpoints accept an optional `assignable` query parameter:
- `GET /common/roles`
- `GET /common/admin/organizations/{organizationID}/roles`
- `GET /common/roles/built-in`

Values: `true` returns only assignable rows, `false` returns only non-assignable, omitted returns everything. The cache key encodes the filter so the three buckets do not poison each other.

### Role Validation Methods (`pkg/database/repositories/roles/`)

| Method | Purpose |
|--------|---------|
| `CheckRolesExist` | Verifies every role ID is visible to the caller's org (`organization_id = ? OR type = 'built-in'`) and is not soft-deleted. Returns existence, not assignability. |
| `CheckRolesAssignable` | Returns the subset of role IDs whose `is_assignable = false`, scoped the same way as `CheckRolesExist`. Used before any write to `common.user_roles`. |

### Enforcing `is_assignable` at the Service Layer

Any service method that accepts user-supplied role IDs **must** validate assignability on the raw request slice **before** any default-role merging, existence check, or write. Two helpers live in `pkg/services/identity/role_assignability.go`:

- `validateRoleAssignability(ctx, roleRepo, organizationID, roleIDs)` — **strict**. A request containing at least one non-assignable ID is rejected with 400 `error_role_not_assignable` and the list of offending IDs. Use for **create** flows, where the target user has no prior roles in the org.
- `validateRoleAssignabilityForUpdate(ctx, roleRepo, organizationID, incomingRoleIDs, currentRoleIDs)` — **relaxed**. Role IDs already present in `currentRoleIDs` are filtered out before the strict check runs. This makes re-sending an existing non-assignable role (e.g., the built-in `User` role that every user auto-receives, or `Organization Owner` for an existing owner) a no-op instead of a policy violation. Use for **update** flows that replace a user's role set. The caller must fetch the user's current roles for the target org *before* any default-role merging and pass them in.

Currently wired into:

- **Create (strict)** — `identity.CreateUser`, `identity.InviteAdminUser`
- **Update (relaxed)** — `identity.UpdateUser`, `identity.UpdateUserOrganizationAssignments`

Exempt: `organization.CreateOrganization` (`POST /admin/organizations`) grants Organization Owner via the internal `assignOrganizationOwnerRole` helper, not via user-supplied `role_ids`, so no check is needed.

### Repository-level protection

`UpdateUserRoles` in `pkg/database/repositories/user_roles/repo.go` enforces the same invariant independently as a second line of defense. Its DELETE subquery targets only rows where `is_assignable = true`:

```sql
DELETE FROM common.user_roles
 WHERE user_id = ? AND organization_id = ?
   AND role_id IN (
     SELECT id FROM common.roles
     WHERE is_assignable = true AND deleted_at IS NULL
   )
```

This means non-assignable roles — User, Organization Owner, and all `type='license'` shadow roles — are never removed by a user-facing PUT `.../roles` call regardless of what the service layer passes. The service-layer `validateRoleAssignability` / `validateRoleAssignabilityForUpdate` helpers remain the primary gate (they also guard the INSERT path by rejecting non-assignable IDs up front, with the relaxed update variant exempting already-held IDs), but the repository enforces the non-removal invariant on its own.

Do NOT use `UpdateUserRoles` to clear all of a user's roles in an org (that would leave the non-assignable ones behind). For the full "remove user from org" flow, use `DeleteUserRolesInOrganization` — which intentionally bypasses `is_assignable` because it represents full org membership removal, not a role-picker update.
