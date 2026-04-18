---
paths:
  - "infra/database/migrations/**"
---

# Database Migration Conventions

For file naming conventions (sequence numbers, description prefixes, up/down), see `.claude/rules/infra/naming.md`.

## Schemas

All tables belong to a schema. Never create tables in the `public` schema.

| Schema | Domain | Examples |
|--------|--------|---------|
| `common` | Users, orgs, devices, permissions, roles, features | `common.users`, `common.organizations` |
| `ftm` | Fuel tank monitoring | `ftm.ftm_processes`, `ftm.ftm_quotas` |
| `operation` | Sites, operational data | `operation.sites`, `operation.site_categories` |
| `billing` | Invoices, payments, subscriptions | `billing.invoices`, `billing.payments` |
| `billing_archive` | Archived billing data | `billing_archive.*` |

Create new schemas with `IF NOT EXISTS`:
```sql
CREATE SCHEMA IF NOT EXISTS billing;
```

## Standard Table Template

```sql
CREATE TABLE {schema}.{table_name} (
    id uuid DEFAULT gen_random_uuid() PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES common.organizations(id),
    -- ... domain columns ...
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    deleted_at timestamp without time zone,
    deleted_by uuid REFERENCES common.users(id) ON DELETE SET NULL
);

-- Indexes
CREATE INDEX idx_{table}_{column} ON {schema}.{table_name}({column});
CREATE INDEX idx_{table}_organization_id ON {schema}.{table_name}(organization_id);
CREATE INDEX idx_{table}_deleted_at ON {schema}.{table_name}(deleted_at) WHERE deleted_at IS NULL;

-- Updated_at trigger (required for all tables with updated_at)
CREATE TRIGGER update_{table}_updated_at
    BEFORE UPDATE ON {schema}.{table_name}
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
```

## Column Types

| Use Case | Type | Example |
|----------|------|---------|
| Primary keys | `uuid DEFAULT gen_random_uuid()` | `id uuid DEFAULT gen_random_uuid() PRIMARY KEY` |
| Foreign keys | `uuid` | `organization_id uuid NOT NULL REFERENCES common.organizations(id)` |
| Short strings | `character varying(N)` | `name character varying(255) NOT NULL` |
| Long content | `text` | `body_template text NOT NULL` |
| Enum-like values | `character varying(N)` + CHECK | `status character varying(20) NOT NULL CHECK (status IN ('active', 'inactive'))` |
| Boolean flags | `boolean NOT NULL DEFAULT {value}` | `is_active boolean NOT NULL DEFAULT true` |
| Whole numbers | `integer` | `quota_limit integer NOT NULL DEFAULT 0` |
| Currency/decimal | `numeric(15,2)` | `amount numeric(15,2) NOT NULL CHECK (amount >= 0)` |
| Timestamps | `timestamp without time zone` | `created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL` |
| JSON data | `jsonb` | `config jsonb NOT NULL DEFAULT '{}'::jsonb` |

Prefer `character varying(N)` + `CHECK` over PostgreSQL `CREATE TYPE ... AS ENUM` — CHECK constraints are easier to modify in later migrations.

## Naming Conventions

### Indexes

Pattern: `idx_{table}_{column}`

```sql
CREATE INDEX idx_billing_payments_invoice_status ON billing.payments(invoice_id, status);
CREATE INDEX idx_billing_payments_org_date ON billing.payments(organization_id, payment_date);
CREATE INDEX idx_billing_payments_deleted_at ON billing.payments(deleted_at) WHERE deleted_at IS NULL;
```

### Foreign Keys

Pattern: `fk_{table}_{column}`

```sql
ALTER TABLE common.user_access_cards
    ADD CONSTRAINT fk_user_access_cards_user_id
    FOREIGN KEY (user_id) REFERENCES common.users(id) ON DELETE CASCADE;
```

### Unique Constraints

Pattern: `uk_{table}_{column}` or `uq_{table}_{column}`

```sql
CONSTRAINT uk_scheduled_jobs_name UNIQUE (name)
CONSTRAINT uk_scheduled_job_runs_job_time UNIQUE (job_id, scheduled_at)
```

### Check Constraints

Pattern: `chk_{table}_{constraint}`

```sql
CONSTRAINT chk_background_processes_status CHECK (status IN ('pending', 'processing', 'completed', 'failed', 'cancelled')),
CONSTRAINT chk_background_processes_progress CHECK (progress >= 0 AND progress <= 100)
```

### Triggers

Standard: `update_{table}_updated_at` for timestamp triggers.
Business logic: `trg_{descriptive_name}` for custom triggers.

```sql
-- Timestamp trigger
CREATE TRIGGER update_billing_payments_updated_at
    BEFORE UPDATE ON billing.payments
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Business logic trigger
CREATE TRIGGER trg_prevent_last_org_owner_removal
    BEFORE UPDATE OR DELETE ON common.organization_users
    FOR EACH ROW
    EXECUTE FUNCTION prevent_last_org_owner_removal();
```

## Foreign Key ON DELETE Behavior

| Behavior | When to Use | Example |
|----------|-------------|---------|
| `CASCADE` | Deleting parent should delete children | `REFERENCES common.devices(id) ON DELETE CASCADE` |
| `SET NULL` | Optional relationship, keep child but clear reference | `deleted_by uuid REFERENCES common.users(id) ON DELETE SET NULL` |
| `RESTRICT` | Prevent parent deletion while children exist | `REFERENCES billing.invoices(id) ON DELETE RESTRICT` |

Default: inline `REFERENCES` in `CREATE TABLE` (preferred). Use separate `ALTER TABLE ADD CONSTRAINT` only when modifying existing tables.

## Soft Delete

Every table that supports deletion must have:

```sql
deleted_at timestamp without time zone,
deleted_by uuid REFERENCES common.users(id) ON DELETE SET NULL
```

Plus a **partial index** for efficient "active records" queries:

```sql
CREATE INDEX idx_{table}_deleted_at ON {schema}.{table}(deleted_at) WHERE deleted_at IS NULL;
```

Tables that do NOT use soft delete (hard delete only): join tables, logs, audit trails.

## Multi-Step Migrations

DDL and DML can be combined in a single migration file. PostgreSQL wraps each migration file in a transaction automatically.

Common pattern — add column, backfill, then make NOT NULL:

```sql
-- Step 1: Add column (nullable)
ALTER TABLE common.devices ADD COLUMN timezone VARCHAR(255);

-- Step 2: Backfill existing rows
UPDATE common.devices d
SET timezone = COALESCE(o.timezone, 'Asia/Jakarta')
FROM common.organizations o
WHERE d.organization_id = o.id AND d.timezone IS NULL;

-- Step 3: Make NOT NULL
ALTER TABLE common.devices ALTER COLUMN timezone SET NOT NULL;
```

## Operational Rules

- **Never modify a deployed migration** — if it's been applied to staging/production, create a new migration to fix it
- **Always create both `.up.sql` and `.down.sql`** — the down migration must perfectly reverse the up
- **Use `IF EXISTS` / `IF NOT EXISTS`** for safety in DDL statements
- **Use `ON CONFLICT` for idempotent data inserts** — seed data may be re-applied
- **Always qualify table names with schema** — `common.users`, not `users`
- **Test both directions** — apply up, verify, rollback down, verify, reapply up
- **Order matters for foreign keys** — create parent tables before child tables in up, drop children before parents in down
