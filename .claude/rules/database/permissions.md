---
paths:
  - "infra/database/migrations/**"
  - "services/*/api/**"
  - "infra/kong/kong.template.yml"
---

# Permission Seeding Rules

## The Rule

**Every authenticated endpoint must have a corresponding permission seed migration.** If a Kong route uses `userAuth` mode (not `publicAccess` or `deviceAuth`), a permission row must exist in `common.permissions` for that endpoint. Without it, Kong's `custom_auth_plugin` will deny access — the endpoint will return 403 for all users.

## When to Create Permission Seeds

Create a permission seed migration when:
- Adding a new authenticated endpoint (new handler + Kong route with `userAuth`)
- Changing an endpoint's path or HTTP method (update the existing permission row)
- Splitting or merging endpoints (add/remove permission rows accordingly)

Do NOT create permission seeds for:
- Public endpoints (`publicAccess` mode in Kong)
- Device-auth endpoints (`deviceAuth` mode in Kong)
- Internal endpoints not exposed through Kong

## Migration Structure

Permission seed migrations follow a **5-step structure**. Not all steps are needed every time — include only the steps that apply.

### Step 1: Insert Permissions

```sql
INSERT INTO common.permissions (endpoint_path, endpoint_action, action, resource, hide, is_admin) VALUES
    ('/billing/bills',      'GET', 'list', 'bills', false, false),
    ('/billing/bills/{id}', 'GET', 'show', 'bills', false, false)
ON CONFLICT (action, resource) DO UPDATE
SET endpoint_path = EXCLUDED.endpoint_path,
    endpoint_action = EXCLUDED.endpoint_action,
    is_admin = EXCLUDED.is_admin,
    deleted_at = NULL;
```

**Column values:**

| Column | Value | Notes |
|--------|-------|-------|
| `endpoint_path` | The Kong route path | Must match the Kong route exactly (e.g., `/ftm/fuel-tank-monitoring-devices/{id}`) |
| `endpoint_action` | HTTP method | `GET`, `POST`, `PUT`, `DELETE`, `PATCH` |
| `action` | Semantic action | `list`, `show`, `create`, `update`, `delete`, or custom (e.g., `set_is_active`, `reset_pin`) |
| `resource` | Resource name in camelCase | Matches the domain entity (e.g., `bills`, `adminPaymentProofs`, `fuelTankMonitoringDevices`) |
| `hide` | `false` (default) | Set `true` only for internal permissions that should not appear in the role permission UI |
| `is_admin` | `false` (default) | Set `true` for admin-portal-only permissions |

**Never use hardcoded UUIDs for `id`** — let `gen_random_uuid()` generate them. Use `ON CONFLICT (action, resource)` for idempotency.

### Step 2: Insert Permission Translations

Both `en` and `id` locales are required for every permission.

```sql
WITH translations (action, resource, en_name, en_desc, id_name, id_desc, res_en, res_id) AS (
    VALUES
        ('list', 'bills',
            'List Bills',        'Permission to view the list of organization bills',
            'Daftar Tagihan',    'Hak akses untuk melihat daftar tagihan organisasi',
            'Bills', 'Tagihan'),
        ('show', 'bills',
            'View Bill Detail',       'Permission to view detailed bill information',
            'Lihat Detail Tagihan',   'Hak akses untuk melihat detail informasi tagihan',
            'Bills', 'Tagihan')
),
target_perms AS (
    SELECT id, action, resource FROM common.permissions
    WHERE resource = 'bills'
      AND deleted_at IS NULL
)
INSERT INTO common.permission_translations (permission_id, locale, name, description, resource)
SELECT tp.id, 'en', t.en_name, t.en_desc, t.res_en
FROM target_perms tp
JOIN translations t ON tp.action = t.action AND tp.resource = t.resource
UNION ALL
SELECT tp.id, 'id', t.id_name, t.id_desc, t.res_id
FROM target_perms tp
JOIN translations t ON tp.action = t.action AND tp.resource = t.resource
ON CONFLICT (permission_id, locale) DO UPDATE
SET name = EXCLUDED.name,
    description = EXCLUDED.description,
    resource = EXCLUDED.resource,
    deleted_at = NULL;
```

