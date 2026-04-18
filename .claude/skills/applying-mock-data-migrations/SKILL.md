---
name: applying-mock-data-migrations
description: Apply mock data migrations to test new test/development data. Use proactively after creating mock data in infra/database/cmd/seed/mock-data/.
---

# Apply Mock Data Migrations

For mock data file conventions (YAML structure, UUID ranges, directory layout), see `.claude/rules/infra/naming.md`.

**Environment:** Development/testing only — never production.

## Prerequisites

Schema migrations must be applied first:
```bash
cd infra/database
make up
```

Seed data is part of SQL migrations (no separate `make seed` step needed).

## Commands

All commands run from `infra/database/`:

```bash
cd infra/database

# Apply mock data + crons (most common)
make seed-mock

# Apply mock data only (no crons)
make seed-mock-data

# Force apply (upsert on conflict, skips dirty state checks)
make seed-mock-force

# Rollback one mock data migration
make rollback-mock-data

# Rollback ALL mock data (interactive confirmation)
make rollback-mock-data-all

# Rollback mock data + crons
make rollback-mock

# Reset ALL mock data from scratch (CI-friendly, no prompts)
make reset-mock

# Full dev setup from scratch: migrations + mock data + crons
make dev-setup
```

### Cron-Only Commands

```bash
make cron-up       # Apply cron migrations
make cron-down     # Rollback cron migrations
make cron-force    # Force apply cron migrations
```

## Testing Workflow

```bash
cd infra/database

# 1. Apply mock data
make seed-mock-data

# 2. Verify test data in database

# 3. Test application with mock data

# 4. Test rollback
make rollback-mock-data

# 5. Verify cleanup

# 6. Reapply
make seed-mock-data
```

## Mock vs Seed Data

| Aspect | Seed Data | Mock Data |
|--------|-----------|-----------|
| Location | `infra/database/migrations/` (SQL) | `infra/database/cmd/seed/mock-data/` (YAML) |
| Format | SQL with `initial_data_` or `seed_` prefix | YAML with `create`/`remove`/`execute` ops |
| Environment | All (dev/staging/prod) | Dev/testing only |
| Examples | Permissions, roles, email templates | Test users, sample orgs, test devices |
| Persistence | Permanent | Disposable |

## Troubleshooting

**Foreign key constraint** — ensure schema migrations applied first (`make up`). Check that referenced records exist (parent org before org users).

**Duplicate key** — migration was already applied. Rollback first:
```bash
make rollback-mock-data
make seed-mock-data
```

**YAML parse error** — check indentation (2 spaces, no tabs), validate structure.

**Reset everything** — nuclear option for a clean slate:
```bash
make reset-mock
```
