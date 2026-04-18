---
paths:
  - "infra/**"
  - "**/docker/**"
  - "**/Dockerfile"
  - "docker-compose*.yaml"
---

# Infrastructure Naming

## Migration Files

Location: `infra/database/migrations/`

### File Naming

Format: `{6-digit-sequence}_{description}.{direction}.sql`

- Sequence: zero-padded, auto-incremented (`000001`, `000002`, ...)
- Direction: `.up.sql` and `.down.sql` (always create both)
- Description: snake_case, starts with the action
- To find the next sequence number, check the latest file in the migrations directory

**Schema migrations** (DDL):
```
000204_create_table_common_files.up.sql
000047_create_table_organization_sites_and_device_sites.up.sql
000243_alter_table_billing_payments_add_payment_method_fields.up.sql
000223_fix_ftm_permission_resource_translations.up.sql
```

**Seed/data migrations** (DML) — use `initial_data_` or `seed_` prefix:
```
000078_initial_data_insert_device_types.up.sql
000149_seed_common_features.up.sql
000155_seed_organization_owner_role.up.sql
000230_initial_data_insert_invoice_sent_and_overdue_email_template.up.sql
```

### Description Prefixes

| Prefix | Use For |
|--------|---------|
| `create_table_{schema}_{table}` | New table |
| `alter_table_{table}_{change}` | Modify table structure |
| `add_{column}_to_{table}` | Add column |
| `add_{feature}_permissions` | Permission inserts |
| `initial_data_insert_{entity}` | Seed data (simple inserts) |
| `seed_{entity}` | Seed data (complex with PL/pgSQL) |
| `fix_{entity}_{issue}` | Data or schema fix |
| `drop_{table_or_column}` | Remove schema elements |
| `rename_{old}_to_{new}` | Rename columns or tables |
| `migrate_{entity}_{change}` | Data transformation |

### SQL Conventions

**Header comment** — every migration starts with:
```sql
-- Description of what this migration does
-- Migration {filename}
```

**Schema qualification** — always qualify table names with their schema:
```sql
CREATE TABLE common.files (...)       -- not just: files
INSERT INTO common.permissions (...)  -- not just: permissions
```

**Safe operations** — use `IF EXISTS` / `IF NOT EXISTS` where appropriate:
```sql
CREATE TABLE IF NOT EXISTS common.files (...);
DROP TABLE IF EXISTS common.files;
DROP INDEX IF EXISTS idx_files_purpose;
ALTER TABLE common.files ADD COLUMN IF NOT EXISTS status varchar(50);
```

**Idempotency for data** — use `ON CONFLICT` for insert migrations:
```sql
INSERT INTO common.device_types (name, description) VALUES
    ('fuel-tank-monitoring', 'Fuel tank monitoring devices')
ON CONFLICT (name) DO NOTHING;
```

### Schema Migration Content — Standard Table Pattern

```sql
-- Create files table — central registry for all files
-- Migration 000204_create_table_common_files.up.sql

CREATE TABLE common.files (
    id uuid DEFAULT gen_random_uuid() PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES common.organizations(id),
    name varchar(255) NOT NULL,
    purpose varchar(100) NOT NULL,
    -- ... domain columns ...
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    deleted_at timestamp without time zone
);

-- Indexes on foreign keys and query columns
CREATE INDEX idx_files_organization_id ON common.files(organization_id);
CREATE INDEX idx_files_purpose ON common.files(purpose);

-- Updated_at trigger (required for all tables with updated_at)
CREATE TRIGGER update_files_updated_at
BEFORE UPDATE ON common.files
FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
```

Standard columns for most tables:
- `id uuid DEFAULT gen_random_uuid() PRIMARY KEY`
- `organization_id uuid NOT NULL REFERENCES common.organizations(id)` (for tenant-scoped tables)
- `created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL`
- `updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL`
- `deleted_at timestamp without time zone` (soft delete, nullable)

### Seed Data Content

Two styles based on complexity:

**Simple inserts** (use `initial_data_` prefix):
```sql
INSERT INTO common.device_types (name, description) VALUES
    ('fuel-tank-monitoring', 'Fuel tank monitoring devices')
ON CONFLICT (name) DO NOTHING;
```