**Translation fields:**

| Field | Example (en) | Example (id) |
|-------|-------------|-------------|
| `name` | `List Bills` | `Daftar Tagihan` |
| `description` | `Permission to view the list of ...` | `Hak akses untuk melihat daftar ...` |
| `resource` | `Bills` (display group name) | `Tagihan` |

The `resource` field in translations is a **display group name** — all permissions for the same `resource` value share the same translated group label.

### Step 3: Create Feature and Bind Permissions (if new feature)

Only needed when the permissions belong to a **new feature** that doesn't exist yet. If the feature already exists, skip to Step 4.

```sql
DO $$
DECLARE
    platform_id UUID;
    parent_feature_id UUID;
    new_feature_id UUID;
BEGIN
    -- Resolve platform
    SELECT id INTO platform_id
    FROM common.platforms
    WHERE code = '<platform_code>'  -- 'admin', 'my', or domain-specific
      AND deleted_at IS NULL;

    IF platform_id IS NULL THEN
        RAISE EXCEPTION '<Platform> platform not found';
    END IF;

    -- Resolve parent feature (if nesting under existing feature)
    SELECT id INTO parent_feature_id
    FROM common.features
    WHERE platform_id = platform_id
      AND code = '<parent_code>'
      AND deleted_at IS NULL;

    -- Create the feature
    INSERT INTO common.features (platform_id, icon, code, parent_id, sort_order, behavior_type, is_active)
    VALUES (platform_id, '<Icon>', '<feature-code>', parent_feature_id, <N>, '<behavior_type>', true)
    ON CONFLICT (platform_id, code) DO UPDATE
        SET icon          = EXCLUDED.icon,
            parent_id     = EXCLUDED.parent_id,
            sort_order    = EXCLUDED.sort_order,
            behavior_type = EXCLUDED.behavior_type,
            is_active     = EXCLUDED.is_active,
            deleted_at    = NULL;

    -- Resolve the feature ID
    SELECT id INTO new_feature_id
    FROM common.features
    WHERE platform_id = platform_id
      AND code = '<feature-code>'
      AND deleted_at IS NULL;

    -- Add feature translations (en + id)
    INSERT INTO common.feature_translations (feature_id, locale, name, description)
    VALUES
        (new_feature_id, 'en', '<English Name>', '<English description>'),
        (new_feature_id, 'id', '<Indonesian Name>', '<Indonesian description>')
    ON CONFLICT (feature_id, locale) DO UPDATE
        SET name = EXCLUDED.name,
            description = EXCLUDED.description;

    -- Bind permissions to the feature
    INSERT INTO common.permission_features (permission_id, feature_id, created_at)
    SELECT p.id, new_feature_id, NOW()
    FROM common.permissions p
    WHERE p.resource = '<resource>'
      AND p.deleted_at IS NULL
      AND NOT EXISTS (
          SELECT 1
          FROM common.permission_features pf
          WHERE pf.permission_id = p.id
            AND pf.feature_id = new_feature_id
            AND pf.deleted_at IS NULL
      )
    ON CONFLICT (permission_id, feature_id) DO UPDATE
    SET deleted_at = NULL;
END $$;
```

**Feature behavior types:**
- `built_in` — always available to all organizations (e.g., billing, roles)
- `standard` — requires explicit enablement per organization (e.g., FTM, sites)

### Step 4: Bind Permissions to Existing Feature (if feature exists)

When adding permissions to an **already existing** feature, resolve the feature and bind:

```sql
DO $$
DECLARE
    target_feature_id UUID;
BEGIN
    SELECT id INTO target_feature_id
    FROM common.features
    WHERE code = '<feature-code>'
      AND platform_id = (SELECT id FROM common.platforms WHERE code = '<platform>' AND deleted_at IS NULL)
      AND deleted_at IS NULL;

    IF target_feature_id IS NULL THEN
        RAISE EXCEPTION '<Feature> feature not found';
    END IF;

    INSERT INTO common.permission_features (permission_id, feature_id, created_at)
    SELECT p.id, target_feature_id, NOW()
    FROM common.permissions p
    WHERE p.resource = '<resource>'
      AND p.deleted_at IS NULL
      AND NOT EXISTS (
          SELECT 1
          FROM common.permission_features pf
          WHERE pf.permission_id = p.id
            AND pf.feature_id = target_feature_id
            AND pf.deleted_at IS NULL
      )
    ON CONFLICT (permission_id, feature_id) DO UPDATE
    SET deleted_at = NULL;
END $$;
```

### Step 5: Insert Permission Dependencies (if applicable)

When one permission logically requires another (e.g., `show` requires `list`):

```sql
INSERT INTO common.permission_dependencies (permission_id, dependency_permission_id)
SELECT
    show_perm.id,
    list_perm.id
FROM common.permissions show_perm
CROSS JOIN common.permissions list_perm
WHERE show_perm.action = 'show'
  AND show_perm.resource = '<resource>'
  AND show_perm.deleted_at IS NULL
  AND list_perm.action = 'list'
  AND list_perm.resource = '<resource>'
  AND list_perm.deleted_at IS NULL
ON CONFLICT (permission_id, dependency_permission_id) DO NOTHING;
```

**Common dependency chains:**
- `show` → `list`
- `update` → `show`
- `delete` → `show`
- `create` → `list`

## Down Migration

Reverse the up migration using **soft deletes** in reverse order (children before parents):

```sql
-- 1) Soft delete permission-feature bindings
UPDATE common.permission_features
SET deleted_at = CURRENT_TIMESTAMP
WHERE permission_id IN (
    SELECT id FROM common.permissions WHERE resource = '<resource>'
)
AND deleted_at IS NULL;

-- 2) Soft delete permission translations
UPDATE common.permission_translations
SET deleted_at = CURRENT_TIMESTAMP
WHERE permission_id IN (
    SELECT id FROM common.permissions WHERE resource = '<resource>'
)
AND deleted_at IS NULL;

-- 3) Soft delete permissions
UPDATE common.permissions
SET deleted_at = CURRENT_TIMESTAMP
WHERE resource = '<resource>'
  AND deleted_at IS NULL;

-- 4) Soft delete feature translations (only if feature was created in the up)
UPDATE common.feature_translations
SET deleted_at = CURRENT_TIMESTAMP
WHERE feature_id IN (
    SELECT f.id
    FROM common.features f
    JOIN common.platforms pl ON pl.id = f.platform_id AND pl.code = '<platform>'
    WHERE f.code = '<feature-code>'
      AND f.deleted_at IS NULL
)
AND deleted_at IS NULL;

-- 5) Soft delete feature (only if feature was created in the up)
UPDATE common.features
SET deleted_at = CURRENT_TIMESTAMP
WHERE code = '<feature-code>'
  AND platform_id = (SELECT id FROM common.platforms WHERE code = '<platform>' AND deleted_at IS NULL)
  AND deleted_at IS NULL;
```

## Resource Naming Convention

The `resource` column uses **camelCase** matching the domain entity:

| Endpoint Pattern | Resource |
|-----------------|----------|
| `/ftm/fuel-tank-monitoring-devices` | `fuelTankMonitoringDevices` |
| `/billing/bills` | `bills` |
| `/billing/admin/payment-proofs` | `adminPaymentProofs` |
| `/common/roles` | `roles` |
| `/operation/sites` | `sites` |

- Prefix with `admin` for admin-portal-specific resources (e.g., `adminInvoices`, `adminPaymentProofs`)
- Use the plural noun form (e.g., `bills`, not `bill`)

## Action Naming Convention

| CRUD Operation | Action | Endpoint Pattern |
|---------------|--------|-----------------|
| List all | `list` | `GET /resource` |
| Get one | `show` | `GET /resource/{id}` |
| Create | `create` | `POST /resource` |
| Update | `update` | `PUT /resource/{id}` |
| Delete | `delete` | `DELETE /resource/{id}` |
| Toggle active | `set_is_active` | `PUT /resource/{id}/set-is-active` |
| Custom action | descriptive verb | `POST /resource/{id}/action` |

## Endpoint Path Convention

The `endpoint_path` must match the **Kong route path** exactly:
- Use `{id}` for path parameters (not `:id` or `*`)
- Include the full service prefix (e.g., `/ftm/`, `/billing/`, `/common/`)
- Do not include query parameters

### String Enum Path Parameters

`NormalizeEndpoint()` only normalizes UUID and numeric segments to `{id}`. String enum values (e.g., platform codes "admin", "my", "app") are **NOT normalized** — they pass through as literal strings. This means endpoints with string enum path parameters need **one permission row per enum value**.

Example: `GET /common/platforms/{platformCode}/access` with platform codes `admin`, `my`, `app` requires 3 permission rows:

```sql
('/common/platforms/admin/access', 'GET', 'check_admin', 'platformAccess', true, false),
('/common/platforms/my/access',    'GET', 'check_my',    'platformAccess', true, false),
('/common/platforms/app/access',   'GET', 'check_app',   'platformAccess', true, false)
```

Use the same `resource` with different `action` values to satisfy the `(action, resource)` unique constraint while grouping them under one resource for translations.

## Step 6: Assign to Built-in Role (Universal Permissions)

Some permissions are **universal** — every user needs them regardless of their role or organization. These permissions should be:
- Assigned to the **"User" built-in role** (auto-assigned to all users)
- Set with `hide = true` (not configurable per role in the UI)

Use the CROSS JOIN pattern from migration `000135_seed_universal_user_role.up.sql`:

```sql
INSERT INTO common.role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM common.permissions p
CROSS JOIN (
    SELECT id FROM common.roles
    WHERE name = 'User' AND type = 'built-in' AND deleted_at IS NULL
    LIMIT 1
) r
WHERE p.resource = '<resource>'
  AND p.deleted_at IS NULL
ON CONFLICT (role_id, permission_id) DO NOTHING;
```

**When to use this step:**
- Endpoint is something every user needs (e.g., checking their own platform access, managing their notifications)
- The permission is NOT configurable — it should not appear in the role management UI
- There is no feature toggle gate — the endpoint is always available

**Down migration for role_permissions:** Use hard DELETE (no `deleted_at` on this table):
```sql
DELETE FROM common.role_permissions
WHERE permission_id IN (
    SELECT id FROM common.permissions WHERE resource = '<resource>'
);
```

See `.claude/rules/database/roles.md` for details on the built-in roles.

## Checklist

When adding a new authenticated endpoint, verify:

- [ ] Permission row inserted in `common.permissions` with correct `endpoint_path`, `endpoint_action`, `action`, `resource`
- [ ] Translations added in both `en` and `id` locales
- [ ] Permission bound to a feature via `common.permission_features` (or explicitly skipped if universal — see Step 6)
- [ ] Permission dependencies inserted if applicable (e.g., `show` → `list`)
- [ ] Down migration reverses all changes with soft deletes (hard DELETE for `role_permissions`)
- [ ] `endpoint_path` matches the Kong route path exactly (including full service prefix)
- [ ] `is_admin` set correctly (`true` for admin-only, `false` for user-facing)
- [ ] If universal: assigned to "User" built-in role via `role_permissions` (Step 6) and `hide = true`
- [ ] If string enum path parameter: one permission row per enum value (see "String Enum Path Parameters")