**Complex seeding with PL/pgSQL** (use `seed_` prefix) — for multi-step logic that resolves IDs:
```sql
DO $$
DECLARE
    admin_platform_id UUID;
    feature_id UUID;
BEGIN
    SELECT id INTO admin_platform_id
    FROM common.platforms
    WHERE code = 'admin' AND deleted_at IS NULL;

    IF admin_platform_id IS NULL THEN
        RAISE EXCEPTION 'Admin platform not found';
    END IF;

    -- Insert feature
    INSERT INTO common.features (code, name, platform_id, parent_id)
    VALUES ('billing', 'Billing', admin_platform_id, NULL)
    ON CONFLICT (code, platform_id) WHERE deleted_at IS NULL DO NOTHING
    RETURNING id INTO feature_id;

    -- Insert permissions linked to the feature
    INSERT INTO common.permission_features (permission_id, feature_id)
    SELECT p.id, feature_id
    FROM common.permissions p
    WHERE p.resource = 'billing' AND p.deleted_at IS NULL;
END $$;
```

For locale-specific data, insert both `en` and `id` rows.

### Seed Data ID Ranges

Use consistent UUID ranges so seed data is identifiable and doesn't collide:

| Range Prefix | Entity |
|-------------|--------|
| `40000000-0000-4000-8000-...` | Permissions |
| `50000000-0000-4000-8000-...` | Roles |
| `30000000-0000-4000-8000-...` | Email templates |
| `60000000-0000-4000-8000-...` | System config |

### Down Migrations

Symmetric inverse of the up migration — must perfectly undo what the up migration did:

- `CREATE TABLE` → `DROP TABLE IF EXISTS`
- `ADD COLUMN` → `ALTER TABLE ... DROP COLUMN IF EXISTS`
- `CREATE INDEX` → `DROP INDEX IF EXISTS`
- `CREATE TRIGGER` → `DROP TRIGGER IF EXISTS`
- `INSERT` → `DELETE FROM ... WHERE` (match exact rows)
- For PL/pgSQL up migrations, the down should reverse the logic in opposite order (children before parents)

## Mock Data

Location: `infra/database/cmd/seed/mock-data/` — YAML format, separate Go module, never mixed with production migrations.

### Directory Structure

```
infra/database/cmd/seed/mock-data/
├── _global/                         # Shared global data (platforms, device types, users)
│   ├── 0001_platforms.yaml
│   ├── 0002_platform_translations.yaml
│   ├── 0003_device_types.yaml
│   ├── 0012_users.yaml
│   └── ...
├── industrix-corp/                  # Organization-specific data
│   ├── 0001_organization.yaml
│   ├── 0002_organization_users.yaml
│   ├── 0003_roles.yaml
│   └── ...
├── nusantara-agro/                  # Another organization
│   └── ...
```

### File Naming

Format: `{4-digit-sequence}_{description}.yaml`

- Sequence: 4-digit, zero-padded (`0001`, `0002`, ...)
- `_global/` for data shared across all organizations
- Organization-named directories for tenant-specific data

### Mock Data UUID Ranges

| Range Prefix | Entity |
|-------------|--------|
| `10000000-0000-4000-8000-...` | Organizations |
| `20000000-0000-4000-8000-...` | Users |
| `30000000-0000-4000-8000-...` | Templates |
| `40000000-0000-4000-8000-...` | Permissions (seed, not mock) |

### YAML Structure

```yaml
# Optional: field transforms (e.g., password hashing)
transforms:
  common.users:
    password: "hash_password"

up:
  - create:
      common.organizations:
        - id: "10000000-0000-4000-8000-000000000001"
          name: "Industrix Corp"
          is_active: true
          locale: "id"
          timezone: "Asia/Jakarta"

down:
  - execute:
      sql:
        - sql: "SET LOCAL session_replication_role = 'replica'"
  - remove:
      common.organizations:
        - id: "10000000-0000-4000-8000-000000000001"
```

**Operations:**
- `create:` — insert rows into a table
- `remove:` — delete rows by ID
- `execute:` — run raw SQL (e.g., disable replication role for cascading deletes)
- `transforms:` — apply field-level transforms before insert (e.g., `hash_password`)
